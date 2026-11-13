package extract_test

import (
	"slices"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/dat/dattest"
	"github.com/xaroth/pbt-extract/extract"
)

// overrideTablesJSON are the node override tables of the test schema, with the columns the
// build reads.
const overrideTablesJSON = `
	{"name": "PassiveSkillOverrides", "columns": [
		{"name": "Id", "type": "string"}, {"name": "Name", "type": "string"}, {"name": "NodeIcon", "type": "string"},
		{"name": "Stats", "type": "foreignrow", "array": true}, {"name": "StatValues", "type": "i32", "array": true},
		{"name": "Type", "type": "foreignrow"},
		{"name": "RequiresAdjacent", "type": "i32"}, {"name": "MaxAdjacent", "type": "i32"},
		{"name": "AllocatedPassiveSkill", "type": "foreignrow"},
		{"name": "TattooBlockingPassive", "type": "foreignrow", "array": true}]},
	{"name": "PassiveSkillOverrideTypes", "columns": [{"name": "Id", "type": "string"}]},
	{"name": "PassiveSkillTattoos", "columns": [
		{"name": "Override", "type": "foreignrow"}, {"name": "Set", "type": "foreignrow"},
		{"name": "OverrideType", "type": "foreignrow"}]},
	{"name": "PassiveSkillTattooTargetSets", "columns": [{"name": "Id", "type": "string"}]},`

// overrideGame adds override tables without rows to g.
func overrideGame(t *testing.T, g fstest.MapFS) {
	t.Helper()
	g["Data/PassiveSkillOverrides.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkillOverrides")}
	g["Data/PassiveSkillOverrideTypes.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkillOverrideTypes")}
	g["Data/PassiveSkillTattoos.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkillTattoos")}
	g["Data/PassiveSkillTattooTargetSets.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkillTattooTargetSets")}
}

// override is one PassiveSkillOverrides row.
func override(id, name, icon string, stats []int, values []int32, typ int, minAdj, maxAdj int32, passive int, blocking []int) []any {
	return []any{id, name, icon, stats, values, typ, minAdj, maxAdj, passive, blocking}
}

const (
	tasalioType = iota
	keystoneType
	runegraftType
	unknownType
)

const (
	smallIntSet = iota
	keystoneSet
	masterySet
)

// overridesGame is the test game with overrides: rows 0 to 3 of PassiveSkills are the tree's
// nodes 1 to 3 and a keystone, Acrobatics, that the tree lacks.
func overridesGame(t *testing.T, overrides [][]any, tattoos [][]any, sets ...string) fstest.MapFS {
	t.Helper()
	g := game(t)
	acrobatics := passive(50, []int{2}, [5]int32{7}, nil, 0, false, nil, [5]int32{})
	acrobatics[len(acrobatics)-2], acrobatics[len(acrobatics)-1] = "Acrobatics", "Art/acrobatics.dds"
	might := passive(2, []int{3, 1}, [5]int32{10, 5}, []int{0, 1}, 1, true, []int{3}, [5]int32{20})
	might[len(might)-2], might[len(might)-1] = "Might", "Art/might.dds"
	g["Data/PassiveSkills.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkills",
		passive(1, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}), might, masteryPassive(3), acrobatics)}
	g["Data/PassiveSkillOverrideTypes.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkillOverrideTypes",
		[]any{"TasalioTattoo"}, []any{"KeystoneTattoo"}, []any{"AlternateMastery"}, []any{"UnknownTattoo"})}
	var setRows [][]any
	for _, s := range sets {
		setRows = append(setRows, []any{s})
	}
	g["Data/PassiveSkillTattooTargetSets.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkillTattooTargetSets", setRows...)}
	g["Data/PassiveSkillOverrides.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkillOverrides", overrides...)}
	g["Data/PassiveSkillTattoos.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkillTattoos", tattoos...)}
	return g
}

