package extract_test

import (
	"fmt"
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

// itemTables are the item tables of the test schema, with the columns items reads.
const itemTables = `
	{"name": "Tags", "columns": [{"name": "Id", "type": "string"}]},
	{"name": "ItemClasses", "columns": [{"name": "Id", "type": "string"}, {"name": "Name", "type": "string"}]},
	{"name": "BaseItemTypes", "columns": [
		{"name": "Id", "type": "string"},
		{"name": "ItemClassesKey", "type": "foreignrow", "references": {"table": "ItemClasses"}},
		{"name": "Name", "type": "string"},
		{"name": "Implicit_ModsKeys", "type": "foreignrow", "array": true, "references": {"table": "Mods"}},
		{"name": "TagsKeys", "type": "foreignrow", "array": true, "references": {"table": "Tags"}},
		{"name": "InheritsFrom", "type": "string"},
		{"name": "ModDomain", "type": "enumrow"}]},
	{"name": "CraftingItemClassCategories", "columns": [
		{"name": "ItemClasses", "type": "foreignrow", "array": true, "references": {"table": "ItemClasses"}}]},
	{"name": "CraftingBenchOptions", "columns": [
		{"name": "AddMod", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "AddEnchantment", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "ItemClasses", "type": "foreignrow", "array": true, "references": {"table": "ItemClasses"}},
		{"name": "CraftingItemClassCategories", "type": "foreignrow", "array": true, "references": {"table": "CraftingItemClassCategories"}},
		{"name": "IsDisabled", "type": "bool"}]},
	{"name": "Essences", "columns": [
		{"name": "Helmet_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "BodyArmour_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "Boots_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "Gloves_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "Shield_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "Belt_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "Amulet_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "Ring_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "Bow_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "Wand_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "Staff_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "Claw_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "Dagger_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "Sceptre_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "OneHandSword_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "OneHandThrustingSword_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "OneHandAxe_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "OneHandMace_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "TwoHandSword_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "TwoHandAxe_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "TwoHandMace_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}},
		{"name": "Display_Quiver_ModsKey", "type": "foreignrow", "references": {"table": "Mods"}}]},
	{"name": "ArmourTypes", "columns": [
		{"name": "BaseItemTypesKey", "type": "foreignrow", "references": {"table": "BaseItemTypes"}},
		{"name": "ArmourMin", "type": "i32"}, {"name": "ArmourMax", "type": "i32"},
		{"name": "EvasionMin", "type": "i32"}, {"name": "EvasionMax", "type": "i32"},
		{"name": "EnergyShieldMin", "type": "i32"}, {"name": "EnergyShieldMax", "type": "i32"},
		{"name": "IncreasedMovementSpeed", "type": "i32"},
		{"name": "WardMin", "type": "i32"}, {"name": "WardMax", "type": "i32"}]},
	{"name": "ShieldTypes", "columns": [
		{"name": "BaseItemTypesKey", "type": "foreignrow", "references": {"table": "BaseItemTypes"}},
		{"name": "Block", "type": "i32"}]},
	{"name": "WeaponTypes", "columns": [
		{"name": "BaseItemTypesKey", "type": "foreignrow", "references": {"table": "BaseItemTypes"}},
		{"name": "Critical", "type": "i32"}, {"name": "Speed", "type": "i32"},
		{"name": "DamageMin", "type": "i32"}, {"name": "DamageMax", "type": "i32"}, {"name": "RangeMax", "type": "i32"}]},
	{"name": "Flasks", "columns": [
		{"name": "BaseItemTypesKey", "type": "foreignrow", "references": {"table": "BaseItemTypes"}},
		{"name": "Type", "type": "enumrow"}, {"name": "LifePerUse", "type": "i32"}, {"name": "ManaPerUse", "type": "i32"},
		{"name": "RecoveryTime", "type": "i32"}, {"name": "RecoveryTime2", "type": "i32"},
		{"name": "BuffDefinitionsKey", "type": "foreignrow", "references": {"table": "BuffDefinitions"}},
		{"name": "BuffStatValues", "type": "i32", "array": true}]},
	{"name": "ComponentCharges", "columns": [
		{"name": "BaseItemTypesKey", "type": "string"}, {"name": "MaxCharges", "type": "i32"}, {"name": "PerCharge", "type": "i32"}]},
	{"name": "ModType", "columns": [{"name": "Name", "type": "string"}]},
	{"name": "ModFamily", "columns": [{"name": "Id", "type": "string"}]},
	{"name": "Mods", "columns": [
		{"name": "Id", "type": "string"}, {"name": "Name", "type": "string"},
		{"name": "Domain", "type": "enumrow"}, {"name": "GenerationType", "type": "enumrow"}, {"name": "Level", "type": "i32"},
		{"name": "ModTypeKey", "type": "foreignrow", "references": {"table": "ModType"}},
		{"name": "Families", "type": "foreignrow", "array": true, "references": {"table": "ModFamily"}},
		{"name": "ImplicitTagsKeys", "type": "foreignrow", "array": true, "references": {"table": "Tags"}},
		{"name": "TagsKeys", "type": "foreignrow", "array": true, "references": {"table": "Tags"}},
		{"name": "SpawnWeight_TagsKeys", "type": "foreignrow", "array": true, "references": {"table": "Tags"}},
		{"name": "SpawnWeight_Values", "type": "i32", "array": true},
		{"name": "GenerationWeight_TagsKeys", "type": "foreignrow", "array": true, "references": {"table": "Tags"}},
		{"name": "GenerationWeight_Values", "type": "i32", "array": true},
		{"name": "StatsKey1", "type": "foreignrow", "references": {"table": "Stats"}},
		{"name": "StatsKey2", "type": "foreignrow", "references": {"table": "Stats"}},
		{"name": "StatsKey3", "type": "foreignrow", "references": {"table": "Stats"}},
		{"name": "StatsKey4", "type": "foreignrow", "references": {"table": "Stats"}},
		{"name": "StatsKey5", "type": "foreignrow", "references": {"table": "Stats"}},
		{"name": "StatsKey6", "type": "foreignrow", "references": {"table": "Stats"}},
		{"name": "StatsKey7", "type": "foreignrow", "references": {"table": "Stats"}},
		{"name": "StatsKey8", "type": "foreignrow", "references": {"table": "Stats"}},
		{"name": "Stat1Min", "type": "i32"}, {"name": "Stat1Max", "type": "i32"},
		{"name": "Stat2Min", "type": "i32"}, {"name": "Stat2Max", "type": "i32"},
		{"name": "Stat3Min", "type": "i32"}, {"name": "Stat3Max", "type": "i32"},
		{"name": "Stat4Min", "type": "i32"}, {"name": "Stat4Max", "type": "i32"},
		{"name": "Stat5Min", "type": "i32"}, {"name": "Stat5Max", "type": "i32"},
		{"name": "Stat6Min", "type": "i32"}, {"name": "Stat6Max", "type": "i32"},
		{"name": "Stat7Min", "type": "i32"}, {"name": "Stat7Max", "type": "i32"},
		{"name": "Stat8Min", "type": "i32"}, {"name": "Stat8Max", "type": "i32"},
		{"name": "HasHardmodeStatValues", "type": "bool"},
		{"name": "Stat1MinHardmode", "type": "i32"}, {"name": "Stat1MaxHardmode", "type": "i32"},
		{"name": "Stat2MinHardmode", "type": "i32"}, {"name": "Stat2MaxHardmode", "type": "i32"},
		{"name": "Stat3MinHardmode", "type": "i32"}, {"name": "Stat3MaxHardmode", "type": "i32"},
		{"name": "Stat4MinHardmode", "type": "i32"}, {"name": "Stat4MaxHardmode", "type": "i32"},
		{"name": "Stat5MinHardmode", "type": "i32"}, {"name": "Stat5MaxHardmode", "type": "i32"},
		{"name": "Stat6MinHardmode", "type": "i32"}, {"name": "Stat6MaxHardmode", "type": "i32"},
		{"name": "Stat7MinHardmode", "type": "i32"}, {"name": "Stat7MaxHardmode", "type": "i32"},
		{"name": "Stat8MinHardmode", "type": "i32"}, {"name": "Stat8MaxHardmode", "type": "i32"},
		{"name": "GameMode", "type": "enumrow"},
		{"name": "GrantedEffectsPerLevelKeys", "type": "foreignrow", "array": true, "references": {"table": "GrantedEffectsPerLevel"}},
		{"name": "GrantedEffectRuthless", "type": "foreignrow", "array": true, "references": {"table": "GrantedEffectsPerLevel"}}]}`

// modStat is one stat slot of a Mods row: a Stats row, its range and its Ruthless range.
type modStat struct {
	stat                 int
	min, max, rMin, rMax int32
}

// modRow is a Mods row; stats fill the first slots and the rest are empty.
type modRow struct {
	id, name                 string
	domain, generation       uint32
	level                    int32
	typ                      int
	families, implicit, tags []int
	spawnTags                []int
	spawn                    []int32
	genTags                  []int
	gen                      []int32
	ruthless                 bool
	stats                    []modStat
	gameMode                 uint32
	// grants and ruthlessGrants are GrantedEffectsPerLevel rows.
	grants, ruthlessGrants []int
}

func (m modRow) row() []any {
	r := []any{m.id, m.name, m.domain, m.generation, m.level, m.typ, m.families, m.implicit, m.tags,
		m.spawnTags, m.spawn, m.genTags, m.gen}
	slots := make([]modStat, 8)
	for i := range slots {
		slots[i].stat = dattest.Null
	}
	copy(slots, m.stats)
	for _, s := range slots {
		r = append(r, s.stat)
	}
	for _, s := range slots {
		r = append(r, s.min, s.max)
	}
	r = append(r, m.ruthless)
	for _, s := range slots {
		r = append(r, s.rMin, s.rMax)
	}
	return append(r, m.gameMode, m.grants, m.ruthlessGrants)
}

func encodeMods(t *testing.T, mods ...modRow) []byte {
	t.Helper()
	rows := make([][]any, len(mods))
	for i, m := range mods {
		rows[i] = m.row()
	}
	return encode(t, "Mods", rows...)
}

// Tags of the item fixture.
const (
	tagDefault = iota
	tagBodyArmour
	tagLife
	tagShield
	tagStrArmour
	tagSceptre
	tagJewel
	tagWarstaff
)

// itemTags are the Tags rows of the item fixture: those named above, then those only item
// templates name.
var itemTags = []string{"default", "body_armour", "life", "shield", "str_armour", "sceptre", "jewel", "warstaff",
	"armour", "helmet", "weapon", "onehand", "twohand", "one_hand_weapon", "two_hand_weapon", "axe", "dagger",
	"attack_dagger", "wand", "ranged", "staff", "attack_staff", "bow", "ring", "flask", "life_flask",
	"utility_flask", "tincture"}

// itemTemplates are the item templates of the test game by path, without .it: the template each
// extends and its tags. They follow the game's chains.
var itemTemplates = map[string]struct {
	extends string
	tags    []string
}{
	"Metadata/Items/Item":                                                {"nothing", []string{"default"}},
	"Metadata/Items/Equipment":                                           {"Metadata/Items/Item", nil},
	"Metadata/Items/Armours/AbstractArmour":                              {"Metadata/Items/Equipment", []string{"armour"}},
	"Metadata/Items/Armours/BodyArmours/AbstractBodyArmour":              {"Metadata/Items/Armours/AbstractArmour", []string{"body_armour"}},
	"Metadata/Items/Armours/Helmets/AbstractHelmet":                      {"Metadata/Items/Armours/AbstractArmour", []string{"helmet"}},
	"Metadata/Items/Armours/Shields/AbstractShield":                      {"Metadata/Items/Armours/AbstractArmour", []string{"shield"}},
	"Metadata/Items/Weapons/AbstractWeapon":                              {"Metadata/Items/Equipment", []string{"weapon"}},
	"Metadata/Items/Weapons/OneHandWeapons/AbstractOneHandWeapon":        {"Metadata/Items/Weapons/AbstractWeapon", []string{"onehand"}},
	"Metadata/Items/Weapons/OneHandWeapons/OneHandMaces/AbstractSceptre": {"Metadata/Items/Weapons/OneHandWeapons/AbstractOneHandWeapon", []string{"sceptre", "one_hand_weapon"}},
	"Metadata/Items/Weapons/OneHandWeapons/Daggers/AbstractDagger": {"Metadata/Items/Weapons/OneHandWeapons/AbstractOneHandWeapon",
		[]string{"dagger", "attack_dagger", "one_hand_weapon"}},
	"Metadata/Items/Weapons/OneHandWeapons/Wands/AbstractWand": {"Metadata/Items/Weapons/OneHandWeapons/AbstractOneHandWeapon",
		[]string{"wand", "ranged", "one_hand_weapon"}},
	"Metadata/Items/Weapons/TwoHandWeapons/AbstractTwoHandWeapon": {"Metadata/Items/Weapons/AbstractWeapon", []string{"twohand"}},
	"Metadata/Items/Weapons/TwoHandWeapons/TwoHandAxes/AbstractTwoHandAxe": {"Metadata/Items/Weapons/TwoHandWeapons/AbstractTwoHandWeapon",
		[]string{"axe", "two_hand_weapon"}},
	"Metadata/Items/Weapons/TwoHandWeapons/Staves/AbstractWarstaff": {"Metadata/Items/Weapons/TwoHandWeapons/AbstractTwoHandWeapon",
		[]string{"staff", "attack_staff", "two_hand_weapon"}},
	"Metadata/Items/Weapons/TwoHandWeapons/Bows/AbstractBow": {"Metadata/Items/Weapons/TwoHandWeapons/AbstractTwoHandWeapon",
		[]string{"bow", "ranged", "two_hand_weapon"}},
	"Metadata/Items/Rings/AbstractRing":          {"Metadata/Items/Equipment", []string{"ring"}},
	"Metadata/Items/Jewels/AbstractJewel":        {"Metadata/Items/Item", nil},
	"Metadata/Items/Flasks/AbstractFlask":        {"Metadata/Items/Item", []string{"flask"}},
	"Metadata/Items/Flasks/AbstractLifeFlask":    {"Metadata/Items/Flasks/AbstractFlask", []string{"life_flask"}},
	"Metadata/Items/Flasks/AbstractUtilityFlask": {"Metadata/Items/Flasks/AbstractFlask", []string{"utility_flask"}},
	"Metadata/Items/Tinctures/AbstractTincture":  {"Metadata/Items/Item", []string{"tincture"}},
}

// itemTemplate is the text of an item template.
func itemTemplate(extends string, tags ...string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "version 3\nabstract\nextends %q\n\nBase\n{\n", extends)
	for _, tag := range tags {
		fmt.Fprintf(&b, "\ttag = %q\n", tag)
	}
	b.WriteString("}\n")
	return utf16le(b.String())
}

