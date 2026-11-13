package edition_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/edition"
)

func TestLoad(t *testing.T) {
	t.Parallel()
	e, err := edition.Load("poe1")
	if err != nil {
		t.Fatal(err)
	}
	if e.Name != "poe1" {
		t.Errorf("got name %q, want poe1", e.Name)
	}
	if len(e.Constants) == 0 {
		t.Error("got no constants, want the poe1 constants")
	}
	if len(e.Totems) == 0 {
		t.Error("got no totems, want the poe1 totems")
	}
	if _, err := edition.Load("nope"); err == nil {
		t.Error("Load(nope): got nil error, want an error")
	}
}

func TestNameColumns(t *testing.T) {
	t.Parallel()
	// QuestStaticRewards as dat-schema 717d788 defines it for poe1.
	const quests = `{"name": "QuestStaticRewards", "columns": [
		{"name": "QuestFlag", "type": "foreignrow"}, {"name": %s, "type": "i32"},
		{"name": "StatsKeys", "type": "foreignrow", "array": true}, {"name": "StatValues", "type": "i32", "array": true},
		{"name": "QuestKey", "type": "foreignrow"}, {"name": null, "type": "i32"},
		{"name": "ClientStringsKey", "type": "foreignrow"}, {"name": null, "type": "i32"},
		{"name": "StatValuesHardmode", "type": "i32", "array": true}, {"name": "ClientStringHardmode", "type": "foreignrow"},
		{"name": null, "type": "bool"}]}`
	testCases := []struct {
		name    string
		table   string
		wantErr bool
	}{
		{name: "unnamed", table: fmt.Sprintf(quests, "null")},
		{name: "named by dat-schema", table: fmt.Sprintf(quests, `"Points"`), wantErr: true},
		{name: "table missing", table: `{"name": "Other", "columns": []}`, wantErr: true},
	}
	e, err := edition.Load("poe1")
	if err != nil {
		t.Fatal(err)
	}
	// The quest columns alone, so the test schema needs no other table.
	names := slices.DeleteFunc(e.ColumnNames(), func(c dat.ColumnName) bool { return c.Table != "QuestStaticRewards" })
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			s, err := dat.ParseSchema([]byte(`{"version": 7, "tables": [` + testCase.table + `]}`))
			if err != nil {
				t.Fatal(err)
			}
			err = s.NameColumns(names)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("NameColumns: got error %v, want error %v", err, testCase.wantErr)
			}
			if testCase.wantErr {
				return
			}
			def, _ := s.Table("QuestStaticRewards")
			for name, want := range map[string]int{"PassivePoints": 1, "AscendancyPoints": 5, "RefundPoints": 7, "LeagueReward": 10} {
				if got, ok := def.Column(name); !ok || got != want {
					t.Errorf("Column(%q): got %d, %v, want %d, true", name, got, ok, want)
				}
			}
		})
	}
}

func TestValuesFor(t *testing.T) {
	t.Parallel()
	e, err := edition.Load("poe1")
	if err != nil {
		t.Fatal(err)
	}
	styles := map[uint32]uint32{1: 1, 2: 2, 3: 3, 4: 5}
	testCases := []struct {
		version string
		want    map[string]map[uint32]uint32
	}{
		{version: "3.27.0.10", want: map[string]map[uint32]uint32{"GemEffects.ItemColor": styles}},
		{version: "3.27", want: map[string]map[uint32]uint32{"GemEffects.ItemColor": styles}},
		{version: "3.28.0.16"},
		{version: "3.29.3.6"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.version, func(t *testing.T) {
			t.Parallel()
			values, err := e.ValuesFor(testCase.version)
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]map[uint32]uint32{}
			for _, v := range values {
				got[v.Table+"."+v.Column] = v.Map
			}
			if diff := cmp.Diff(testCase.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("values mismatch (-want +got):\n%s", diff)
			}
		})
	}
	if _, err := e.ValuesFor("3.x"); err == nil {
		t.Error("ValuesFor(3.x): got nil error, want an error")
	}
}

func TestNameColumnsByLayout(t *testing.T) {
	t.Parallel()
	// AlternateTreeVersions as dat-schema 717d788 (3.26 to 3.29) and 1416c40 (3.30, with the
	// version's name last) define it for poe1.
	const versions = `{"name": "AlternateTreeVersions", "columns": [{"name": "Id", "type": "string"},
		{"name": null, "type": "bool"}, {"name": null, "type": "bool"},
		{"name": null, "type": "i32"}, {"name": null, "type": "i32"}, {"name": null, "type": "i32"},
		{"name": null, "type": "i32"}, {"name": null, "type": "i32"}, {"name": null, "type": "i32"},
		{"name": null, "type": "i32"}%s]}`
	e, err := edition.Load("poe1")
	if err != nil {
		t.Fatal(err)
	}
	names := slices.DeleteFunc(e.ColumnNames(), func(c dat.ColumnName) bool { return c.Table != "AlternateTreeVersions" })
	for _, extra := range []string{"", `, {"name": null, "type": "string"}`} {
		s, err := dat.ParseSchema([]byte(`{"version": 7, "tables": [` + fmt.Sprintf(versions, extra) + `]}`))
		if err != nil {
			t.Fatal(err)
		}
		if err := s.NameColumns(names); err != nil {
			t.Fatalf("NameColumns with %q: %v", extra, err)
		}
		def, _ := s.Table("AlternateTreeVersions")
		for name, want := range map[string]int{"SmallAttributeReplaced": 1, "NotableReplacementChance": 9} {
			if got, ok := def.Column(name); !ok || got != want {
				t.Errorf("with %q: Column(%q): got %d, %v, want %d, true", extra, name, got, ok, want)
			}
		}
	}
}
