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
		{"name": "InheritsFrom", "type": "string"}]},
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
		{"name": "GameMode", "type": "enumrow"}]}`

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
	return append(r, m.gameMode)
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
		{id: "FlaskInstantRecovery", name: "Bubbling", domain: 2, generation: 1, level: 1, typ: 0},
	}
}

// itemGame adds the item tables to g: a body armour, a shield, an axe and a life flask, a
// currency base that is left out, and four mods of which the monster mod is left out.
// itemBases are the BaseItemTypes rows of items; gemBases follow them.
func itemBases() [][]any {
	return [][]any{
		{"Metadata/Items/Armours/BodyArmours/BodyStr15", 0, "Astral Plate", []int{2}, []int{tagBodyArmour},
			"Metadata/Items/Armours/BodyArmours/AbstractBodyArmour"},
		{"Metadata/Items/Currency/CurrencyRerollRare", 1, "Chaos Orb", []int{1}, []int(nil), "Metadata/Items/Currency/StackableCurrency"},
		{"Metadata/Items/Armours/Shields/ShieldInt16", 2, "Titanium Spirit Shield", []int(nil), []int{tagShield},
			"Metadata/Items/Armours/Shields/AbstractShield"},
		{"Metadata/Items/Weapons/TwoHandWeapons/TwoHandAxes/TwoHandAxe15", 3, "Sundering Axe", []int(nil), []int(nil),
			"Metadata/Items/Weapons/TwoHandWeapons/TwoHandAxes/AbstractTwoHandAxe"},
		{"Metadata/Items/Flasks/FlaskLife11", 4, "Divine Life Flask", []int(nil), []int(nil), "Metadata/Items/Flasks/AbstractLifeFlask"},
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
			{ID: "Metadata/Items/Armours/BodyArmours/BodyStr15", Class: "Body Armour",
				Implicits: []string{"AllResistancesImplicitArmour1"}, Tags: []string{"body_armour", "armour", "default"},
				Armour: &dataset.ArmourValues{Armour: dataset.ValueRange{Min: 711, Max: 782}, MovementSpeed: -5}},
			{ID: "Metadata/Items/Armours/Shields/ShieldInt16", Class: "Shield", Tags: []string{"shield", "armour", "default"},
				Armour: &dataset.ArmourValues{EnergyShield: dataset.ValueRange{Min: 67, Max: 77}, MovementSpeed: -3}, Block: 25},
			{ID: "Metadata/Items/Weapons/TwoHandWeapons/TwoHandAxes/TwoHandAxe15", Class: "Two Hand Axe",
				Tags:   []string{"axe", "two_hand_weapon", "twohand", "weapon", "default"},
				Weapon: &dataset.WeaponValues{PhysicalDamage: dataset.ValueRange{Min: 74, Max: 155}, AttackTime: 769, CriticalChance: 500, Range: 13}},
			{ID: "Metadata/Items/Flasks/FlaskLife11", Class: "LifeFlask", Tags: []string{"life_flask", "flask", "default"},
				Flask: &dataset.FlaskValues{Kind: dataset.LifeFlask, Life: 2400, Duration: 35, MaxCharges: 45, ChargesPerUse: 15}},
		},
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
		{ID: "FlaskInstantRecovery", Domain: 2, Generation: 1, Level: 1, Type: "IncreasedLife"},
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
		return []any{fmt.Sprintf("Base%d", id), class, "", []int(nil), tags, template}
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
			[]any{"Metadata/Items/Armours/BodyArmours/BodyStr15", 0, "Astral Plate", []int{1}, []int(nil), "Metadata/Items/Item"}),
			"implicit mod row 1 is not an item mod"},
		{"base class out of range", set("BaseItemTypes", []any{"X", 9, "X", []int(nil), []int(nil), "Metadata/Items/Item"}), "ItemClasses row 9 out of range"},
		{"base tag out of range", set("BaseItemTypes", []any{"X", 0, "X", []int(nil), []int{99}, "Metadata/Items/Item"}), "Tags row 99 out of range"},
		{"base without template", set("BaseItemTypes", []any{"X", 0, "X", []int(nil), []int(nil), ""}), "BaseItemTypes row 0: no item template"},
		{"missing template", set("BaseItemTypes", []any{"X", 0, "X", []int(nil), []int(nil), "Metadata/Items/Nope"}),
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
		{"mod stat out of range", setMods(func(m []modRow) { m[0].stats[0].stat = 9 }), "Mods row 0: stat row 9 out of range"},
		{"mod type out of range", setMods(func(m []modRow) { m[0].typ = 9 }), "ModType row 9 out of range"},
		{"game mode out of range", setMods(func(m []modRow) { m[0].gameMode = 1 << 31 }), "Mods row 0: column GameMode: 2147483648 out of range"},
		{"mod family out of range", setMods(func(m []modRow) { m[0].families = []int{9} }), "ModFamily row 9 out of range"},
		{"spawn weights without tags", setMods(func(m []modRow) { m[0].spawn = []int32{1} }), "spawn weights: 2 tags, 1 weights"},
		{"generation tag out of range", setMods(func(m []modRow) { m[0].genTags, m[0].gen = []int{99}, []int32{1} }),
			"generation weights: Tags row 99 out of range"},
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