// essenceColumns are the mod columns of the test Essences table, in order.
var essenceColumns = []string{"Helmet_ModsKey", "BodyArmour_ModsKey", "Boots_ModsKey", "Gloves_ModsKey", "Shield_ModsKey", "Belt_ModsKey", "Amulet_ModsKey", "Ring_ModsKey", "Bow_ModsKey", "Wand_ModsKey", "Staff_ModsKey", "Claw_ModsKey", "Dagger_ModsKey", "Sceptre_ModsKey", "OneHandSword_ModsKey", "OneHandThrustingSword_ModsKey", "OneHandAxe_ModsKey", "OneHandMace_ModsKey", "TwoHandSword_ModsKey", "TwoHandAxe_ModsKey", "TwoHandMace_ModsKey", "Display_Quiver_ModsKey"}

// essence is an Essences row with mods, Mods rows by column; other columns are null.
func essence(mods map[string]int) []any {
	row := make([]any, len(essenceColumns))
	for i, c := range essenceColumns {
		row[i] = dattest.Null
		if m, ok := mods[c]; ok {
			row[i] = m
		}
	}
	return row
}

func itemMods() []modRow {
	return []modRow{
		{id: "IncreasedLife1", name: "Healthy", domain: 1, generation: 1, level: 5, typ: 0, families: []int{0}, gameMode: 1,
			implicit: []int{tagLife}, spawnTags: []int{tagShield, tagDefault}, spawn: []int32{0, 1000},
			stats: []modStat{{stat: 1, min: 10, max: 24}}},
		{id: "MonsterLife", domain: 3, generation: 3, typ: 0, stats: []modStat{{stat: 1, min: 1, max: 1}}},
		// a stat slot left empty between two stats, and a Ruthless range
		{id: "AllResistancesImplicitArmour1", domain: 1, generation: 3, level: 1, typ: 1, ruthless: true, gameMode: 2,
			tags: []int{tagBodyArmour}, genTags: []int{tagDefault}, gen: []int32{50},
			stats: []modStat{{stat: 3, min: 8, max: 12, rMin: 4, rMax: 6}, {stat: dattest.Null}, {stat: 4, min: 1, max: 2, rMin: 1, rMax: 1}}},
		{id: "FlaskInstantRecovery", name: "Bubbling", domain: 2, generation: 1, level: 1, typ: 0, grants: []int{1, 0}, ruthlessGrants: []int{0}},
	}
}

