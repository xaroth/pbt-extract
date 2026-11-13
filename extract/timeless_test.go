package extract_test

import (
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/extract"
)

// timelessTablesJSON are the alternate tree tables of the test schema, with the columns the
// build reads.
const timelessTablesJSON = `
	{"name": "AlternateTreeVersions", "columns": [
		{"name": "Id", "type": "string"},
		{"name": "SmallAttributeReplaced", "type": "bool"}, {"name": "SmallNormalReplaced", "type": "bool"},
		{"name": "SmallAttributeAdditionsMin", "type": "i32"}, {"name": "SmallAttributeAdditionsMax", "type": "i32"},
		{"name": "SmallNormalAdditionsMin", "type": "i32"}, {"name": "SmallNormalAdditionsMax", "type": "i32"},
		{"name": "NotableAdditionsMin", "type": "i32"}, {"name": "NotableAdditionsMax", "type": "i32"},
		{"name": "NotableReplacementChance", "type": "i32"}]},
	{"name": "AlternatePassiveSkills", "columns": [
		{"name": "Id", "type": "string"}, {"name": "AlternateTreeVersionsKey", "type": "foreignrow"},
		{"name": "PassiveType", "type": "enumrow", "array": true}, {"name": "StatsKeys", "type": "foreignrow", "array": true},
		{"name": "Stat1Min", "type": "i32"}, {"name": "Stat1Max", "type": "i32"},
		{"name": "Stat2Min", "type": "i32"}, {"name": "Stat2Max", "type": "i32"},
		{"name": "SpawnWeight", "type": "i32"}, {"name": "ConquerorIndex", "type": "i32"},
		{"name": "RandomMin", "type": "i32"}, {"name": "RandomMax", "type": "i32"},
		{"name": "ConquerorVersion", "type": "i32"}]},
	{"name": "AlternatePassiveAdditions", "columns": [
		{"name": "Id", "type": "string"}, {"name": "AlternateTreeVersionsKey", "type": "foreignrow"},
		{"name": "SpawnWeight", "type": "i32"}, {"name": "StatsKeys", "type": "foreignrow", "array": true},
		{"name": "Stat1Min", "type": "i32"}, {"name": "Stat1Max", "type": "i32"},
		{"name": "Stat2Min", "type": "i32"}, {"name": "Stat2Max", "type": "i32"},
		{"name": "PassiveType", "type": "i32", "array": true}]},`

// timelessGame adds alternate tree tables without rows to g.
func timelessGame(t *testing.T, g fstest.MapFS) {
	t.Helper()
	for _, name := range []string{"AlternateTreeVersions", "AlternatePassiveSkills", "AlternatePassiveAdditions"} {
		g["Data/"+name+".datc64"] = &fstest.MapFile{Data: encode(t, name)}
	}
}

// treeVersion is one AlternateTreeVersions row; the small attribute, small normal and notable
// addition counts are each n to n.
func treeVersion(id string, replaced bool, n, chance int32) []any {
	return []any{id, replaced, replaced, n, n, n, n, n, n, chance}
}

// altPassive is one AlternatePassiveSkills row with stat ranges [lo, hi] each.
func altPassive(id string, version int, kinds []uint32, stats []int, lo, hi, weight, conqueror, conqVersion int32) []any {
	return []any{id, version, kinds, stats, lo, hi, lo, hi, weight, conqueror, int32(0), int32(2), conqVersion}
}

// altAddition is one AlternatePassiveAdditions row with stat ranges [lo, hi] each.
func altAddition(id string, version int, kinds []int32, stats []int, lo, hi, weight int32) []any {
	return []any{id, version, weight, stats, lo, hi, lo, hi, kinds}
}

// timelessRows sets the rows of the alternate tree tables of g.
func timelessRows(t *testing.T, g fstest.MapFS, versions, passives, additions [][]any) {
	t.Helper()
	g["Data/AlternateTreeVersions.datc64"] = &fstest.MapFile{Data: encode(t, "AlternateTreeVersions", versions...)}
	g["Data/AlternatePassiveSkills.datc64"] = &fstest.MapFile{Data: encode(t, "AlternatePassiveSkills", passives...)}
	g["Data/AlternatePassiveAdditions.datc64"] = &fstest.MapFile{Data: encode(t, "AlternatePassiveAdditions", additions...)}
}

