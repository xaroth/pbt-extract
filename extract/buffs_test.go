package extract_test

import (
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat/dattest"
	"github.com/xaroth/pbt-extract/extract"
)

// buffTables are the buff tables of the test schema, with the columns the build reads. Flasks
// is in itemTables.
const buffTables = `
	{"name": "BuffDefinitions", "columns": [
		{"name": "Id", "type": "string"}, {"name": "Name", "type": "string"},
		{"name": "StatsKeys", "type": "foreignrow", "array": true, "references": {"table": "Stats"}},
		{"name": "Stats2", "type": "foreignrow", "array": true, "references": {"table": "Stats"}},
		{"name": "GrantedStats", "type": "foreignrow", "array": true, "references": {"table": "Stats"}},
		{"name": "GrantedFlags", "type": "foreignrow", "array": true, "references": {"table": "Stats"}},
		{"name": "ConditionStats", "type": "foreignrow", "array": true, "references": {"table": "Stats"}},
		{"name": "Maximum_StatsKey", "type": "foreignrow", "references": {"table": "Stats"}},
		{"name": "Current_StatsKey", "type": "foreignrow", "references": {"table": "Stats"}},
		{"name": "MinStat", "type": "foreignrow", "references": {"table": "Stats"}},
		{"name": "BuffCategory", "type": "i32"}, {"name": "Id2", "type": "string"}, {"name": "BuffLimit", "type": "i32"},
		{"name": "IsSkillBuff", "type": "bool"}, {"name": "Invisible", "type": "bool"}, {"name": "Removable", "type": "bool"},
		{"name": "CancelOnDeath", "type": "bool"}, {"name": "IsRecovery", "type": "bool"}]},
	{"name": "Tinctures", "columns": [
		{"name": "BaseItem", "type": "foreignrow", "references": {"table": "BaseItemTypes"}},
		{"name": "DebuffInterval", "type": "i32"}, {"name": "Cooldown", "type": "i32"}]}`

// buffDef is a BuffDefinitions row. GrantedStats is stats followed by extra unless granted is
// set; maximum, current and minimum are Stats rows or 0 for none.
type buffDef struct {
	id, name, id2               string
	stats, extra, granted       []int
	flags, condition            []int
	maximum, current, minimum   int
	category, limit             int32
	skill, invisible, removable bool
	cancelOnDeath, recovery     bool
}

func (b buffDef) row() []any {
	granted := b.granted
	if granted == nil {
		granted = slices.Concat(b.stats, b.extra)
	}
	ref := func(stat int) int {
		if stat == 0 {
			return dattest.Null
		}
		return stat
	}
	return []any{b.id, b.name, b.stats, b.extra, granted, b.flags, b.condition,
		ref(b.maximum), ref(b.current), ref(b.minimum), b.category, b.id2, b.limit,
		b.skill, b.invisible, b.removable, b.cancelOnDeath, b.recovery}
}

// buffDefs are the BuffDefinitions rows of the test game. The tree's BuffTemplates use the
// first two.
func buffDefs() []buffDef {
	return []buffDef{
		{id: "player_aura_damage", name: "Anger", stats: []int{4}, category: 1, skill: true, removable: true, cancelOnDeath: true},
		{id: "frenzy_charge", name: "Frenzy Charges", maximum: 1, current: 2, minimum: 3, category: 3, cancelOnDeath: true},
		// no name: left out of the names
		{id: "curse_test", id2: "CurseDebuff", stats: []int{2}, extra: []int{3}, flags: []int{1}, condition: []int{0},
			category: 5, limit: 1, skill: true, invisible: true, recovery: true},
	}
}

func encodeBuffDefs(t *testing.T, defs ...buffDef) []byte {
	t.Helper()
	rows := make([][]any, len(defs))
	for i, d := range defs {
		rows[i] = d.row()
	}
	return encode(t, "BuffDefinitions", rows...)
}

