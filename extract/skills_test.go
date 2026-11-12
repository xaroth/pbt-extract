package extract_test

import (
	"maps"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat/dattest"
	"github.com/xaroth/pbt-extract/edition"
	"github.com/xaroth/pbt-extract/extract"
)

// skillTablesJSON are the skill tables of the test schema, with the columns the build reads.
// ItemClasses and BaseItemTypes are those of itemTables.
const skillTablesJSON = `
	{"name": "ActiveSkillType", "columns": [{"name": "Id", "type": "string"}]},
	{"name": "GemTags", "columns": [{"name": "Id", "type": "string"}]},
	{"name": "CostTypes", "columns": [{"name": "Id", "type": "string"}]},
	{"name": "EffectivenessCostConstants", "columns": [{"name": "Id", "type": "string"}]},
	{"name": "SkillGems", "columns": [
		{"name": "BaseItemTypesKey", "type": "foreignrow"},
		{"name": "StrengthRequirementPercent", "type": "i32"}, {"name": "DexterityRequirementPercent", "type": "i32"},
		{"name": "IntelligenceRequirementPercent", "type": "i32"},
		{"name": "VaalVariant_BaseItemTypesKey", "type": "foreignrow"},
		{"name": "IsVaalVariant", "type": "bool"}, {"name": "IsSupport", "type": "bool"},
		{"name": "AwakenedVariant", "type": "row"}, {"name": "RegularVariant", "type": "row"},
		{"name": "GemColour", "type": "i32"},
		{"name": "GemVariants", "type": "foreignrow", "array": true}]},
	{"name": "GemEffects", "columns": [
		{"name": "Id", "type": "string"},
		{"name": "GrantedEffect", "type": "foreignrow"}, {"name": "GrantedEffectHardmode", "type": "foreignrow"},
		{"name": "GrantedEffect2", "type": "foreignrow"}, {"name": "GrantedEffect2Hardmode", "type": "foreignrow"},
		{"name": "GemTags", "type": "foreignrow", "array": true},
		{"name": "ItemColor", "type": "enumrow"}]},
	{"name": "GrantedEffects", "columns": [
		{"name": "Id", "type": "string"}, {"name": "IsSupport", "type": "bool"},
		{"name": "AllowedActiveSkillTypes", "type": "foreignrow", "array": true},
		{"name": "AddedActiveSkillTypes", "type": "foreignrow", "array": true},
		{"name": "ExcludedActiveSkillTypes", "type": "foreignrow", "array": true},
		{"name": "SupportsGemsOnly", "type": "bool"}, {"name": "CannotBeSupported", "type": "bool"},
		{"name": "CastTime", "type": "i32"}, {"name": "ActiveSkill", "type": "foreignrow"},
		{"name": "AddedMinionActiveSkillTypes", "type": "foreignrow", "array": true},
		{"name": "SupportWeaponRestrictions", "type": "foreignrow", "array": true},
		{"name": "StatSet1", "type": "foreignrow"}, {"name": "StatSet2", "type": "foreignrow", "array": true}]},
	{"name": "GrantedEffectsPerLevel", "columns": [
		{"name": "GrantedEffect", "type": "foreignrow"}, {"name": "Level", "type": "i32"},
		{"name": "PlayerLevelReq", "type": "f32"}, {"name": "CostMultiplier", "type": "i32"},
		{"name": "StoredUses", "type": "i32"}, {"name": "Cooldown", "type": "i32"},
		{"name": "VaalSouls", "type": "i32"}, {"name": "VaalStoredUses", "type": "i32"},
		{"name": "CooldownGroup", "type": "i32"}, {"name": "SoulGainPreventionDuration", "type": "i32"},
		{"name": "AttackSpeedMultiplier", "type": "i32"},
		{"name": "CostAmounts", "type": "i32", "array": true}, {"name": "CostTypes", "type": "foreignrow", "array": true},
		{"name": "ManaReservationFlat", "type": "i32"}, {"name": "ManaReservationPercent", "type": "i32"},
		{"name": "LifeReservationFlat", "type": "i32"}, {"name": "LifeReservationPercent", "type": "i32"},
		{"name": "AttackTime", "type": "i32"}]},
	{"name": "GrantedEffectQualityStats", "columns": [
		{"name": "GrantedEffectsKey", "type": "foreignrow"},
		{"name": "StatsKeys", "type": "foreignrow", "array": true},
		{"name": "StatsValuesPermille", "type": "i32", "array": true}]},
	{"name": "GrantedEffectStatSets", "columns": [
		{"name": "Id", "type": "string"},
		{"name": "ImplicitStats", "type": "foreignrow", "array": true},
		{"name": "ConstantStats", "type": "foreignrow", "array": true},
		{"name": "ConstantStatsValues", "type": "i32", "array": true},
		{"name": "BaseEffectiveness", "type": "f32"}, {"name": "IncrementalEffectiveness", "type": "f32"}]},
	{"name": "GrantedEffectStatSetsPerLevel", "columns": [
		{"name": "StatSet", "type": "foreignrow"}, {"name": "GemLevel", "type": "i32"},
		{"name": "PlayerLevelReq", "type": "f32"}, {"name": "SpellCritChance", "type": "i32"},
		{"name": "AttackCritChance", "type": "i32"}, {"name": "BaseMultiplier", "type": "i32"},
		{"name": "DamageEffectiveness", "type": "i32"},
		{"name": "BaseResolvedValues", "type": "i32", "array": true},
		{"name": "AdditionalStatsValues", "type": "i32", "array": true},
		{"name": "AdditionalFlags", "type": "foreignrow", "array": true},
		{"name": "FloatStats", "type": "foreignrow", "array": true},
		{"name": "InterpolationBases", "type": "foreignrow", "array": true},
		{"name": "AdditionalStats", "type": "foreignrow", "array": true},
		{"name": "StatInterpolations", "type": "enumrow", "array": true},
		{"name": "FloatStatsValues", "type": "f32", "array": true}]},
	{"name": "ActiveSkills", "columns": [
		{"name": "Id", "type": "string"}, {"name": "DisplayedName", "type": "string"},
		{"name": "ActiveSkillTypes", "type": "foreignrow", "array": true},
		{"name": "WeaponRestriction_ItemClassesKeys", "type": "foreignrow", "array": true},
		{"name": "SkillTotemId", "type": "enumrow"},
		{"name": "MinionActiveSkillTypes", "type": "foreignrow", "array": true},
		{"name": "SkillMine", "type": "i32"}, {"name": "TransfigureBase", "type": "row"}]},`

