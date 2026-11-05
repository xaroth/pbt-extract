package build_test

import (
	"bytes"
	"encoding/binary"
	"testing"
	"testing/fstest"
	"unicode/utf16"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/build"
	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/tree"
)

const schemaJSON = `{"version": 7, "tables": [
	{"name": "Stats", "columns": [{"name": "Id", "type": "string"}]},
	{"name": "Characters", "columns": [
		{"name": "Id", "type": "string"}, {"name": "Name", "type": "string"},
		{"name": "BaseStrength", "type": "i32"}, {"name": "BaseDexterity", "type": "i32"},
		{"name": "BaseIntelligence", "type": "i32"}, {"name": "BaseMaxLife", "type": "i32"},
		{"name": "BaseMaxMana", "type": "i32"}]},
	{"name": "PassiveSkills", "columns": [
		{"name": "PassiveSkillGraphId", "type": "u16"},
		{"name": "Stats", "type": "foreignrow", "array": true},
		{"name": "Stat1Value", "type": "i32"}, {"name": "Stat2Value", "type": "i32"},
		{"name": "Stat3Value", "type": "i32"}, {"name": "Stat4Value", "type": "i32"},
		{"name": "Stat5Value", "type": "i32"},
		{"name": "PassiveSkillBuffs", "type": "foreignrow", "array": true},
		{"name": "SkillPointsGranted", "type": "i32"},
		{"name": "HasHardmodeAlternate", "type": "bool"},
		{"name": "StatsHardmode", "type": "foreignrow", "array": true},
		{"name": "Stat1ValueHardmode", "type": "i32"}, {"name": "Stat2ValueHardmode", "type": "i32"},
		{"name": "Stat3ValueHardmode", "type": "i32"}, {"name": "Stat4ValueHardmode", "type": "i32"},
		{"name": "Stat5ValueHardmode", "type": "i32"},
		{"name": "BuffsHardmode", "type": "foreignrow", "array": true},
		{"name": "SkillPointsGrantedHardmode", "type": "i32"}]},
	{"name": "PassiveSkillMasteryEffects", "columns": [
		{"name": "HASH16", "type": "u16"},
		{"name": "Stats", "type": "foreignrow", "array": true},
		{"name": "Stat1Value", "type": "i32"}, {"name": "Stat2Value", "type": "i32"},
		{"name": "Stat3Value", "type": "i32"}]},
	{"name": "BuffTemplates", "columns": [
		{"name": "BuffDefinitionsKey", "type": "foreignrow"},
		{"name": "Buff_StatValues", "type": "i32", "array": true}]},
	{"name": "BuffDefinitions", "columns": [{"name": "StatsKeys", "type": "foreignrow", "array": true}]}
]}`

// Cell types for datc64 beyond string and int32.
type (
	ref  int // foreignrow; -1 is null
	refs []int
	i32s []int32
)

// datc64 encodes rows of string, int32, uint16, bool, ref, refs and i32s cells.
func datc64(rows ...[]any) []byte {
	le := binary.LittleEndian
	vars := bytes.Repeat([]byte{0xbb}, 8)
	fixed := le.AppendUint32(nil, uint32(len(rows)))
	list := func(n int, elems []byte) {
		fixed = le.AppendUint64(fixed, uint64(n))
		fixed = le.AppendUint64(fixed, uint64(len(vars)))
		vars = append(vars, elems...)
	}
	foreign := func(b []byte, r int) []byte {
		if r < 0 {
			return append(b, bytes.Repeat([]byte{0xfe}, 16)...)
		}
		return le.AppendUint64(le.AppendUint64(b, uint64(r)), 0)
	}
	for _, r := range rows {
		for _, v := range r {
			switch v := v.(type) {
			case string:
				fixed = le.AppendUint64(fixed, uint64(len(vars)))
				for _, u := range utf16.Encode([]rune(v)) {
					vars = le.AppendUint16(vars, u)
				}
				vars = append(vars, 0, 0, 0, 0)
			case int32:
				fixed = le.AppendUint32(fixed, uint32(v))
			case uint16:
				fixed = le.AppendUint16(fixed, v)
			case bool:
				b := byte(0)
				if v {
					b = 1
				}
				fixed = append(fixed, b)
			case ref:
				fixed = foreign(fixed, int(v))
			case refs:
				var e []byte
				for _, x := range v {
					e = foreign(e, x)
				}
				list(len(v), e)
			case i32s:
				var e []byte
				for _, x := range v {
					e = le.AppendUint32(e, uint32(x))
				}
				list(len(v), e)
			}
		}
	}
	return append(fixed, vars...)
}