// buffGame adds the buff tables to g, with no tinctures.
func buffGame(t *testing.T, g fstest.MapFS) {
	t.Helper()
	g["Data/BuffDefinitions.datc64"] = &fstest.MapFile{Data: encodeBuffDefs(t, buffDefs()...)}
	g["Data/Tinctures.datc64"] = &fstest.MapFile{Data: encode(t, "Tinctures")}
}

// flaskGame returns the test game with a utility flask with a buff and a tincture added to the
// items: classes 5 and 6, bases after the gems.
func flaskGame(t *testing.T) fstest.MapFS {
	t.Helper()
	g := game(t)
	utility, tincture := len(itemBases())+len(gemBases()), len(itemBases())+len(gemBases())+1
	g["Data/ItemClasses.datc64"] = &fstest.MapFile{Data: encode(t, "ItemClasses",
		[]any{"Body Armour", "Body Armours"}, []any{"Currency", "Currency"}, []any{"Shield", "Shields"},
		[]any{"Two Hand Axe", "Two Hand Axes"}, []any{"LifeFlask", "Life Flasks"},
		[]any{"UtilityFlask", "Utility Flasks"}, []any{"Tincture", "Tinctures"})}
	g["Data/BaseItemTypes.datc64"] = &fstest.MapFile{Data: encode(t, "BaseItemTypes", slices.Concat(itemBases(), gemBases(), [][]any{
		{"Metadata/Items/Flasks/FlaskUtility5", 5, "Granite Flask", []int(nil), []int(nil), "Metadata/Items/Flasks/AbstractUtilityFlask"},
		{"Metadata/Items/Tinctures/Tincture1", 6, "Prismatic Tincture", []int(nil), []int(nil), "Metadata/Items/Tinctures/AbstractTincture"},
	})...)}
	g["Data/Flasks.datc64"] = &fstest.MapFile{Data: encode(t, "Flasks",
		[]any{4, uint32(1), int32(2400), int32(0), int32(35), int32(30), dattest.Null, []int32(nil)},
		[]any{utility, uint32(4), int32(0), int32(0), int32(60), int32(40), 2, []int32{1500, 7}})}
	g["Data/ComponentCharges.datc64"] = &fstest.MapFile{Data: encode(t, "ComponentCharges",
		[]any{"Metadata/Items/Flasks/FlaskLife11", int32(45), int32(15)},
		[]any{"Metadata/Items/Flasks/FlaskUtility5", int32(60), int32(30)})}
	g["Data/Tinctures.datc64"] = &fstest.MapFile{Data: encode(t, "Tinctures", []any{tincture, int32(700), int32(8000)})}
	return g
}