func TestOverrides(t *testing.T) {
	t.Parallel()
	null := dattest.Null
	g := overridesGame(t, [][]any{
		// blocks Might (node 2) and Acrobatics, which the tree lacks
		override("Tasalio3", "Tattoo of the Tasalio Shaman", "Art/curse.dds", []int{4}, []int32{5}, tasalioType, 0, 0, null, []int{1, 3}),
		// the placeholder of the random keystone tattoo
		override("DisplayRandomKeystone", "", "Art/random.dds", []int{0}, []int32{1}, unknownType, 0, 0, null, nil),
		override("KeystoneAcrobatics", "", "", nil, nil, keystoneType, 0, 0, 3, nil),
		override("KeystoneMight", "", "", nil, nil, keystoneType, 0, 0, 1, nil),
		override("runegraft_1", "Runegraft of the Warp", "Art/warp.dds", []int{2, 1}, []int32{3, 2}, runegraftType, 2, 4, null, nil),
		override("runegraft_12", "[DNT - UNUSED] Grafted", "Art/x.dds", []int{1}, []int32{1}, runegraftType, 0, 0, null, nil),
		override("Bad1", "Tattoo of Nine", "Art/x.dds", []int{9}, []int32{1}, tasalioType, 0, 0, null, nil),
		override("Bad2", "Tattoo of Nowhere", "Art/x.dds", []int{1}, []int32{1}, tasalioType, 0, 0, null, nil),
		override("Bad3", "Tattoo of Nothing", "", []int{1}, []int32{1}, tasalioType, 0, 0, null, nil),
		override("Nameless", "", "Art/x.dds", []int{1}, []int32{1}, tasalioType, 0, 0, null, nil),
	}, [][]any{
		{0, smallIntSet, null},
		{1, keystoneSet, keystoneType},
		{4, masterySet, null},
		{5, masterySet, null},
		{6, smallIntSet, null},
		{8, smallIntSet, null},
		{9, smallIntSet, null},
	}, "SmallIntelligence", "Keystone", "Mastery")
	in := input(t, g, export())
	var warnings []string
	in.Warn = func(msg string) { warnings = append(warnings, msg) }
	got, err := extract.Dataset(in)
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	want := []dataset.NodeOverride{
		{ID: "KeystoneAcrobatics", Kind: dataset.Tattoo, Stats: []dataset.StatValue{{Stat: 2, Value: 7}}, Icon: "Art/acrobatics.png", Target: dataset.KeystoneTarget},
		{ID: "KeystoneMight", Kind: dataset.Tattoo, Stats: []dataset.StatValue{{Stat: 1, Value: 5}, {Stat: 3, Value: 10}}, Icon: "Art/might.png", Target: dataset.KeystoneTarget},
		{ID: "Nameless", Kind: dataset.Tattoo, Stats: []dataset.StatValue{{Stat: 1, Value: 1}}, Icon: "Art/x.png", Target: dataset.SmallIntelligence},
		{ID: "Tasalio3", Kind: dataset.Tattoo, Stats: []dataset.StatValue{{Stat: 4, Value: 5}}, Icon: "Art/curse.png", Target: dataset.SmallIntelligence, ExcludedNodes: []uint32{2}},
		{ID: "runegraft_1", Kind: dataset.Runegraft, Stats: []dataset.StatValue{{Stat: 1, Value: 2}, {Stat: 2, Value: 3}}, Icon: "Art/warp.png", Target: dataset.MasteryTarget, MinNeighbours: 2, MaxNeighbours: 4},
	}
	// Might has a ruthless alternate without buffs; Acrobatics has none.
	ruthless := slices.Clone(want)
	ruthless[1].Stats = []dataset.StatValue{{Stat: 3, Value: 20}}
	for i, w := range [][]dataset.NodeOverride{want, ruthless} {
		if diff := cmp.Diff(w, got.Trees[i].Overrides, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("tree %s overrides mismatch (-want +got):\n%s", got.Trees[i].Variant, diff)
		}
	}
	wantNames := []dataset.Name{
		{ID: "KeystoneAcrobatics", Name: "Acrobatics"},
		{ID: "KeystoneMight", Name: "Might"},
		{ID: "Tasalio3", Name: "Tattoo of the Tasalio Shaman"},
		{ID: "runegraft_1", Name: "Runegraft of the Warp"},
	}
	if diff := cmp.Diff(wantNames, got.Texts[0].OverrideNames); diff != "" {
		t.Errorf("override names mismatch (-want +got):\n%s", diff)
	}
	wantWarnings := []string{
		"override KeystoneMight: the buffs of passive Might are not carried",
		`override runegraft_12: named "[DNT - UNUSED] Grafted", unused; left out`,
		"override Bad1: stat 1 does not resolve; left out",
		"override Bad2: no PassiveSkillTattoos row gives it a target; left out",
		"override Bad3: no icon; left out",
		"override Nameless: no English name",
	}
	if diff := cmp.Diff(wantWarnings, warnings); diff != "" {
		t.Errorf("warnings mismatch (-want +got):\n%s", diff)
	}
	if _, err := dataset.Write(t.TempDir(), got); err != nil {
		t.Errorf("Write: %v", err)
	}
}