// itemGame adds the item tables to g: a body armour, a shield, an axe and a life flask, a
// currency base that is left out, four mods of which the monster mod is left out, a bench
// craft of the life mod and an essence of the resistances mod for body armours.
// itemBases are the BaseItemTypes rows of items; gemBases follow them.
func itemBases() [][]any {
	return [][]any{
		{"Metadata/Items/Armours/BodyArmours/BodyStr15", 0, "Astral Plate", []int{2}, []int{tagBodyArmour},
			"Metadata/Items/Armours/BodyArmours/AbstractBodyArmour", uint32(1)},
		{"Metadata/Items/Currency/CurrencyRerollRare", 1, "Chaos Orb", []int{1}, []int(nil), "Metadata/Items/Currency/StackableCurrency", uint32(43)},
		{"Metadata/Items/Armours/Shields/ShieldInt16", 2, "Titanium Spirit Shield", []int(nil), []int{tagShield},
			"Metadata/Items/Armours/Shields/AbstractShield", uint32(1)},
		{"Metadata/Items/Weapons/TwoHandWeapons/TwoHandAxes/TwoHandAxe15", 3, "Sundering Axe", []int(nil), []int(nil),
			"Metadata/Items/Weapons/TwoHandWeapons/TwoHandAxes/AbstractTwoHandAxe", uint32(1)},
		{"Metadata/Items/Flasks/FlaskLife11", 4, "Divine Life Flask", []int(nil), []int(nil), "Metadata/Items/Flasks/AbstractLifeFlask", uint32(2)},
	}
}