func TestBuffs(t *testing.T) {
	t.Parallel()
	got, err := extract.Dataset(input(t, flaskGame(t), export()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	handle := func(h uint32) *uint32 { return &h }
	want := &dataset.Buffs{
		Definitions: []dataset.BuffDefinition{
			{ID: "player_aura_damage", Category: 1, Stats: []uint32{4}, SkillBuff: true, Removable: true, CancelOnDeath: true},
			{ID: "frenzy_charge", Category: 3, MaximumStat: handle(1), CurrentStat: handle(2), MinimumStat: handle(3), CancelOnDeath: true},
			{ID: "curse_test", Category: 5, ID2: "CurseDebuff", Stats: []uint32{2}, ExtraStats: []uint32{3},
				Flags: []uint32{1}, ConditionStats: []uint32{0}, Limit: 1, SkillBuff: true, Invisible: true, Recovery: true},
		},
		// the life flask has no buff
		Flasks:    []dataset.FlaskBuff{{Item: "Metadata/Items/Flasks/FlaskUtility5", Buff: "curse_test", Values: []int32{1500, 7}}},
		Tinctures: []dataset.Tincture{{Item: "Metadata/Items/Tinctures/Tincture1", DebuffInterval: 700, Cooldown: 8000}},
	}
	if diff := cmp.Diff(want, got.Buffs, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Buffs mismatch (-want +got):\n%s", diff)
	}
	wantNames := []dataset.Name{{ID: "player_aura_damage", Name: "Anger"}, {ID: "frenzy_charge", Name: "Frenzy Charges"}}
	if diff := cmp.Diff(wantNames, got.Texts[0].BuffNames); diff != "" {
		t.Errorf("BuffNames mismatch (-want +got):\n%s", diff)
	}
	if _, err := dataset.Write(t.TempDir(), got); err != nil {
		t.Errorf("Write: %v", err)
	}
}

func TestBuffsErrors(t *testing.T) {
	t.Parallel()
	setDefs := func(edit func([]buffDef)) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, g fstest.MapFS) {
			defs := buffDefs()
			edit(defs)
			g["Data/BuffDefinitions.datc64"] = &fstest.MapFile{Data: encodeBuffDefs(t, defs...)}
		}
	}
	setFlask := func(row []any) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, g fstest.MapFS) {
			g["Data/Flasks.datc64"] = &fstest.MapFile{Data: encode(t, "Flasks", row)}
		}
	}
	testCases := []struct {
		name    string
		game    func(*testing.T, fstest.MapFS)
		wantMsg string
	}{
		{"missing table", func(_ *testing.T, g fstest.MapFS) { delete(g, "Data/Tinctures.datc64") }, "Tinctures.datc64"},
		{"granted stats differ", setDefs(func(d []buffDef) { d[2].granted = []int{2} }),
			"BuffDefinitions row 2: GrantedStats is not StatsKeys followed by Stats2"},
		{"stat out of range", setDefs(func(d []buffDef) { d[2].extra = []int{9} }), "row 2: column Stats2: Stats row 9 out of range"},
		{"flag out of range", setDefs(func(d []buffDef) { d[2].flags = []int{9} }), "row 2: column GrantedFlags: Stats row 9 out of range"},
		{"condition stat out of range", setDefs(func(d []buffDef) { d[2].condition = []int{9} }), "column ConditionStats: Stats row 9 out of range"},
		{"maximum stat out of range", setDefs(func(d []buffDef) { d[1].maximum = 9 }), "row 1: column Maximum_StatsKey: Stats row 9 out of range"},
		{"minimum stat out of range", setDefs(func(d []buffDef) { d[1].minimum = 9 }), "row 1: column MinStat: Stats row 9 out of range"},
		{"flask values for another stat count", setFlask([]any{4, uint32(1), int32(0), int32(0), int32(35), int32(30), 2, []int32{1}}),
			"Flasks row 0: 1 values for the 2 stats of buff curse_test"},
		{"flask buff out of range", setFlask([]any{4, uint32(1), int32(0), int32(0), int32(35), int32(30), 9, []int32{1}}),
			"Flasks row 0: BuffDefinitions row 9 out of range"},
		{"flask base not in items", setFlask([]any{1, uint32(1), int32(0), int32(0), int32(35), int32(30), 0, []int32{1}}),
			"Flasks row 0: base Metadata/Items/Currency/CurrencyRerollRare is not in the items group"},
		{"tincture base not in items", func(t *testing.T, g fstest.MapFS) {
			g["Data/Tinctures.datc64"] = &fstest.MapFile{Data: encode(t, "Tinctures", []any{1, int32(700), int32(8000)})}
		}, "Tinctures row 0: base Metadata/Items/Currency/CurrencyRerollRare is not in the items group"},
		{"tincture base out of range", func(t *testing.T, g fstest.MapFS) {
			g["Data/Tinctures.datc64"] = &fstest.MapFile{Data: encode(t, "Tinctures", []any{99, int32(700), int32(8000)})}
		}, "Tinctures row 0: BaseItemTypes row 99 out of range"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := game(t)
			testCase.game(t, g)
			_, err := extract.Dataset(input(t, g, export()))
			if err == nil || !strings.Contains(err.Error(), testCase.wantMsg) {
				t.Errorf("got error %v, want one containing %q", err, testCase.wantMsg)
			}
		})
	}
}