func TestTimeless(t *testing.T) {
	t.Parallel()
	g := game(t)
	timelessRows(t, g,
		// BlightJewel follows an Abyss version, so its index could not be its row
		[][]any{treeVersion("None", false, 0, 0), treeVersion("Vaal", true, 0, 100), treeVersion("AbyssMurderous", true, 0, 0),
			treeVersion("BlightJewel", false, 0, 0)},
		[][]any{
			altPassive("vaal_keystone_2_v2", 1, []uint32{4}, []int{4}, 1, 1, 100, 2, 1),
			// stats stay in row order
			altPassive("vaal_small_life", 1, []uint32{1, 2}, []int{3, 1}, 5, 9, 50, 0, 0),
			altPassive("abyss_notable", 2, []uint32{3}, []int{1}, 1, 1, 100, 0, 0),
			altPassive("blight_notable", 3, []uint32{3}, []int{1}, 1, 1, 100, 0, 0),
		},
		[][]any{
			altAddition("abyss_addition", 2, []int32{3}, []int{1}, 1, 1, 100),
			altAddition("vaal_addition", 1, []int32{3}, []int{2}, 7, 12, 30),
		})
	in := input(t, g, export())
	var warnings []string
	in.Warn = func(msg string) { warnings = append(warnings, msg) }
	got, err := extract.Dataset(in)
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	wantWarnings := []string{"timeless version BlightJewel (row 3) follows an Abyss version; left out with its 1 replacements and 0 additions"}
	if diff := cmp.Diff(wantWarnings, warnings); diff != "" {
		t.Errorf("warnings mismatch (-want +got):\n%s", diff)
	}
	want := []dataset.TimelessVersion{
		{ID: "None"},
		{
			ID: "Vaal", SmallAttributeReplaced: true, SmallNormalReplaced: true, NotableReplacementChance: 100,
			Replacements: []dataset.TimelessReplacement{
				{ID: "vaal_keystone_2_v2", Kinds: []dataset.TimelessKind{dataset.TimelessKeystone}, Stats: []dataset.StatRange{{Stat: 4, Min: 1, Max: 1}},
					Weight: 100, Additions: dataset.ValueRange{Max: 2}, Conqueror: dataset.TimelessConqueror{Index: 2, Version: 1}},
				{ID: "vaal_small_life", Kinds: []dataset.TimelessKind{dataset.TimelessSmallAttribute, dataset.TimelessSmallNormal},
					Stats:  []dataset.StatRange{{Stat: 3, Min: 5, Max: 9}, {Stat: 1, Min: 5, Max: 9}},
					Weight: 50, Additions: dataset.ValueRange{Max: 2}},
			},
			Additions: []dataset.TimelessAddition{
				{ID: "vaal_addition", Kinds: []dataset.TimelessKind{dataset.TimelessNotable}, Stats: []dataset.StatRange{{Stat: 2, Min: 7, Max: 12}}, Weight: 30},
			},
		},
	}
	for _, tree := range got.Trees {
		if diff := cmp.Diff(want, tree.TimelessVersions, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("tree %s timeless versions mismatch (-want +got):\n%s", tree.Variant, diff)
		}
	}
}

func TestTimelessErrors(t *testing.T) {
	t.Parallel()
	versions := [][]any{treeVersion("None", false, 0, 0), treeVersion("Vaal", true, 0, 100)}
	testCases := []struct {
		name                          string
		versions, passives, additions [][]any
	}{
		{name: "replacement without kind", passives: [][]any{altPassive("x", 1, nil, []int{1}, 1, 1, 100, 0, 0)}},
		{name: "addition without kind", additions: [][]any{altAddition("x", 1, nil, []int{1}, 1, 1, 100)}},
		{name: "unknown kind", passives: [][]any{altPassive("x", 1, []uint32{6}, []int{1}, 1, 1, 100, 0, 0)}},
		{name: "third stat", passives: [][]any{altPassive("x", 1, []uint32{3}, []int{1, 2, 3}, 1, 1, 100, 0, 0)}},
		{name: "stat out of range", additions: [][]any{altAddition("x", 1, []int32{3}, []int{99}, 1, 1, 100)}},
		{name: "version out of range", additions: [][]any{altAddition("x", 5, []int32{3}, []int{1}, 1, 1, 100)}},
		{name: "negative weight", additions: [][]any{altAddition("x", 1, []int32{3}, []int{1}, 1, 1, -1)}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			v := testCase.versions
			if v == nil {
				v = versions
			}
			g := game(t)
			timelessRows(t, g, v, testCase.passives, testCase.additions)
			if _, err := extract.Dataset(input(t, g, export())); err == nil {
				t.Errorf("Dataset: got nil error, want an error")
			}
		})
	}
}