func TestOverridesErrors(t *testing.T) {
	t.Parallel()
	null := dattest.Null
	tasalio := override("Tasalio3", "Tattoo of the Tasalio Shaman", "Art/curse.dds", []int{4}, []int32{5}, tasalioType, 0, 0, null, nil)
	testCases := []struct {
		name      string
		overrides [][]any
		tattoos   [][]any
		sets      []string
	}{
		{name: "target set without an override target", overrides: [][]any{tasalio}, tattoos: [][]any{{0, 0, null}}, sets: []string{"Wormhole"}},
		{name: "two tattoos of one override", overrides: [][]any{tasalio}, tattoos: [][]any{{0, 0, null}, {0, 0, null}}, sets: []string{"Keystone"}},
		{name: "random tattoos of one type with two targets", overrides: [][]any{tasalio, tasalio, tasalio},
			tattoos: [][]any{{0, 0, keystoneType}, {1, 1, keystoneType}}, sets: []string{"Keystone", "Mastery"}},
		{name: "repeated id", overrides: [][]any{tasalio, tasalio}, tattoos: [][]any{{0, 0, null}, {1, 0, null}}, sets: []string{"Keystone"}},
		{name: "null type", overrides: [][]any{override("X", "X", "Art/x.dds", nil, nil, null, 0, 0, null, nil)},
			tattoos: [][]any{{0, 0, null}}, sets: []string{"Keystone"}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := overridesGame(t, testCase.overrides, testCase.tattoos, testCase.sets...)
			if _, err := extract.Dataset(input(t, g, export())); err == nil {
				t.Errorf("Dataset: got nil error, want an error")
			}
		})
	}
}

func TestOverridesWithoutBlockingColumn(t *testing.T) {
	t.Parallel()
	null := dattest.Null
	g := overridesGame(t, nil, [][]any{{0, smallIntSet, null}}, "SmallIntelligence")
	def, _ := schema(t).Table("PassiveSkillOverrides")
	older := dat.TableDef{Name: def.Name, Columns: def.Columns[:len(def.Columns)-1]}
	g["Data/PassiveSkillOverrides.datc64"] = &fstest.MapFile{Data: dattest.Encode(t, older,
		override("Tasalio3", "Tattoo of the Tasalio Shaman", "Art/curse.dds", []int{4}, []int32{5}, tasalioType, 0, 0, null, nil)[:len(def.Columns)-1])}
	got, err := extract.Dataset(input(t, g, export()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	want := []dataset.NodeOverride{{ID: "Tasalio3", Kind: dataset.Tattoo, Stats: []dataset.StatValue{{Stat: 4, Value: 5}}, Icon: "Art/curse.png", Target: dataset.SmallIntelligence}}
	if diff := cmp.Diff(want, got.Trees[0].Overrides, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("overrides mismatch (-want +got):\n%s", diff)
	}
}
