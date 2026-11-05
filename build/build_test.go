package build_test

import (
	"bytes"
	"encoding/binary"
	"testing"
	"testing/fstest"
	"unicode/utf16"

	"github.com/google/go-cmp/cmp"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/build"
	"github.com/xaroth/pbt-extract/dat"
)

const schemaJSON = `{"version": 7, "tables": [
	{"name": "Stats", "columns": [{"name": "Id", "type": "string"}]},
	{"name": "Characters", "columns": [
		{"name": "Id", "type": "string"}, {"name": "Name", "type": "string"},
		{"name": "BaseStrength", "type": "i32"}, {"name": "BaseDexterity", "type": "i32"},
		{"name": "BaseIntelligence", "type": "i32"}, {"name": "BaseMaxLife", "type": "i32"},
		{"name": "BaseMaxMana", "type": "i32"}]},
	{"name": "Unused", "columns": []}
]}`

// datc64 encodes rows whose cells are strings or int32s.
func datc64(rows ...[]any) []byte {
	vars := bytes.Repeat([]byte{0xbb}, 8)
	fixed := binary.LittleEndian.AppendUint32(nil, uint32(len(rows)))
	for _, r := range rows {
		for _, v := range r {
			switch v := v.(type) {
			case string:
				fixed = binary.LittleEndian.AppendUint64(fixed, uint64(len(vars)))
				for _, u := range utf16.Encode([]rune(v)) {
					vars = binary.LittleEndian.AppendUint16(vars, u)
				}
				vars = append(vars, 0, 0, 0, 0)
			case int32:
				fixed = binary.LittleEndian.AppendUint32(fixed, uint32(v))
			}
		}
	}
	return append(fixed, vars...)
}

func ot(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return b
}

func schema(t *testing.T) *dat.Schema {
	t.Helper()
	s, err := dat.ParseSchema([]byte(schemaJSON))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func game() fstest.MapFS {
	return fstest.MapFS{
		"Data/Stats.datc64": {Data: datc64([]any{"level"}, []any{"life_per_level"}, []any{"base_speed"})},
		"Data/Characters.datc64": {Data: datc64(
			[]any{"Metadata/Characters/Str/Str", "Marauder", int32(32), int32(14), int32(14), int32(38), int32(34)},
		)},
		"Metadata/Characters/Character.ot": {Data: ot("extends \"nothing\"\nStats\n{\n\tlife_per_level = 12\n\tlevel = 1\n}\n")},
		"Metadata/Characters/Str/Str.ot":   {Data: ot("extends \"Metadata/Characters/Character\"\nStats\n{\n}\n")},
	}
}

func TestDataset(t *testing.T) {
	t.Parallel()
	got, err := build.Dataset(game(), schema(t))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	want := &dataset.Dataset{
		Stats: []dataset.Stat{{ID: "level"}, {ID: "life_per_level"}, {ID: "base_speed"}},
		Characters: []dataset.Character{{
			ID: "Metadata/Characters/Str/Str", Name: "Marauder",
			BaseStrength: 32, BaseDexterity: 14, BaseIntelligence: 14, BaseMaxLife: 38, BaseMaxMana: 34,
			Stats: []dataset.StatValue{{Stat: 0, Value: 1}, {Stat: 1, Value: 12}},
		}},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Dataset mismatch (-want +got):\n%s", diff)
	}
}

func TestDatasetErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		change func(fstest.MapFS)
		schema string
	}{
		{name: "unknown stat", change: func(g fstest.MapFS) {
			g["Metadata/Characters/Str/Str.ot"] = &fstest.MapFile{Data: ot("extends \"nothing\"\nStats\n{\n\tnew_stat = 1\n}\n")}
		}},
		{name: "stat overflows int32", change: func(g fstest.MapFS) {
			g["Metadata/Characters/Str/Str.ot"] = &fstest.MapFile{Data: ot("extends \"nothing\"\nStats\n{\n\tlevel = 4294967296\n}\n")}
		}},
		{name: "missing ot", change: func(g fstest.MapFS) { delete(g, "Metadata/Characters/Str/Str.ot") }},
		{name: "missing table file", change: func(g fstest.MapFS) { delete(g, "Data/Characters.datc64") }},
		{name: "row size mismatch", change: func(g fstest.MapFS) {
			g["Data/Stats.datc64"] = &fstest.MapFile{Data: datc64([]any{"level", int32(1)})}
		}},
		{name: "table not in schema", schema: `{"version": 7, "tables": []}`},
		{name: "column not in schema", schema: `{"version": 7, "tables": [{"name": "Stats", "columns": [{"name": "Other", "type": "string"}]}]}`},
		{name: "column of wrong type", schema: `{"version": 7, "tables": [{"name": "Stats", "columns": [{"name": "Id", "type": "i32"}]}]}`, change: func(g fstest.MapFS) {
			g["Data/Stats.datc64"] = &fstest.MapFile{Data: datc64([]any{int32(1)})}
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := game()
			if testCase.change != nil {
				testCase.change(g)
			}
			s := schema(t)
			if testCase.schema != "" {
				var err error
				if s, err = dat.ParseSchema([]byte(testCase.schema)); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := build.Dataset(g, s); err == nil {
				t.Errorf("Dataset: got nil error, want an error")
			}
		})
	}
}