func itemGame(t *testing.T, g fstest.MapFS) {
	t.Helper()
	tagRows := make([][]any, len(itemTags))
	for i, tag := range itemTags {
		tagRows[i] = []any{tag}
	}
	files := map[string][]byte{
		"Tags": encode(t, "Tags", tagRows...),
		"ItemClasses": encode(t, "ItemClasses",
			[]any{"Body Armour", "Body Armours"}, []any{"Currency", "Currency"}, []any{"Shield", "Shields"},
			[]any{"Two Hand Axe", "Two Hand Axes"}, []any{"LifeFlask", "Life Flasks"}),
		"BaseItemTypes": encode(t, "BaseItemTypes", append(itemBases(), gemBases()...)...),
		"ArmourTypes": encode(t, "ArmourTypes",
			[]any{0, int32(711), int32(782), int32(0), int32(0), int32(0), int32(0), int32(-5), int32(0), int32(0)},
			[]any{1, int32(1), int32(1), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0), int32(0)},
			[]any{2, int32(0), int32(0), int32(0), int32(0), int32(67), int32(77), int32(-3), int32(0), int32(0)}),
		"ShieldTypes": encode(t, "ShieldTypes", []any{2, int32(25)}),
		"WeaponTypes": encode(t, "WeaponTypes", []any{3, int32(500), int32(769), int32(74), int32(155), int32(13)}),
		"Flasks":      encode(t, "Flasks", []any{4, uint32(1), int32(2400), int32(0), int32(35), int32(30), dattest.Null, []int32(nil)}),
		"ComponentCharges": encode(t, "ComponentCharges",
			[]any{"Metadata/Items/Flasks/FlaskLife11", int32(45), int32(15)}),
		"ModType":   encode(t, "ModType", []any{"IncreasedLife"}, []any{"AllResistances"}),
		"ModFamily": encode(t, "ModFamily", []any{"IncreasedLife"}),
		"Mods":      encodeMods(t, itemMods()...),
		// a craft adding a mod to body armours, and one adding none
		"CraftingItemClassCategories": encode(t, "CraftingItemClassCategories", []any{[]int{0}}),
		"CraftingBenchOptions": encode(t, "CraftingBenchOptions",
			[]any{0, dattest.Null, []int(nil), []int{0}, false}, []any{dattest.Null, dattest.Null, []int(nil), []int{0}, false}),
		"Essences": encode(t, "Essences", essence(map[string]int{"BodyArmour_ModsKey": 2})),
	}
	for name, b := range files {
		g["Data/"+name+".datc64"] = &fstest.MapFile{Data: b}
	}
	for path, it := range itemTemplates {
		g[path+".it"] = &fstest.MapFile{Data: itemTemplate(it.extends, it.tags...)}
	}
}