// skillGem is a SkillGems row: base item, attribute split, Vaal base item and flag, support
// flag, awakened and regular gem rows, colour and gem effects.
func skillGem(base int, attrs [3]int32, vaalBase int, vaal, support bool, awakened, regular int, colour int32, effects ...int) []any {
	return []any{base, attrs[0], attrs[1], attrs[2], vaalBase, vaal, support, awakened, regular, colour, effects}
}

// grantedEffect is a GrantedEffects row of an active skill: id, cast time, active skill and
// stat sets.
func grantedEffect(id string, cast int32, active, set int, more ...int) []any {
	return []any{id, false, []int(nil), []int(nil), []int(nil), false, false, cast, active, []int(nil), []int(nil), set, more}
}

// supportEffect is a GrantedEffects row of a support with its allowed, added, excluded and
// added minion types, weapon restrictions and stat set.
func supportEffect(id string, allowed, added, excluded, minion, weapons []int, set int) []any {
	return []any{id, true, allowed, added, excluded, true, false, int32(0), dattest.Null, minion, weapons, set, []int(nil)}
}

// effectLevel is a GrantedEffectsPerLevel row with one cost. Fields not given are zero.
func effectLevel(effect int, level int32, player float32, cost int32) []any {
	return []any{effect, level, player, int32(100), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0),
		[]int32{cost}, []int{0}, int32(0), int32(0), int32(0), int32(0), int32(0)}
}

// statSetLevel is a GrantedEffectStatSetsPerLevel row with no stats.
func statSetLevel(set int, level int32, player float32) []any {
	return []any{set, level, player, int32(0), int32(0), int32(0), int32(0),
		[]int32(nil), []int32(nil), []int(nil), []int(nil), []int(nil), []int(nil), []uint32(nil), []float32(nil)}
}

// activeSkill is an ActiveSkills row with no weapon restriction or minion types.
func activeSkill(id, name string, types []int, totem uint32, mine int32, base int) []any {
	return []any{id, name, types, []int(nil), totem, []int(nil), mine, base}
}

