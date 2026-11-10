package edition_test

import (
	"fmt"
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
	if _, err := edition.Load("nope"); err == nil {
		t.Error("Load(nope): got nil error, want an error")
	}
}

func TestNameColumns(t *testing.T) {
	t.Parallel()
	// QuestStaticRewards with the columns poe1 names left unnamed, as dat-schema has them.
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
			err = e.NameColumns(s)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("NameColumns: got error %v, want error %v", err, testCase.wantErr)
			}
			if testCase.wantErr {
				return
			}
			def, _ := s.Table("QuestStaticRewards")
			for _, c := range e.Columns {
				if got, ok := def.Column(c.Name); !ok || got != c.Index {
					t.Errorf("Column(%q): got %d, %v, want %d, true", c.Name, got, ok, c.Index)
				}
			}
		})
	}
}