func utf16le(s string) []byte {
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

// passive is one PassiveSkills row: graph id, stats and values, buffs, points, then the
// hardmode flag, stats and values. Hardmode buffs and points are empty.
func passive(id uint16, stats refs, vals [5]int32, buffs refs, points int32, hard bool, hStats refs, hVals [5]int32) []any {
	row := []any{id, stats}
	for _, v := range vals {
		row = append(row, v)
	}
	row = append(row, buffs, points, hard, hStats)
	for _, v := range hVals {
		row = append(row, v)
	}
	return append(row, refs(nil), int32(0))
}

const sdDir = "Metadata/StatDescriptions/"

func game() fstest.MapFS {
	return fstest.MapFS{
		"Data/Stats.datc64": {Data: datc64(
			[]any{"level"}, []any{"life_per_level"}, []any{"base_speed"}, []any{"base_strength"}, []any{"aura_damage_+%"},
		)},
		"Data/Characters.datc64": {Data: datc64(
			[]any{"Metadata/Characters/Str/Str", "Marauder", int32(32), int32(14), int32(14), int32(38), int32(34)},
		)},
		"Data/PassiveSkills.datc64": {Data: datc64(
			passive(1, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
			passive(2, refs{3, 1}, [5]int32{10, 5}, refs{0, 1}, 1, true, refs{3}, [5]int32{20}),
			passive(3, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
		)},
		"Data/PassiveSkillMasteryEffects.datc64": {Data: datc64(
			[]any{uint16(7), refs{1}, int32(3), int32(0), int32(0)},
		)},
		"Data/BuffTemplates.datc64": {Data: datc64(
			[]any{ref(0), i32s{-5, 9}},
			[]any{ref(1), i32s{1}},
		)},
		"Data/BuffDefinitions.datc64": {Data: datc64(
			[]any{refs{4}},
			[]any{refs(nil)},
		)},
		"Metadata/Characters/Character.ot":                 {Data: utf16le("extends \"nothing\"\nStats\n{\n\tlife_per_level = 12\n\tlevel = 1\n}\n")},
		"Metadata/Characters/Str/Str.ot":                   {Data: utf16le("extends \"Metadata/Characters/Character\"\nStats\n{\n}\n")},
		sdDir + "passive_skill_aura_stat_descriptions.txt": {Data: utf16le("include \"" + sdDir + "passive_skill_stat_descriptions.txt\"\n")},
		sdDir + "passive_skill_stat_descriptions.txt":      {Data: utf16le("include \"" + sdDir + "stat_descriptions.txt\"\n")},
		sdDir + "stat_descriptions.txt":                    {Data: utf16le("description\n\t1 base_strength\n\t1\n\t\t# \"{0:+d} to Strength\"\n")},
	}
}

func export() *tree.Export {
	start := 0
	e := &tree.Export{
		Classes: []tree.Class{{
			Name: "Marauder", BaseStrength: 32, BaseDexterity: 14, BaseIntelligence: 14,
			Ascendancies: []tree.Ascendancy{{ID: "Juggernaut", Name: "Juggernaut"}},
		}},
		AlternateAscendancies: []tree.Ascendancy{{ID: "Bloodline", Name: "Bloodline"}},
		Nodes: map[string]tree.Node{
			"1": {Skill: 1, Name: "MARAUDER", ClassStartIndex: &start, Out: []string{"2"}},
			"2": {Skill: 2, Name: "Might", IsNotable: true, AscendancyName: "Juggernaut", In: []string{"1"}, Out: []string{"1"}},
			"3": {Skill: 3, Name: "Mastery", IsMastery: true, MasteryEffects: []tree.MasteryEffect{{Effect: 7}}},
		},
		JewelSlots: []uint32{2},
	}
	e.Points.Total, e.Points.Ascendancy = 123, 8
	return e
}

func input(t *testing.T, g fstest.MapFS, e *tree.Export) build.Input {
	t.Helper()
	return build.Input{
		Game:   g,
		Schema: schema(t),
		Trees: []build.Variant{
			{Name: "standard", Export: e},
			{Name: "ruthless", Export: e, Hardmode: true},
		},
		Constants: []dataset.Constant{{ID: "life_per_strength", Value: 1, Per: 2}},
	}
}

func TestDataset(t *testing.T) {
	t.Parallel()
	got, err := build.Dataset(input(t, game(), export()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	baseTree := func(variant string, n2 dataset.Node) dataset.Tree {
		return dataset.Tree{
			Variant: variant, TotalPoints: 123, AscendancyPoints: 8,
			Classes: []dataset.Class{{
				Name: "Marauder", BaseStrength: 32, BaseDexterity: 14, BaseIntelligence: 14, StartNode: 1,
				Ascendancies: []dataset.Ascendancy{{ID: "Juggernaut", Name: "Juggernaut"}},
			}},
			AlternateAscendancies: []dataset.Ascendancy{{ID: "Bloodline", Name: "Bloodline"}},
			Nodes: []dataset.Node{
				{ID: 1, Name: "MARAUDER", Kind: dataset.ClassStart, Neighbours: []uint32{2}},
				n2,
				{ID: 3, Name: "Mastery", Kind: dataset.Mastery, MasteryEffects: []dataset.MasteryEffect{{ID: 7, Stats: []dataset.StatValue{{Stat: 1, Value: 3}}}}},
			},
			JewelSlots: []uint32{2},
		}
	}
	might := dataset.Node{ID: 2, Name: "Might", Kind: dataset.Notable, Ascendancy: "Juggernaut", Neighbours: []uint32{1}}
	standard, ruthless := might, might
	standard.Stats = []dataset.StatValue{{Stat: 1, Value: 5}, {Stat: 3, Value: 10}}
	standard.Buffs = [][]dataset.StatValue{{{Stat: 4, Value: -5}}}
	standard.SkillPoints = 1
	ruthless.Stats = []dataset.StatValue{{Stat: 3, Value: 20}}

	want := &dataset.Dataset{
		Stats: []dataset.Stat{{ID: "level"}, {ID: "life_per_level"}, {ID: "base_speed"}, {ID: "base_strength"}, {ID: "aura_damage_+%"}},
		Characters: []dataset.Character{{
			ID: "Metadata/Characters/Str/Str", Name: "Marauder",
			BaseStrength: 32, BaseDexterity: 14, BaseIntelligence: 14, BaseMaxLife: 38, BaseMaxMana: 34,
			Stats: []dataset.StatValue{{Stat: 0, Value: 1}, {Stat: 1, Value: 12}},
		}},
		Constants: []dataset.Constant{{ID: "life_per_strength", Value: 1, Per: 2}},
		Trees:     []dataset.Tree{baseTree("standard", standard), baseTree("ruthless", ruthless)},
	}
	if diff := cmp.Diff(want, got, cmpopts.IgnoreFields(dataset.Dataset{}, "Descriptions"), cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Dataset mismatch (-want +got):\n%s", diff)
	}
	var scopes []string
	for _, s := range got.Descriptions {
		scopes = append(scopes, s.Name)
	}
	wantScopes := []string{build.DescriptionsRoot, sdDir + "passive_skill_stat_descriptions.txt", sdDir + "stat_descriptions.txt"}
	if diff := cmp.Diff(wantScopes, scopes); diff != "" {
		t.Errorf("description scopes mismatch (-want +got):\n%s", diff)
	}
}

func TestDatasetErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		game   func(fstest.MapFS)
		export func(*tree.Export)
		schema string
	}{
		{name: "unknown stat", game: func(g fstest.MapFS) {
			g["Metadata/Characters/Str/Str.ot"] = &fstest.MapFile{Data: utf16le("extends \"nothing\"\nStats\n{\n\tnew_stat = 1\n}\n")}
		}},
		{name: "stat overflows int32", game: func(g fstest.MapFS) {
			g["Metadata/Characters/Str/Str.ot"] = &fstest.MapFile{Data: utf16le("extends \"nothing\"\nStats\n{\n\tlevel = 4294967296\n}\n")}
		}},
		{name: "missing ot", game: func(g fstest.MapFS) { delete(g, "Metadata/Characters/Str/Str.ot") }},
		{name: "missing table file", game: func(g fstest.MapFS) { delete(g, "Data/Characters.datc64") }},
		{name: "missing descriptions", game: func(g fstest.MapFS) { delete(g, sdDir+"stat_descriptions.txt") }},
		{name: "row size mismatch", game: func(g fstest.MapFS) {
			g["Data/Stats.datc64"] = &fstest.MapFile{Data: datc64([]any{"level", int32(1)})}
		}},
		{name: "table not in schema", schema: `{"version": 7, "tables": []}`},
		{name: "column not in schema", schema: `{"version": 7, "tables": [{"name": "Stats", "columns": [{"name": "Other", "type": "string"}]}]}`},
		{name: "column of wrong type", schema: `{"version": 7, "tables": [{"name": "Stats", "columns": [{"name": "Id", "type": "i32"}]}]}`, game: func(g fstest.MapFS) {
			g["Data/Stats.datc64"] = &fstest.MapFile{Data: datc64([]any{int32(1)})}
		}},
		{name: "missing tree table", game: func(g fstest.MapFS) { delete(g, "Data/BuffDefinitions.datc64") }},
		{name: "repeated graph id", game: func(g fstest.MapFS) {
			g["Data/PassiveSkills.datc64"] = &fstest.MapFile{Data: datc64(
				passive(1, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
				passive(1, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
			)}
		}},
		{name: "repeated node stat", game: func(g fstest.MapFS) {
			g["Data/PassiveSkills.datc64"] = &fstest.MapFile{Data: datc64(
				passive(1, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
				passive(2, refs{3, 3}, [5]int32{1, 1}, nil, 0, false, nil, [5]int32{}),
				passive(3, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
			)}
		}},
		{name: "more stats than value columns", game: func(g fstest.MapFS) {
			g["Data/PassiveSkills.datc64"] = &fstest.MapFile{Data: datc64(
				passive(1, refs{0, 1, 2, 3, 4, 0}, [5]int32{}, nil, 0, false, nil, [5]int32{}),
				passive(2, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
				passive(3, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
			)}
		}},
		{name: "node stat out of range", game: func(g fstest.MapFS) {
			g["Data/PassiveSkills.datc64"] = &fstest.MapFile{Data: datc64(
				passive(1, refs{9}, [5]int32{1}, nil, 0, false, nil, [5]int32{}),
				passive(2, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
				passive(3, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
			)}
		}},
		{name: "buff with fewer values than stats", game: func(g fstest.MapFS) {
			g["Data/BuffDefinitions.datc64"] = &fstest.MapFile{Data: datc64([]any{refs{4, 4, 4}}, []any{refs(nil)})}
		}},
		{name: "buff without definition", game: func(g fstest.MapFS) {
			g["Data/BuffTemplates.datc64"] = &fstest.MapFile{Data: datc64([]any{ref(-1), i32s{1}}, []any{ref(1), i32s{1}})}
		}},
		{name: "buff stat out of range", game: func(g fstest.MapFS) {
			g["Data/BuffDefinitions.datc64"] = &fstest.MapFile{Data: datc64([]any{refs{9}}, []any{refs(nil)})}
		}},
		{name: "node not in PassiveSkills", export: func(e *tree.Export) { e.Nodes["9"] = tree.Node{Skill: 9} }},
		{name: "unknown mastery effect", export: func(e *tree.Export) {
			n := e.Nodes["3"]
			n.MasteryEffects = []tree.MasteryEffect{{Effect: 8}}
			e.Nodes["3"] = n
		}},
		{name: "edge to unknown node", export: func(e *tree.Export) {
			n := e.Nodes["2"]
			n.Out = []string{"9"}
			e.Nodes["2"] = n
		}},
		{name: "edge not a number", export: func(e *tree.Export) {
			n := e.Nodes["2"]
			n.Out = []string{"x"}
			e.Nodes["2"] = n
		}},
		{name: "class without start node", export: func(e *tree.Export) {
			e.Classes = append(e.Classes, tree.Class{Name: "Ranger"})
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g, e := game(), export()
			if testCase.game != nil {
				testCase.game(g)
			}
			if testCase.export != nil {
				testCase.export(e)
			}
			in := input(t, g, e)
			if testCase.schema != "" {
				var err error
				if in.Schema, err = dat.ParseSchema([]byte(testCase.schema)); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := build.Dataset(in); err == nil {
				t.Errorf("Dataset: got nil error, want an error")
			}
		})
	}
}