// gemBase is the BaseItemTypes row of the first of gemBases.
var gemBase = len(itemBases())

// gemBases are the BaseItemTypes rows of gems, which follow itemBases. Their class, Currency, is
// not one the items group keeps.
func gemBases() [][]any {
	return [][]any{
		{"Metadata/Items/Gems/SkillGemFireball", 1, "Fireball", []int(nil), []int(nil), "", uint32(43)},
		{"Metadata/Items/Gems/SkillGemVaalFireball", 1, "Vaal Fireball", []int(nil), []int(nil), "", uint32(43)},
		{"Metadata/Items/Gems/SupportGemEcho", 1, "Echo Support", []int(nil), []int(nil), "", uint32(43)},
		{"Metadata/Items/Gems/SupportGemAwakenedEcho", 1, "", []int(nil), []int(nil), "", uint32(43)},
	}
}

// skillRows are the rows of the skill tables of the test game. Stats are those of game: 0 level,
// 1 life_per_level, 2 base_speed, 3 base_strength, 4 aura_damage_+%.
func skillRows() map[string][][]any {
	return map[string][][]any{
		"ActiveSkillType":            {[]any{"Spell"}, []any{"Projectile"}, []any{"AND"}, []any{"Minion"}},
		"GemTags":                    {[]any{"fire"}, []any{"spell"}},
		"CostTypes":                  {[]any{"Mana"}},
		"EffectivenessCostConstants": {[]any{"DamageValue"}},
		"SkillGems": {
			skillGem(gemBase, [3]int32{0, 40, 60}, gemBase+1, false, false, dattest.Null, dattest.Null, 3, 0),
			skillGem(gemBase+1, [3]int32{0, 0, 100}, dattest.Null, true, false, dattest.Null, 0, 3, 1),
			skillGem(gemBase+2, [3]int32{0, 0, 100}, dattest.Null, false, true, 3, dattest.Null, 4, 2),
			skillGem(gemBase+3, [3]int32{0, 0, 100}, dattest.Null, false, true, dattest.Null, 2, 3, 3),
		},
		"GemEffects": {
			[]any{"Fireball", 0, dattest.Null, dattest.Null, dattest.Null, []int{0, 1}, uint32(5)},
			[]any{"VaalFireball", 1, dattest.Null, 0, dattest.Null, []int{1}, uint32(5)},
			[]any{"SupportEcho", 2, 3, dattest.Null, 3, []int(nil), uint32(1)},
			[]any{"SupportAwakenedEcho", 4, dattest.Null, dattest.Null, dattest.Null, []int(nil), uint32(5)},
		},
		"GrantedEffects": {
			grantedEffect("Fireball", 750, 0, 0),
			grantedEffect("VaalFireball", 750, 1, 1, 0),
			// Spell AND Projectile, or Minion
			supportEffect("SupportEcho", []int{0, 1, 2, 3}, []int{1}, []int{3}, []int{0}, []int{0, 1}, 2),
			supportEffect("SupportEchoHardMode", []int{0}, nil, nil, nil, nil, 2),
			supportEffect("SupportAwakenedEcho", []int{0}, nil, nil, nil, nil, 2),
			// granted by no gem
			[]any{"MonsterFireball", false, []int(nil), []int(nil), []int(nil), false, true, int32(1000), 0, []int(nil), []int(nil), 0, []int(nil)},
		},
		"GrantedEffectsPerLevel": {
			effectLevel(0, 2, 2, 6),
			effectLevel(0, 1, 1, 5),
			[]any{2, int32(1), float32(10), int32(130), int32(2), int32(3000), int32(30), int32(1), int32(4), int32(5000), int32(-10),
				[]int32{4, 7}, []int{0, 0}, int32(10), int32(2500), int32(11), int32(1500), int32(800)},
		},
		"GrantedEffectQualityStats": {
			[]any{0, []int{4, 2}, []int32{500, 250}},
		},
		"GrantedEffectStatSets": {
			[]any{"Fireball", []int{2}, []int{3, 3}, []int32{25, 5}, float32(2.5), float32(0.05)},
			[]any{"VaalFireball", []int(nil), []int(nil), []int32(nil), float32(0), float32(0)},
			[]any{"Echo", []int(nil), []int{4}, []int32{-10}, float32(0), float32(0)},
		},
		"GrantedEffectStatSetsPerLevel": {
			[]any{0, int32(20), float32(70), int32(500), int32(0), int32(100), int32(38000),
				[]int32{1883, 2825}, []int32{18}, []int{2}, []int{0, 1}, []int{0, 0}, []int{4}, []uint32{3, 3, 1}, []float32{0.8, 1.2}},
			statSetLevel(0, 1, 1),
			statSetLevel(2, 1, 1),
		},
		"ActiveSkills": {
			activeSkill("fireball", "Fireball", []int{0, 1}, 22, 8, dattest.Null),
			[]any{"vaal_fireball", "Vaal Fireball", []int{0}, []int{0}, uint32(5), []int{3}, int32(2), 0},
			activeSkill("monster_fireball", "", nil, 22, 8, dattest.Null),
		},
	}
}