func TestItems(t *testing.T) {
	t.Parallel()
	got, err := extract.Dataset(input(t, game(t), export()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	wantItems := &dataset.Items{
		Classes: []dataset.ItemClass{{ID: "Body Armour"}, {ID: "Shield"}, {ID: "Two Hand Axe"}, {ID: "LifeFlask"}},
		Tags:    itemTags,
		Bases: []dataset.ItemBase{
			{ID: "Metadata/Items/Armours/BodyArmours/BodyStr15", Class: "Body Armour", ModDomain: 1,
				Implicits: []string{"AllResistancesImplicitArmour1"}, Tags: []string{"body_armour", "armour", "default"},
				Armour: &dataset.ArmourValues{Armour: dataset.ValueRange{Min: 711, Max: 782}, MovementSpeed: -5}},
			{ID: "Metadata/Items/Armours/Shields/ShieldInt16", Class: "Shield", ModDomain: 1, Tags: []string{"shield", "armour", "default"},
				Armour: &dataset.ArmourValues{EnergyShield: dataset.ValueRange{Min: 67, Max: 77}, MovementSpeed: -3}, Block: 25},
			{ID: "Metadata/Items/Weapons/TwoHandWeapons/TwoHandAxes/TwoHandAxe15", Class: "Two Hand Axe", ModDomain: 1,
				Tags:   []string{"axe", "two_hand_weapon", "twohand", "weapon", "default"},
				Weapon: &dataset.WeaponValues{PhysicalDamage: dataset.ValueRange{Min: 74, Max: 155}, AttackTime: 769, CriticalChance: 500, Range: 13}},
			{ID: "Metadata/Items/Flasks/FlaskLife11", Class: "LifeFlask", ModDomain: 2, Tags: []string{"life_flask", "flask", "default"},
				Flask: &dataset.FlaskValues{Kind: dataset.LifeFlask, Life: 2400, Duration: 35, MaxCharges: 45, ChargesPerUse: 15}},
		},
		Crafting:    true,
		BenchCrafts: []dataset.ModClasses{{Mod: "IncreasedLife1", Classes: []string{"Body Armour"}}},
		EssenceMods: []dataset.ModClasses{{Mod: "AllResistancesImplicitArmour1", Classes: []string{"Body Armour"}}},
	}
	wantMods := []dataset.Mod{
		{ID: "IncreasedLife1", Domain: 1, Generation: 1, Level: 5, Type: "IncreasedLife", Families: []string{"IncreasedLife"}, GameMode: 1,
			Stats: []dataset.ModStat{{Stat: 1, Min: 10, Max: 24, RuthlessMin: 10, RuthlessMax: 24}}, Tags: []string{"life"},
			SpawnWeights: []dataset.TagWeight{{Tag: "shield", Weight: 0}, {Tag: "default", Weight: 1000}}},
		{ID: "AllResistancesImplicitArmour1", Domain: 1, Generation: 3, Level: 1, Type: "AllResistances", GameMode: 2,
			Stats: []dataset.ModStat{
				{Stat: 3, Min: 8, Max: 12, RuthlessMin: 4, RuthlessMax: 6},
				{Stat: 4, Min: 1, Max: 2, RuthlessMin: 1, RuthlessMax: 1},
			},
			AddedTags: []string{"body_armour"}, GenerationWeights: []dataset.TagWeight{{Tag: "default", Weight: 50}}},
		{ID: "FlaskInstantRecovery", Domain: 2, Generation: 1, Level: 1, Type: "IncreasedLife",
			GrantedSkills:         []dataset.GrantedSkill{{Effect: "Fireball", Level: 1}, {Effect: "Fireball", Level: 2}},
			RuthlessGrantedSkills: []dataset.GrantedSkill{{Effect: "Fireball", Level: 2}}},
	}
	opt := cmpopts.EquateEmpty()
	if diff := cmp.Diff(wantItems, got.Items, opt); diff != "" {
		t.Errorf("Items mismatch (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(wantMods, got.Mods, opt); diff != "" {
		t.Errorf("Mods mismatch (-want +got):\n%s", diff)
	}
	text := got.Texts[0]
	wantNames := [][]dataset.Name{
		{{ID: "Body Armour", Name: "Body Armours"}, {ID: "Shield", Name: "Shields"}, {ID: "Two Hand Axe", Name: "Two Hand Axes"}, {ID: "LifeFlask", Name: "Life Flasks"}},
		{
			{ID: "Metadata/Items/Armours/BodyArmours/BodyStr15", Name: "Astral Plate"},
			{ID: "Metadata/Items/Armours/Shields/ShieldInt16", Name: "Titanium Spirit Shield"},
			{ID: "Metadata/Items/Weapons/TwoHandWeapons/TwoHandAxes/TwoHandAxe15", Name: "Sundering Axe"},
			{ID: "Metadata/Items/Flasks/FlaskLife11", Name: "Divine Life Flask"},
		},
		// mods without a name have no entry
		{{ID: "IncreasedLife1", Name: "Healthy"}, {ID: "FlaskInstantRecovery", Name: "Bubbling"}},
	}
	if diff := cmp.Diff(wantNames, [][]dataset.Name{text.ItemClassNames, text.BaseNames, text.ModNames}); diff != "" {
		t.Errorf("names mismatch (-want +got):\n%s", diff)
	}
	if _, err := dataset.Write(t.TempDir(), got); err != nil {
		t.Errorf("Write: %v", err)
	}
}

// TestItemBaseTags checks that a base has its own tags and those of its template chain, for a
// base of each class whose spawn tags come mostly from templates.
func TestItemBaseTags(t *testing.T) {
	t.Parallel()
	g := game(t)
	classes := []string{"Helmet", "Ring", "Jewel", "Sceptre", "Dagger", "Warstaff", "Bow", "Wand", "Tincture"}
	classRows := [][]any{
		{"Body Armour", "Body Armours"}, {"Currency", "Currency"}, {"Shield", "Shields"},
		{"Two Hand Axe", "Two Hand Axes"}, {"LifeFlask", "Life Flasks"},
	}
	for _, c := range classes {
		classRows = append(classRows, []any{c, c})
	}
	g["Data/ItemClasses.datc64"] = &fstest.MapFile{Data: encode(t, "ItemClasses", classRows...)}
	class := func(name string) int { return 5 + slices.Index(classes, name) }
	base := func(id, class int, tags []int, template string) []any {
		return []any{fmt.Sprintf("Base%d", id), class, "", []int(nil), tags, template, uint32(1)}
	}
	g["Data/BaseItemTypes.datc64"] = &fstest.MapFile{Data: encode(t, "BaseItemTypes", slices.Concat(itemBases(), gemBases(), [][]any{
		base(0, class("Helmet"), []int{tagStrArmour}, "Metadata/Items/Armours/Helmets/AbstractHelmet"),
		base(1, class("Ring"), nil, "Metadata/Items/Rings/AbstractRing"),
		base(2, class("Jewel"), []int{tagJewel}, "Metadata/Items/Jewels/AbstractJewel"),
		// sceptre is also the template's: listed once
		base(3, class("Sceptre"), []int{tagSceptre}, "Metadata/Items/Weapons/OneHandWeapons/OneHandMaces/AbstractSceptre"),
		base(4, class("Dagger"), nil, "Metadata/Items/Weapons/OneHandWeapons/Daggers/AbstractDagger"),
		base(5, class("Warstaff"), []int{tagWarstaff}, "Metadata/Items/Weapons/TwoHandWeapons/Staves/AbstractWarstaff"),
		base(6, class("Bow"), nil, "Metadata/Items/Weapons/TwoHandWeapons/Bows/AbstractBow"),
		base(7, class("Wand"), nil, "Metadata/Items/Weapons/OneHandWeapons/Wands/AbstractWand"),
		base(8, class("Tincture"), nil, "Metadata/Items/Tinctures/AbstractTincture"),
	})...)}
	got, err := extract.Dataset(input(t, g, export()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	want := map[string][]string{
		"Metadata/Items/Armours/BodyArmours/BodyStr15":                   {"body_armour", "armour", "default"},
		"Metadata/Items/Armours/Shields/ShieldInt16":                     {"shield", "armour", "default"},
		"Metadata/Items/Weapons/TwoHandWeapons/TwoHandAxes/TwoHandAxe15": {"axe", "two_hand_weapon", "twohand", "weapon", "default"},
		"Metadata/Items/Flasks/FlaskLife11":                              {"life_flask", "flask", "default"},
		"Base0":                                                          {"str_armour", "helmet", "armour", "default"},
		"Base1":                                                          {"ring", "default"},
		"Base2":                                                          {"jewel", "default"},
		"Base3":                                                          {"sceptre", "one_hand_weapon", "onehand", "weapon", "default"},
		"Base4":                                                          {"dagger", "attack_dagger", "one_hand_weapon", "onehand", "weapon", "default"},
		"Base5":                                                          {"warstaff", "staff", "attack_staff", "two_hand_weapon", "twohand", "weapon", "default"},
		"Base6":                                                          {"bow", "ranged", "two_hand_weapon", "twohand", "weapon", "default"},
		"Base7":                                                          {"wand", "ranged", "one_hand_weapon", "onehand", "weapon", "default"},
		"Base8":                                                          {"tincture", "default"},
	}
	tags := map[string][]string{}
	for _, b := range got.Items.Bases {
		tags[b.ID] = b.Tags
	}
	if diff := cmp.Diff(want, tags); diff != "" {
		t.Errorf("base tags mismatch (-want +got):\n%s", diff)
	}
}

// TestItemModDomains checks that a base has its own mod domain, which its class does not decide:
// a jewel and a cluster jewel share the Jewel class.
func TestItemModDomains(t *testing.T) {
	t.Parallel()
	g := game(t)
	g["Data/ItemClasses.datc64"] = &fstest.MapFile{Data: encode(t, "ItemClasses",
		[]any{"Body Armour", "Body Armours"}, []any{"Currency", "Currency"}, []any{"Shield", "Shields"},
		[]any{"Two Hand Axe", "Two Hand Axes"}, []any{"LifeFlask", "Life Flasks"},
		[]any{"Helmet", "Helmets"}, []any{"Jewel", "Jewels"}, []any{"AbyssJewel", "Abyss Jewels"})}
	base := func(id string, class int, domain uint32, template string) []any {
		return []any{id, class, "", []int(nil), []int(nil), template, domain}
	}
	g["Data/BaseItemTypes.datc64"] = &fstest.MapFile{Data: encode(t, "BaseItemTypes", slices.Concat(itemBases(), gemBases(), [][]any{
		base("Metadata/Items/Armours/Helmets/HelmetStr1", 5, 1, "Metadata/Items/Armours/Helmets/AbstractHelmet"),
		base("Metadata/Items/Jewels/JewelStr", 6, 10, "Metadata/Items/Jewels/AbstractJewel"),
		base("Metadata/Items/Jewels/JewelPassiveTreeExpansionLarge", 6, 21, "Metadata/Items/Jewels/AbstractJewel"),
		base("Metadata/Items/Jewels/JewelAbyssMelee", 7, 13, "Metadata/Items/Jewels/AbstractJewel"),
	})...)}
	got, err := extract.Dataset(input(t, g, export()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	want := map[string]int32{
		"Metadata/Items/Armours/BodyArmours/BodyStr15":                   1,
		"Metadata/Items/Armours/Shields/ShieldInt16":                     1,
		"Metadata/Items/Weapons/TwoHandWeapons/TwoHandAxes/TwoHandAxe15": 1,
		"Metadata/Items/Flasks/FlaskLife11":                              2,
		"Metadata/Items/Armours/Helmets/HelmetStr1":                      1,
		"Metadata/Items/Jewels/JewelStr":                                 10,
		"Metadata/Items/Jewels/JewelPassiveTreeExpansionLarge":           21,
		"Metadata/Items/Jewels/JewelAbyssMelee":                          13,
	}
	domains := map[string]int32{}
	for _, b := range got.Items.Bases {
		domains[b.ID] = b.ModDomain
	}
	if diff := cmp.Diff(want, domains); diff != "" {
		t.Errorf("mod domains mismatch (-want +got):\n%s", diff)
	}
}

// TestBenchCrafts checks which crafting bench options become crafts and the classes each
// applies to: by category, by class, or both; outside the kept classes; disabled; adding an
// enchantment; adding nothing.
func TestBenchCrafts(t *testing.T) {
	t.Parallel()
	g := game(t)
	// Rows 5 to 9 follow those of itemGame; Map is not a kept class.
	g["Data/ItemClasses.datc64"] = &fstest.MapFile{Data: encode(t, "ItemClasses",
		[]any{"Body Armour", "Body Armours"}, []any{"Currency", "Currency"}, []any{"Shield", "Shields"},
		[]any{"Two Hand Axe", "Two Hand Axes"}, []any{"LifeFlask", "Life Flasks"},
		[]any{"Boots", "Boots"}, []any{"Amulet", "Amulets"}, []any{"Ring", "Rings"},
		[]any{"One Hand Sword", "One Hand Swords"}, []any{"Map", "Maps"})}
	const lifeFlask, boots, amulet, ring, sword, mapClass = 4, 5, 6, 7, 8, 9
	g["Data/CraftingItemClassCategories.datc64"] = &fstest.MapFile{Data: encode(t, "CraftingItemClassCategories",
		[]any{[]int{boots}}, []any{[]int{amulet}}, []any{[]int{ring}},
		[]any{[]int{sword, 1}}, []any{[]int{3}}, []any{[]int{mapClass}})}
	const catBoots, catAmulet, catRing, catOneHandMelee, catTwoHandMelee, catMap = 0, 1, 2, 3, 4, 5
	// the crafted mods follow the monster mods of game
	mods := itemMods()
	for _, id := range monsterMods {
		mods = append(mods, modRow{id: id, domain: 3, generation: 3, typ: 0})
	}
	first := len(mods)
	mods = append(mods,
		modRow{id: "EinharMasterFireResist1", domain: 9, generation: 2, level: 15, typ: 1},
		modRow{id: "EinharMasterAllResist1", domain: 9, generation: 2, level: 30, typ: 1},
		modRow{id: "EinharMasterLocalIncreasedPhysicalDamage1", domain: 9, generation: 1, level: 15, typ: 0},
		modRow{id: "StrDexIntMasterMapQuantity1", domain: 9, generation: 1, level: 1, typ: 0},
		modRow{id: "DisabledCraft", domain: 9, generation: 1, level: 1, typ: 0},
		modRow{id: "FlaskEnchantmentInjectorOnUsingLifeFlask", domain: 1, generation: 22, level: 1, typ: 0},
	)
	g["Data/Mods.datc64"] = &fstest.MapFile{Data: encodeMods(t, mods...)}
	g["Data/CraftingBenchOptions.datc64"] = &fstest.MapFile{Data: encode(t, "CraftingBenchOptions",
		[]any{first, dattest.Null, []int(nil), []int{catBoots}, false},
		// classes named directly and through categories are listed once, in ItemClasses order
		[]any{first + 1, dattest.Null, []int{amulet}, []int{catRing, catAmulet}, false},
		// the Currency class of the category is not kept
		[]any{first + 2, dattest.Null, []int(nil), []int{catOneHandMelee, catTwoHandMelee}, false},
		[]any{first + 3, dattest.Null, []int{mapClass}, []int{catMap}, false},
		[]any{first + 4, dattest.Null, []int(nil), []int{catBoots}, true},
		[]any{dattest.Null, first + 5, []int{lifeFlask}, []int(nil), false},
		[]any{dattest.Null, dattest.Null, []int(nil), []int{catBoots}, false})}
	got, err := extract.Dataset(input(t, g, export()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	want := []dataset.ModClasses{
		{Mod: "EinharMasterFireResist1", Classes: []string{"Boots"}},
		{Mod: "EinharMasterAllResist1", Classes: []string{"Amulet", "Ring"}},
		{Mod: "EinharMasterLocalIncreasedPhysicalDamage1", Classes: []string{"Two Hand Axe", "One Hand Sword"}},
		{Mod: "FlaskEnchantmentInjectorOnUsingLifeFlask", Classes: []string{"LifeFlask"}},
	}
	if diff := cmp.Diff(want, got.Items.BenchCrafts); diff != "" {
		t.Errorf("bench crafts mismatch (-want +got):\n%s", diff)
	}
	if _, err := dataset.Write(t.TempDir(), got); err != nil {
		t.Errorf("Write: %v", err)
	}
}

// TestEssenceMods checks that each essence mod is listed once with the classes of every column
// and essence that adds it: the quiver column, staff mods on warstaves, and classes not in the
// game left out.
func TestEssenceMods(t *testing.T) {
	t.Parallel()
	g := game(t)
	g["Data/ItemClasses.datc64"] = &fstest.MapFile{Data: encode(t, "ItemClasses",
		[]any{"Body Armour", "Body Armours"}, []any{"Currency", "Currency"}, []any{"Shield", "Shields"},
		[]any{"Two Hand Axe", "Two Hand Axes"}, []any{"LifeFlask", "Life Flasks"},
		[]any{"Warstaff", "Warstaves"}, []any{"Helmet", "Helmets"}, []any{"Quiver", "Quivers"}, []any{"Staff", "Staves"})}
	const life, resists, flask = 0, 2, 3
	g["Data/Essences.datc64"] = &fstest.MapFile{Data: encode(t, "Essences",
		essence(map[string]int{"BodyArmour_ModsKey": life, "Display_Quiver_ModsKey": life, "Staff_ModsKey": resists, "Dagger_ModsKey": flask}),
		essence(map[string]int{"Helmet_ModsKey": life, "Shield_ModsKey": life, "TwoHandAxe_ModsKey": resists}),
		// classes a mod already has are not added again
		essence(map[string]int{"Helmet_ModsKey": life, "BodyArmour_ModsKey": life, "Staff_ModsKey": resists}),
		essence(nil))}
	got, err := extract.Dataset(input(t, g, export()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	want := []dataset.ModClasses{
		{Mod: "IncreasedLife1", Classes: []string{"Body Armour", "Shield", "Helmet", "Quiver"}},
		{Mod: "AllResistancesImplicitArmour1", Classes: []string{"Two Hand Axe", "Warstaff", "Staff"}},
	}
	if diff := cmp.Diff(want, got.Items.EssenceMods); diff != "" {
		t.Errorf("essence mods mismatch (-want +got):\n%s", diff)
	}
}

// TestEssenceColumns checks the item classes each Essences column applies to, with every class
// a column names present in ItemClasses and a mod of its own in each column.
func TestEssenceColumns(t *testing.T) {
	t.Parallel()
	want := map[string][]string{
		"Helmet_ModsKey": {"Helmet"}, "BodyArmour_ModsKey": {"Body Armour"}, "Boots_ModsKey": {"Boots"},
		"Gloves_ModsKey": {"Gloves"}, "Shield_ModsKey": {"Shield"}, "Belt_ModsKey": {"Belt"},
		"Amulet_ModsKey": {"Amulet"}, "Ring_ModsKey": {"Ring"}, "Display_Quiver_ModsKey": {"Quiver"},
		"Bow_ModsKey": {"Bow"}, "Wand_ModsKey": {"Wand"}, "Staff_ModsKey": {"Staff", "Warstaff"},
		"Claw_ModsKey": {"Claw"}, "Dagger_ModsKey": {"Dagger", "Rune Dagger"}, "Sceptre_ModsKey": {"Sceptre"},
		"OneHandSword_ModsKey": {"One Hand Sword"}, "OneHandThrustingSword_ModsKey": {"Thrusting One Hand Sword"},
		"OneHandAxe_ModsKey": {"One Hand Axe"}, "OneHandMace_ModsKey": {"One Hand Mace"},
		"TwoHandSword_ModsKey": {"Two Hand Sword"}, "TwoHandAxe_ModsKey": {"Two Hand Axe"},
		"TwoHandMace_ModsKey": {"Two Hand Mace"},
	}
	if len(want) != len(essenceColumns) {
		t.Fatalf("want covers %d columns, the test schema has %d", len(want), len(essenceColumns))
	}
	g := game(t)
	// the classes of game's bases keep their rows; the column classes follow, sorted
	classes := []string{"Body Armour", "Currency", "Shield", "Two Hand Axe", "LifeFlask"}
	var more []string
	for _, cs := range want {
		for _, c := range cs {
			if !slices.Contains(classes, c) {
				more = append(more, c)
			}
		}
	}
	slices.Sort(more)
	classes = append(classes, more...)
	rows := make([][]any, len(classes))
	for i, c := range classes {
		rows[i] = []any{c, c}
	}
	g["Data/ItemClasses.datc64"] = &fstest.MapFile{Data: encode(t, "ItemClasses", rows...)}
	// the column mods follow the monster mods of game
	mods := itemMods()
	for _, id := range monsterMods {
		mods = append(mods, modRow{id: id, domain: 3, generation: 3, typ: 0})
	}
	cols := map[string]int{}
	for _, c := range essenceColumns {
		cols[c] = len(mods)
		mods = append(mods, modRow{id: c, domain: 1, generation: 1, typ: 0})
	}
	g["Data/Mods.datc64"] = &fstest.MapFile{Data: encodeMods(t, mods...)}
	g["Data/Essences.datc64"] = &fstest.MapFile{Data: encode(t, "Essences", essence(cols))}
	got, err := extract.Dataset(input(t, g, export()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	gotClasses := map[string][]string{}
	for _, m := range got.Items.EssenceMods {
		gotClasses[m.Mod] = m.Classes
	}
	for c := range want {
		slices.SortFunc(want[c], func(a, b string) int { return slices.Index(classes, a) - slices.Index(classes, b) })
	}
	if diff := cmp.Diff(want, gotClasses); diff != "" {
		t.Errorf("classes by column mismatch (-want +got):\n%s", diff)
	}
}

func TestItemsErrors(t *testing.T) {
	t.Parallel()
	setMods := func(edit func([]modRow)) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, g fstest.MapFS) {
			mods := itemMods()
			edit(mods)
			g["Data/Mods.datc64"] = &fstest.MapFile{Data: encodeMods(t, mods...)}
		}
	}
	set := func(table string, rows ...[]any) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, g fstest.MapFS) {
			g["Data/"+table+".datc64"] = &fstest.MapFile{Data: encode(t, table, rows...)}
		}
	}
	testCases := []struct {
		name    string
		game    func(*testing.T, fstest.MapFS)
		wantMsg string
	}{
		{"missing table", func(_ *testing.T, g fstest.MapFS) { delete(g, "Data/Mods.datc64") }, "Mods.datc64"},
		{"implicit not an item mod", set("BaseItemTypes",
			[]any{"Metadata/Items/Armours/BodyArmours/BodyStr15", 0, "Astral Plate", []int{1}, []int(nil), "Metadata/Items/Item", uint32(1)}),
			"implicit mod row 1 is not an item mod"},
		{"base class out of range", set("BaseItemTypes", []any{"X", 9, "X", []int(nil), []int(nil), "Metadata/Items/Item", uint32(1)}), "ItemClasses row 9 out of range"},
		{"base tag out of range", set("BaseItemTypes", []any{"X", 0, "X", []int(nil), []int{99}, "Metadata/Items/Item", uint32(1)}), "Tags row 99 out of range"},
		{"base without template", set("BaseItemTypes", []any{"X", 0, "X", []int(nil), []int(nil), "", uint32(1)}), "BaseItemTypes row 0: no item template"},
		{"missing template", set("BaseItemTypes", []any{"X", 0, "X", []int(nil), []int(nil), "Metadata/Items/Nope", uint32(1)}),
			"item template Metadata/Items/Nope"},
		{"template tag not in Tags", func(_ *testing.T, g fstest.MapFS) {
			g["Metadata/Items/Item.it"] = &fstest.MapFile{Data: itemTemplate("nothing", "unknown")}
		}, `tag "unknown" is not in Tags`},
		{"values for a base out of range", set("ShieldTypes", []any{9, int32(25)}), "ShieldTypes row 0: base row 9 out of range"},
		{"second values row", set("ShieldTypes", []any{2, int32(25)}, []any{2, int32(26)}), "base 2 has a second row"},
		{"flask without charges", set("ComponentCharges"), "has no ComponentCharges row"},
		{"second charges row", set("ComponentCharges",
			[]any{"Metadata/Items/Flasks/FlaskLife11", int32(45), int32(15)}, []any{"Metadata/Items/Flasks/FlaskLife11", int32(45), int32(15)}),
			"has a second row"},
		{"base mod domain out of range", set("BaseItemTypes", []any{"X", 0, "X", []int(nil), []int(nil), "Metadata/Items/Item", uint32(1 << 31)}),
			"BaseItemTypes row 0: column ModDomain: 2147483648 out of range"},
		{"craft of a monster mod", set("CraftingBenchOptions", []any{1, dattest.Null, []int(nil), []int{0}, false}),
			"CraftingBenchOptions row 0: mod row 1 is not an item mod"},
		{"enchantment craft of a monster mod", set("CraftingBenchOptions", []any{dattest.Null, 1, []int(nil), []int{0}, false}),
			"CraftingBenchOptions row 0: mod row 1 is not an item mod"},
		{"craft category out of range", set("CraftingBenchOptions", []any{0, dattest.Null, []int(nil), []int{9}, false}),
			"CraftingBenchOptions row 0: CraftingItemClassCategories row 9 out of range"},
		{"craft class out of range", set("CraftingBenchOptions", []any{0, dattest.Null, []int{9}, []int(nil), false}),
			"CraftingBenchOptions row 0: ItemClasses row 9 out of range"},
		{"essence of a monster mod", set("Essences", essence(map[string]int{"Helmet_ModsKey": 1})),
			"Essences row 0: mod row 1 is not an item mod"},
		{"category class out of range", set("CraftingItemClassCategories", []any{[]int{9}}),
			"CraftingBenchOptions row 0: ItemClasses row 9 out of range"},
		{"mod stat out of range", setMods(func(m []modRow) { m[0].stats[0].stat = 9 }), "Mods row 0: stat row 9 out of range"},
		{"mod type out of range", setMods(func(m []modRow) { m[0].typ = 9 }), "ModType row 9 out of range"},
		{"game mode out of range", setMods(func(m []modRow) { m[0].gameMode = 1 << 31 }), "Mods row 0: column GameMode: 2147483648 out of range"},
		{"mod family out of range", setMods(func(m []modRow) { m[0].families = []int{9} }), "ModFamily row 9 out of range"},
		{"spawn weights without tags", setMods(func(m []modRow) { m[0].spawn = []int32{1} }), "spawn weights: 2 tags, 1 weights"},
		{"generation tag out of range", setMods(func(m []modRow) { m[0].genTags, m[0].gen = []int{99}, []int32{1} }),
			"generation weights: Tags row 99 out of range"},
		{"granted skill out of range", setMods(func(m []modRow) { m[0].grants = []int{9} }), "Mods row 0: GrantedEffectsPerLevel row 9 out of range"},
		{"Ruthless granted skill out of range", setMods(func(m []modRow) { m[0].ruthlessGrants = []int{9} }),
			"Mods row 0: GrantedEffectsPerLevel row 9 out of range"},
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
