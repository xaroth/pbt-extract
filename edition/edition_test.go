package edition_test

import (
	"fmt"
	"slices"
	"testing"

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
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			s, err := dat.ParseSchema([]byte(`{"version": 7, "tables": [` + testCase.table + `]}`))
			if err != nil {
				t.Fatal(err)
			}
			err = s.NameColumns(e.ColumnNames())
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
	testCases := []struct {
		version string
		want    []string
	}{
		{version: "3.27.0.10", want: []string{"GemEffects.ItemColor 4->5"}},
		{version: "3.27", want: []string{"GemEffects.ItemColor 4->5"}},
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
			var got []string
			for _, v := range values {
				got = append(got, fmt.Sprintf("%s.%s 4->%d", v.Table, v.Column, v.Map[4]))
			}
			if !slices.Equal(got, testCase.want) {
				t.Errorf("got %v, want %v", got, testCase.want)
			}
		})
	}
	if _, err := e.ValuesFor("3.x"); err == nil {
		t.Error("ValuesFor(3.x): got nil error, want an error")
	}
}