// skillFiles encodes rows as table files.
func skillFiles(t *testing.T, rows map[string][][]any) fstest.MapFS {
	t.Helper()
	g := fstest.MapFS{}
	for table, rs := range rows {
		g["Data/"+table+".datc64"] = &fstest.MapFile{Data: encode(t, table, rs...)}
	}
	return g
}

func TestSkills(t *testing.T) {
	t.Parallel()
	got, err := extract.Dataset(input(t, game(t), export()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	want := &dataset.Skills{
		Types: []string{"Spell", "Projectile", "AND", "Minion"},
		Gems: []dataset.Gem{
			{
				ID: "Metadata/Items/Gems/SkillGemFireball", Colour: dataset.Blue, DexterityPercent: 40, IntelligencePercent: 60,
				VaalGem: "Metadata/Items/Gems/SkillGemVaalFireball",
				Effects: []dataset.GemEffect{{ID: "Fireball", GrantedEffect: "Fireball", Tags: []string{"fire", "spell"}, Style: dataset.DefaultStyle}},
			},
			{
				ID: "Metadata/Items/Gems/SkillGemVaalFireball", Colour: dataset.Blue, IntelligencePercent: 100, Vaal: true,
				RegularGem: "Metadata/Items/Gems/SkillGemFireball",
				Effects: []dataset.GemEffect{{
					ID: "VaalFireball", GrantedEffect: "VaalFireball", SecondaryGrantedEffect: "Fireball",
					Tags: []string{"spell"}, Style: dataset.DefaultStyle,
				}},
			},
			{
				ID: "Metadata/Items/Gems/SupportGemEcho", Colour: dataset.White, IntelligencePercent: 100, Support: true,
				AwakenedGem: "Metadata/Items/Gems/SupportGemAwakenedEcho",
				Effects: []dataset.GemEffect{{
					ID: "SupportEcho", GrantedEffect: "SupportEcho", HardmodeGrantedEffect: "SupportEchoHardMode",
					HardmodeSecondaryGrantedEffect: "SupportEchoHardMode", Style: dataset.TransfiguredX,
				}},
			},
			{
				ID: "Metadata/Items/Gems/SupportGemAwakenedEcho", Colour: dataset.Blue, IntelligencePercent: 100, Support: true,
				RegularGem: "Metadata/Items/Gems/SupportGemEcho",
				Effects:    []dataset.GemEffect{{ID: "SupportAwakenedEcho", GrantedEffect: "SupportAwakenedEcho", Style: dataset.DefaultStyle}},
			},
		},
		GrantedEffects: []dataset.GrantedEffect{
			{
				ID: "Fireball", ActiveSkill: "fireball", CastTime: 750, StatSets: []string{"Fireball"},
				QualityStats: []dataset.StatValue{{Stat: 4, Value: 500}, {Stat: 2, Value: 250}},
				// sorted by level
				Levels: []dataset.EffectLevel{
					{Level: 1, PlayerLevel: 1, Costs: []dataset.Cost{{Type: "Mana", Amount: 5}}, CostMultiplier: 100},
					{Level: 2, PlayerLevel: 2, Costs: []dataset.Cost{{Type: "Mana", Amount: 6}}, CostMultiplier: 100},
				},
			},
			{ID: "VaalFireball", ActiveSkill: "vaal_fireball", CastTime: 750, StatSets: []string{"VaalFireball", "Fireball"}},
			{
				ID: "SupportEcho", Support: true, StatSets: []string{"Echo"},
				AllowedTypes: []uint16{0, 1, 2, 3}, AddedTypes: []uint16{1}, ExcludedTypes: []uint16{3}, AddedMinionTypes: []uint16{0},
				SupportWeaponRestrictions: []string{"Body Armour", "Currency"}, SupportsGemsOnly: true,
				Levels: []dataset.EffectLevel{{
					Level: 1, PlayerLevel: 10, Costs: []dataset.Cost{{Type: "Mana", Amount: 4}, {Type: "Mana", Amount: 7}},
					CostMultiplier: 130, StoredUses: 2, Cooldown: 3000, VaalSouls: 30, VaalStoredUses: 1, CooldownGroup: 4,
					SoulGainPrevention: 5000, AttackSpeedMultiplier: -10, ManaReservationFlat: 10, ManaReservationPercent: 2500,
					LifeReservationFlat: 11, LifeReservationPercent: 1500, AttackTime: 800,
				}},
			},
			{ID: "SupportEchoHardMode", Support: true, StatSets: []string{"Echo"}, AllowedTypes: []uint16{0}, SupportsGemsOnly: true},
			{ID: "SupportAwakenedEcho", Support: true, StatSets: []string{"Echo"}, AllowedTypes: []uint16{0}, SupportsGemsOnly: true},
			{ID: "MonsterFireball", ActiveSkill: "fireball", CastTime: 1000, StatSets: []string{"Fireball"}, CannotBeSupported: true},
		},
		StatSets: []dataset.StatSet{
			{
				ID: "Fireball", ImplicitStats: []uint32{2}, BaseEffectiveness: 2.5, IncrementalEffectiveness: 0.05,
				// game order, repeats kept
				ConstantStats: []dataset.StatValue{{Stat: 3, Value: 25}, {Stat: 3, Value: 5}},
				Levels: []dataset.StatSetLevel{
					{Level: 1, PlayerLevel: 1},
					{
						Level: 20, PlayerLevel: 70, SpellCritChance: 500, BaseMultiplier: 100, DamageEffectiveness: 38000,
						FloatStats: []dataset.FloatStat{
							{Stat: 0, Value: 0.8, Resolved: 1883, Interpolation: dataset.EffectivenessInterpolation, Base: "DamageValue"},
							{Stat: 1, Value: 1.2, Resolved: 2825, Interpolation: dataset.EffectivenessInterpolation, Base: "DamageValue"},
						},
						Stats: []dataset.LevelStat{{Stat: 4, Value: 18, Interpolation: dataset.NoInterpolation}},
						Flags: []uint32{2},
					},
				},
			},
			{ID: "VaalFireball"},
			{ID: "Echo", ConstantStats: []dataset.StatValue{{Stat: 4, Value: -10}}, Levels: []dataset.StatSetLevel{{Level: 1, PlayerLevel: 1}}},
		},
		ActiveSkills: []dataset.ActiveSkill{
			{ID: "fireball", Types: []uint16{0, 1}, Totem: 22, Mine: 8},
			{ID: "vaal_fireball", Types: []uint16{0}, MinionTypes: []uint16{3}, WeaponRestrictions: []string{"Body Armour"}, Totem: 5, Mine: 2, TransfigureBase: "fireball"},
			{ID: "monster_fireball", Totem: 22, Mine: 8},
		},
	}
	if diff := cmp.Diff(want, got.Skills, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Skills mismatch (-want +got):\n%s", diff)
	}
	// A gem or skill without a name has no entry.
	wantGems := []dataset.Name{
		{ID: "Metadata/Items/Gems/SkillGemFireball", Name: "Fireball"},
		{ID: "Metadata/Items/Gems/SkillGemVaalFireball", Name: "Vaal Fireball"},
		{ID: "Metadata/Items/Gems/SupportGemEcho", Name: "Echo Support"},
	}
	wantSkills := []dataset.Name{{ID: "fireball", Name: "Fireball"}, {ID: "vaal_fireball", Name: "Vaal Fireball"}}
	if diff := cmp.Diff(wantGems, got.Texts[0].GemNames); diff != "" {
		t.Errorf("gem names mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantSkills, got.Texts[0].SkillNames); diff != "" {
		t.Errorf("skill names mismatch (-want +got):\n%s", diff)
	}
	if _, err := dataset.Write(t.TempDir(), got); err != nil {
		t.Errorf("Write: %v", err)
	}
}

func TestSkillsErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		edit  func(rows map[string][][]any)
		game  func(fstest.MapFS)
		error string
	}{
		{name: "missing table", game: func(g fstest.MapFS) { delete(g, "Data/GemEffects.datc64") }, error: "GemEffects"},
		{name: "gem without base item", edit: func(r map[string][][]any) { r["SkillGems"][0][0] = dattest.Null }, error: "SkillGems row 0: null BaseItemTypes reference"},
		{name: "base item out of range", edit: func(r map[string][][]any) { r["SkillGems"][0][0] = 99 }, error: "SkillGems row 0: BaseItemTypes row 99 out of range"},
		{name: "regular gem out of range", edit: func(r map[string][][]any) { r["SkillGems"][1][8] = 9 }, error: "SkillGems row 1: row reference 9 outside table"},
		{name: "gem effect out of range", edit: func(r map[string][][]any) { r["SkillGems"][0][10] = []int{9} }, error: "SkillGems row 0: GemEffects row 9 out of range"},
		{name: "colour out of range", edit: func(r map[string][][]any) { r["SkillGems"][0][9] = int32(256) }, error: "column GemColour: 256 out of range"},
		{name: "granted effect out of range", edit: func(r map[string][][]any) { r["GemEffects"][0][1] = 9 }, error: "GemEffects row 0: GrantedEffects row 9 out of range"},
		{name: "null gem tag", edit: func(r map[string][][]any) { r["GemEffects"][0][5] = []int{dattest.Null} }, error: "null GemTags reference"},
		{name: "style out of range", edit: func(r map[string][][]any) { r["GemEffects"][0][6] = uint32(256) }, error: "column ItemColor: 256 out of range"},
		{name: "skill type out of range", edit: func(r map[string][][]any) { r["ActiveSkills"][0][2] = []int{9} }, error: "ActiveSkills row 0: ActiveSkillType row 9 out of range"},
		{name: "totem out of range", edit: func(r map[string][][]any) { r["ActiveSkills"][0][4] = uint32(1 << 31) }, error: "column SkillTotemId"},
		{name: "no stat set", edit: func(r map[string][][]any) { r["GrantedEffects"][0][11] = dattest.Null }, error: "GrantedEffects row 0: null GrantedEffectStatSets reference"},
		{name: "weapon restriction out of range", edit: func(r map[string][][]any) { r["GrantedEffects"][2][10] = []int{99} }, error: "GrantedEffects row 2: ItemClasses row 99 out of range"},
		{name: "level of unknown effect", edit: func(r map[string][][]any) { r["GrantedEffectsPerLevel"][0][0] = 9 }, error: "GrantedEffectsPerLevel row 0: GrantedEffects row 9 out of range"},
		{name: "player level not whole", edit: func(r map[string][][]any) { r["GrantedEffectsPerLevel"][0][2] = float32(1.5) }, error: "1.5 is not a whole number"},
		{name: "cost types without amounts", edit: func(r map[string][][]any) { r["GrantedEffectsPerLevel"][0][11] = []int32(nil) }, error: "1 cost types, 0 amounts"},
		{name: "quality stats without values", edit: func(r map[string][][]any) { r["GrantedEffectQualityStats"][0][2] = []int32{1} }, error: "2 stats, 1 values"},
		{name: "quality of unknown effect", edit: func(r map[string][][]any) { r["GrantedEffectQualityStats"][0][0] = 9 }, error: "GrantedEffectQualityStats row 0: GrantedEffects row 9 out of range"},
		{name: "constant stat out of range", edit: func(r map[string][][]any) { r["GrantedEffectStatSets"][0][2] = []int{3, 9} }, error: "GrantedEffectStatSets row 0: Stats row 9 out of range"},
		{name: "level of unknown stat set", edit: func(r map[string][][]any) { r["GrantedEffectStatSetsPerLevel"][1][0] = 9 }, error: "GrantedEffectStatSetsPerLevel row 1: GrantedEffectStatSets row 9 out of range"},
		{name: "interpolation per stat missing", edit: func(r map[string][][]any) { r["GrantedEffectStatSetsPerLevel"][0][13] = []uint32{3, 3} }, error: "2 interpolations"},
		{name: "float stat without base", edit: func(r map[string][][]any) { r["GrantedEffectStatSetsPerLevel"][0][11] = []int{0} }, error: "1 bases"},
		{name: "interpolation out of range", edit: func(r map[string][][]any) { r["GrantedEffectStatSetsPerLevel"][0][13] = []uint32{3, 3, 256} }, error: "column StatInterpolations: 256 out of range"},
		{name: "level stat out of range", edit: func(r map[string][][]any) { r["GrantedEffectStatSetsPerLevel"][0][12] = []int{9} }, error: "GrantedEffectStatSetsPerLevel row 0: Stats row 9 out of range"},
		{name: "flag out of range", edit: func(r map[string][][]any) { r["GrantedEffectStatSetsPerLevel"][0][9] = []int{9} }, error: "GrantedEffectStatSetsPerLevel row 0: Stats row 9 out of range"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g, rows := game(t), skillRows()
			if testCase.edit != nil {
				testCase.edit(rows)
				maps.Copy(g, skillFiles(t, rows))
			}
			if testCase.game != nil {
				testCase.game(g)
			}
			_, err := extract.Dataset(input(t, g, export()))
			if err == nil || !strings.Contains(err.Error(), testCase.error) {
				t.Errorf("Dataset: got error %v, want one containing %q", err, testCase.error)
			}
		})
	}
}

// values27 returns the value maps edition/poe1/values.json gives for 3.27.0.10.
func values27(t *testing.T) []extract.ColumnValues {
	t.Helper()
	e, err := edition.Load("poe1")
	if err != nil {
		t.Fatal(err)
	}
	values, err := e.ValuesFor("3.27.0.10")
	if err != nil {
		t.Fatal(err)
	}
	var out []extract.ColumnValues
	for _, v := range values {
		out = append(out, extract.ColumnValues{Table: v.Table, Column: v.Column, Map: v.Map})
	}
	return out
}

// TestSkillsValues builds gems with every style, numbered as 3.27 numbers them and read through
// the shipped 3.27 map, and as the schema numbers them.
func TestSkillsValues(t *testing.T) {
	t.Parallel()
	withStyles := func(styles ...uint32) fstest.MapFS {
		rows := skillRows()
		for i, r := range rows["GemEffects"] {
			r[6] = styles[i]
		}
		g := game(t)
		maps.Copy(g, skillFiles(t, rows))
		return g
	}
	in := input(t, withStyles(1, 2, 3, 4), export())
	in.Values = values27(t)
	got, err := extract.Dataset(in)
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	want, err := extract.Dataset(input(t, withStyles(1, 2, 3, 5), export()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	if diff := cmp.Diff(want.Skills.Gems, got.Skills.Gems, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Gems mismatch (-want +got):\n%s", diff)
	}
	var styles []dataset.GemStyle
	for _, g := range got.Skills.Gems {
		for _, e := range g.Effects {
			styles = append(styles, e.Style)
		}
	}
	if w := []dataset.GemStyle{dataset.TransfiguredX, dataset.TransfiguredY, dataset.Trarthan, dataset.DefaultStyle}; !slices.Equal(styles, w) {
		t.Errorf("got styles %v, want %v", styles, w)
	}
}

func TestSkillsValuesErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		values []extract.ColumnValues
		error  string
	}{
		// The default style of the game files is 5, which the map lacks.
		{name: "value not in map", values: values27(t), error: "column ItemColor: 5 not in the edition's values"},
		{name: "column not mapped", values: []extract.ColumnValues{{Table: "Mods", Column: "GameMode", Map: map[uint32]uint32{1: 1}}}, error: "values of Mods.GameMode: column not read through a value map"},
		{name: "column twice", values: append(values27(t), values27(t)...), error: "values of GemEffects.ItemColor given twice"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			in := input(t, game(t), export())
			in.Values = testCase.values
			_, err := extract.Dataset(in)
			if err == nil || !strings.Contains(err.Error(), testCase.error) {
				t.Errorf("Dataset: got error %v, want one containing %q", err, testCase.error)
			}
		})
	}
}
