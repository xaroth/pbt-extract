package extract

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
)

// itemClasses are the ItemClasses.Id of the classes a character equips: armour, weapons,
// jewellery, quivers, flasks, jewels and tinctures. Their bases go into the items group.
var itemClasses = map[string]bool{
	"Helmet": true, "Body Armour": true, "Gloves": true, "Boots": true, "Shield": true,
	"Amulet": true, "Ring": true, "Belt": true, "Quiver": true,
	"Claw": true, "Dagger": true, "Rune Dagger": true, "Wand": true, "Sceptre": true,
	"One Hand Sword": true, "Thrusting One Hand Sword": true, "One Hand Axe": true, "One Hand Mace": true,
	"Bow": true, "Staff": true, "Warstaff": true, "Two Hand Sword": true, "Two Hand Axe": true, "Two Hand Mace": true,
	"FishingRod": true, "LifeFlask": true, "ManaFlask": true, "HybridFlask": true, "UtilityFlask": true,
	"Jewel": true, "AbyssJewel": true, "Tincture": true,
}

// modDomains are the Mods.Domain values of mods an item of those classes can have, as the
// game numbers them (the ModDomains enumeration, from 1). The other domains hold monster,
// area, map, atlas, league mechanic and similar mods.
var modDomains = map[uint32]bool{
	1:  true, // item: prefixes, suffixes, uniques, implicits, enchantments
	2:  true, // flask
	9:  true, // crafted (crafting bench)
	10: true, // jewel
	13: true, // abyss jewel
	16: true, // delve (fossil)
	21: true, // cluster jewel
	28: true, // unveiled
	34: true, // tincture
	38: true, // graft
}

// removedClass is the ItemClasses.Id of bases the game has taken out, such as grafts from 3.28 on.
const removedClass = "RemovedItem"

type itemClassRow struct {
	ID   string `dat:"Id"`
	Name string `dat:"Name"`
}

type baseRow struct {
	ID        string    `dat:"Id"`
	Class     dat.Ref   `dat:"ItemClassesKey" ref:"ItemClasses"`
	Name      string    `dat:"Name"`
	Implicits []dat.Ref `dat:"Implicit_ModsKeys" ref:"Mods"`
	Tags      []dat.Ref `dat:"TagsKeys" ref:"Tags"`
	// Template is the item template (.it) the base extends.
	Template  string `dat:"InheritsFrom"`
	ModDomain uint32 `dat:"ModDomain"`
}

type benchCraftRow struct {
	Mod         dat.Ref   `dat:"AddMod" ref:"Mods"`
	Enchantment dat.Ref   `dat:"AddEnchantment" ref:"Mods"`
	Classes     []dat.Ref `dat:"ItemClasses" ref:"ItemClasses"`
	Categories  []dat.Ref `dat:"CraftingItemClassCategories" ref:"CraftingItemClassCategories"`
	Disabled    bool      `dat:"IsDisabled"`
}

type craftCategoryRow struct {
	Classes []dat.Ref `dat:"ItemClasses" ref:"ItemClasses"`
}

// essenceRow holds the mod an essence adds to items of each class. The Display_ columns other
// than Display_Quiver_ModsKey repeat these for the essence's tooltip.
type essenceRow struct {
	Helmet       dat.Ref `dat:"Helmet_ModsKey" ref:"Mods"`
	BodyArmour   dat.Ref `dat:"BodyArmour_ModsKey" ref:"Mods"`
	Boots        dat.Ref `dat:"Boots_ModsKey" ref:"Mods"`
	Gloves       dat.Ref `dat:"Gloves_ModsKey" ref:"Mods"`
	Shield       dat.Ref `dat:"Shield_ModsKey" ref:"Mods"`
	Belt         dat.Ref `dat:"Belt_ModsKey" ref:"Mods"`
	Amulet       dat.Ref `dat:"Amulet_ModsKey" ref:"Mods"`
	Ring         dat.Ref `dat:"Ring_ModsKey" ref:"Mods"`
	Bow          dat.Ref `dat:"Bow_ModsKey" ref:"Mods"`
	Wand         dat.Ref `dat:"Wand_ModsKey" ref:"Mods"`
	Staff        dat.Ref `dat:"Staff_ModsKey" ref:"Mods"`
	Claw         dat.Ref `dat:"Claw_ModsKey" ref:"Mods"`
	Dagger       dat.Ref `dat:"Dagger_ModsKey" ref:"Mods"`
	Sceptre      dat.Ref `dat:"Sceptre_ModsKey" ref:"Mods"`
	OneHandSword dat.Ref `dat:"OneHandSword_ModsKey" ref:"Mods"`
	Thrusting    dat.Ref `dat:"OneHandThrustingSword_ModsKey" ref:"Mods"`
	OneHandAxe   dat.Ref `dat:"OneHandAxe_ModsKey" ref:"Mods"`
	OneHandMace  dat.Ref `dat:"OneHandMace_ModsKey" ref:"Mods"`
	TwoHandSword dat.Ref `dat:"TwoHandSword_ModsKey" ref:"Mods"`
	TwoHandAxe   dat.Ref `dat:"TwoHandAxe_ModsKey" ref:"Mods"`
	TwoHandMace  dat.Ref `dat:"TwoHandMace_ModsKey" ref:"Mods"`
	// Quiver is the only column with quiver mods; dat-schema names it as a display column. It
	// is set on every essence that has the other item columns set, and Path of Building's
	// Export/spec.lua reads it as the quiver mod.
	Quiver dat.Ref `dat:"Display_Quiver_ModsKey" ref:"Mods"`
}

// classMods pairs each mod of e with the ItemClasses.Ids it is for. Warstaves take the staff mod
// and rune daggers the dagger mod: the game has no column for them, and they are staves and
// daggers to the game (their bases carry the staff and dagger tags).
func (e essenceRow) classMods() []struct {
	mod     dat.Ref
	classes []string
} {
	return []struct {
		mod     dat.Ref
		classes []string
	}{
		{e.Helmet, []string{"Helmet"}}, {e.BodyArmour, []string{"Body Armour"}}, {e.Boots, []string{"Boots"}},
		{e.Gloves, []string{"Gloves"}}, {e.Shield, []string{"Shield"}}, {e.Belt, []string{"Belt"}},
		{e.Amulet, []string{"Amulet"}}, {e.Ring, []string{"Ring"}}, {e.Quiver, []string{"Quiver"}},
		{e.Bow, []string{"Bow"}}, {e.Wand, []string{"Wand"}}, {e.Staff, []string{"Staff", "Warstaff"}},
		{e.Claw, []string{"Claw"}}, {e.Dagger, []string{"Dagger", "Rune Dagger"}}, {e.Sceptre, []string{"Sceptre"}},
		{e.OneHandSword, []string{"One Hand Sword"}}, {e.Thrusting, []string{"Thrusting One Hand Sword"}},
		{e.OneHandAxe, []string{"One Hand Axe"}}, {e.OneHandMace, []string{"One Hand Mace"}},
		{e.TwoHandSword, []string{"Two Hand Sword"}}, {e.TwoHandAxe, []string{"Two Hand Axe"}},
		{e.TwoHandMace, []string{"Two Hand Mace"}},
	}
}

type armourRow struct {
	Base          dat.Ref `dat:"BaseItemTypesKey" ref:"BaseItemTypes"`
	ArmourMin     int32   `dat:"ArmourMin"`
	ArmourMax     int32   `dat:"ArmourMax"`
	EvasionMin    int32   `dat:"EvasionMin"`
	EvasionMax    int32   `dat:"EvasionMax"`
	ESMin         int32   `dat:"EnergyShieldMin"`
	ESMax         int32   `dat:"EnergyShieldMax"`
	WardMin       int32   `dat:"WardMin"`
	WardMax       int32   `dat:"WardMax"`
	MovementSpeed int32   `dat:"IncreasedMovementSpeed"`
}

type shieldRow struct {
	Base  dat.Ref `dat:"BaseItemTypesKey" ref:"BaseItemTypes"`
	Block int32   `dat:"Block"`
}

type weaponRow struct {
	Base      dat.Ref `dat:"BaseItemTypesKey" ref:"BaseItemTypes"`
	Critical  int32   `dat:"Critical"`
	Speed     int32   `dat:"Speed"`
	DamageMin int32   `dat:"DamageMin"`
	DamageMax int32   `dat:"DamageMax"`
	Range     int32   `dat:"RangeMax"`
}

// flaskRow and chargesRow read the first of two sets of duration and charge columns. The
// second set (RecoveryTime2, MaxCharges2, PerCharge2) differs only in the duration of utility
// flasks, which it makes shorter.
type flaskRow struct {
	Base         dat.Ref `dat:"BaseItemTypesKey" ref:"BaseItemTypes"`
	Type         uint32  `dat:"Type"`
	Life         int32   `dat:"LifePerUse"`
	Mana         int32   `dat:"ManaPerUse"`
	RecoveryTime int32   `dat:"RecoveryTime"`
}

type chargesRow struct {
	Base      string `dat:"BaseItemTypesKey"`
	Max       int32  `dat:"MaxCharges"`
	PerCharge int32  `dat:"PerCharge"`
}

type modTypeRow struct {
	Name string `dat:"Name"`
}

type modRow struct {
	ID                string    `dat:"Id"`
	Name              string    `dat:"Name"`
	Domain            uint32    `dat:"Domain"`
	Generation        uint32    `dat:"GenerationType"`
	GameMode          uint32    `dat:"GameMode"`
	Level             int32     `dat:"Level"`
	Type              dat.Ref   `dat:"ModTypeKey" ref:"ModType"`
	Families          []dat.Ref `dat:"Families" ref:"ModFamily"`
	ImplicitTags      []dat.Ref `dat:"ImplicitTagsKeys" ref:"Tags"`
	Tags              []dat.Ref `dat:"TagsKeys" ref:"Tags"`
	SpawnTags         []dat.Ref `dat:"SpawnWeight_TagsKeys" ref:"Tags"`
	SpawnWeights      []int32   `dat:"SpawnWeight_Values"`
	GenerationTags    []dat.Ref `dat:"GenerationWeight_TagsKeys" ref:"Tags"`
	GenerationWeights []int32   `dat:"GenerationWeight_Values"`
	GrantedSkills     []dat.Ref `dat:"GrantedEffectsPerLevelKeys" ref:"GrantedEffectsPerLevel"`
	RuthlessGrants    []dat.Ref `dat:"GrantedEffectRuthless" ref:"GrantedEffectsPerLevel"`
	Stat1             dat.Ref   `dat:"StatsKey1" ref:"Stats"`
	Stat2             dat.Ref   `dat:"StatsKey2" ref:"Stats"`
	Stat3             dat.Ref   `dat:"StatsKey3" ref:"Stats"`
	Stat4             dat.Ref   `dat:"StatsKey4" ref:"Stats"`
	Stat5             dat.Ref   `dat:"StatsKey5" ref:"Stats"`
	Stat6             dat.Ref   `dat:"StatsKey6" ref:"Stats"`
	Stat7             dat.Ref   `dat:"StatsKey7" ref:"Stats"`
	Stat8             dat.Ref   `dat:"StatsKey8" ref:"Stats"`
	Stat1Min          int32     `dat:"Stat1Min"`
	Stat1Max          int32     `dat:"Stat1Max"`
	Stat2Min          int32     `dat:"Stat2Min"`
	Stat2Max          int32     `dat:"Stat2Max"`
	Stat3Min          int32     `dat:"Stat3Min"`
	Stat3Max          int32     `dat:"Stat3Max"`
	Stat4Min          int32     `dat:"Stat4Min"`
	Stat4Max          int32     `dat:"Stat4Max"`
	Stat5Min          int32     `dat:"Stat5Min"`
	Stat5Max          int32     `dat:"Stat5Max"`
	Stat6Min          int32     `dat:"Stat6Min"`
	Stat6Max          int32     `dat:"Stat6Max"`
	Stat7Min          int32     `dat:"Stat7Min"`
	Stat7Max          int32     `dat:"Stat7Max"`
	Stat8Min          int32     `dat:"Stat8Min"`
	Stat8Max          int32     `dat:"Stat8Max"`
	HasRuthless       bool      `dat:"HasHardmodeStatValues"`
	Stat1RuthlessMin  int32     `dat:"Stat1MinHardmode"`
	Stat1RuthlessMax  int32     `dat:"Stat1MaxHardmode"`
	Stat2RuthlessMin  int32     `dat:"Stat2MinHardmode"`
	Stat2RuthlessMax  int32     `dat:"Stat2MaxHardmode"`
	Stat3RuthlessMin  int32     `dat:"Stat3MinHardmode"`
	Stat3RuthlessMax  int32     `dat:"Stat3MaxHardmode"`
	Stat4RuthlessMin  int32     `dat:"Stat4MinHardmode"`
	Stat4RuthlessMax  int32     `dat:"Stat4MaxHardmode"`
	Stat5RuthlessMin  int32     `dat:"Stat5MinHardmode"`
	Stat5RuthlessMax  int32     `dat:"Stat5MaxHardmode"`
	Stat6RuthlessMin  int32     `dat:"Stat6MinHardmode"`
	Stat6RuthlessMax  int32     `dat:"Stat6MaxHardmode"`
	Stat7RuthlessMin  int32     `dat:"Stat7MinHardmode"`
	Stat7RuthlessMax  int32     `dat:"Stat7MaxHardmode"`
	Stat8RuthlessMin  int32     `dat:"Stat8MinHardmode"`
	Stat8RuthlessMax  int32     `dat:"Stat8MaxHardmode"`
}

// stats returns the mod's stats with their ranges, in column order, leaving out empty slots.
// nStats is the number of rows in Stats.
func (m modRow) stats(nStats int) ([]dataset.ModStat, error) {
	slots := [8]struct {
		stat                 dat.Ref
		min, max, rMin, rMax int32
	}{
		{m.Stat1, m.Stat1Min, m.Stat1Max, m.Stat1RuthlessMin, m.Stat1RuthlessMax},
		{m.Stat2, m.Stat2Min, m.Stat2Max, m.Stat2RuthlessMin, m.Stat2RuthlessMax},
		{m.Stat3, m.Stat3Min, m.Stat3Max, m.Stat3RuthlessMin, m.Stat3RuthlessMax},
		{m.Stat4, m.Stat4Min, m.Stat4Max, m.Stat4RuthlessMin, m.Stat4RuthlessMax},
		{m.Stat5, m.Stat5Min, m.Stat5Max, m.Stat5RuthlessMin, m.Stat5RuthlessMax},
		{m.Stat6, m.Stat6Min, m.Stat6Max, m.Stat6RuthlessMin, m.Stat6RuthlessMax},
		{m.Stat7, m.Stat7Min, m.Stat7Max, m.Stat7RuthlessMin, m.Stat7RuthlessMax},
		{m.Stat8, m.Stat8Min, m.Stat8Max, m.Stat8RuthlessMin, m.Stat8RuthlessMax},
	}
	var out []dataset.ModStat
	for _, s := range slots {
		row, ok := s.stat.Row()
		if !ok {
			continue
		}
		if row >= nStats {
			return nil, fmt.Errorf("stat row %d out of range", row)
		}
		ms := dataset.ModStat{Stat: uint32(row), Min: s.min, Max: s.max, RuthlessMin: s.min, RuthlessMax: s.max}
		if m.HasRuthless {
			ms.RuthlessMin, ms.RuthlessMax = s.rMin, s.rMax
		}
		out = append(out, ms)
	}
	return out, nil
}

// itemNames are the display names of the items group, for one language.
type itemNames struct {
	classes, bases, mods []dataset.Name
}

// items reads the items group: the classes in itemClasses with their bases and their values,
// the mods of modDomains, and the bench crafts and essences that add one of them to items of
// those classes. Classes, bases, mods and bench crafts are in table order, essence mods in the
// order essences first add them. A base's tags are its own, then those of its item template
// chain (see ot.ItemTags). Bases of removedClass, and mods of a domain that only bases of other
// classes have, are left out with a warning that counts them. nStats is the number of rows in
// Stats; grants holds the skill of each GrantedEffectsPerLevel row.
func items(r ot.Reader, s *dat.Schema, nStats int, grants []dataset.GrantedSkill, warn func(string)) (*dataset.Items, []dataset.Mod, itemNames, error) {
	var names itemNames
	tags, err := readIDs(r, s, "Tags")
	if err != nil {
		return nil, nil, names, err
	}
	it := &dataset.Items{Tags: tags}
	bases, err := readRows[baseRow](r, s, "BaseItemTypes")
	if err != nil {
		return nil, nil, names, err
	}
	classes, err := readRows[itemClassRow](r, s, "ItemClasses")
	if err != nil {
		return nil, nil, names, err
	}
	orphans, removed, err := orphanDomains(bases, classes)
	if err != nil {
		return nil, nil, names, err
	}
	if removed > 0 {
		warn(fmt.Sprintf("table BaseItemTypes: %d bases of class %s left out", removed, removedClass))
	}
	mods, modIDs, err := itemMods(r, s, it.Tags, nStats, grants, orphans, warn, &names)
	if err != nil {
		return nil, nil, names, err
	}
	for _, c := range classes {
		if itemClasses[c.ID] {
			it.Classes = append(it.Classes, dataset.ItemClass{ID: c.ID})
			names.classes = append(names.classes, dataset.Name{ID: c.ID, Name: c.Name})
		}
	}
	templateTags := templateTagger(r, it.Tags)
	kept := map[int]int{} // BaseItemTypes row to index in it.Bases
	for row, b := range bases {
		base, keep, err := itemBase(b, classes, it.Tags, modIDs)
		if err != nil {
			return nil, nil, names, fmt.Errorf("table BaseItemTypes row %d: %w", row, err)
		}
		if !keep {
			continue
		}
		inherited, err := templateTags(b.Template)
		if err != nil {
			return nil, nil, names, fmt.Errorf("table BaseItemTypes row %d: %w", row, err)
		}
		for _, tag := range inherited {
			if !slices.Contains(base.Tags, tag) {
				base.Tags = append(base.Tags, tag)
			}
		}
		kept[row] = len(it.Bases)
		it.Bases = append(it.Bases, base)
		if b.Name != "" {
			names.bases = append(names.bases, dataset.Name{ID: b.ID, Name: b.Name})
		}
	}
	if err := baseValues(r, s, it.Bases, kept, len(bases)); err != nil {
		return nil, nil, names, err
	}
	if it.BenchCrafts, err = benchCrafts(r, s, classes, modIDs); err != nil {
		return nil, nil, names, err
	}
	if it.EssenceMods, err = essenceMods(r, s, classes, modIDs); err != nil {
		return nil, nil, names, err
	}
	it.Crafting = true
	return it, mods, names, nil
}

// orphanDomains returns the mod domains of modDomains that some base has but no base of
// itemClasses has, so no kept base rolls their mods, and the number of bases of removedClass.
func orphanDomains(bases []baseRow, classes []itemClassRow) (map[uint32]bool, int, error) {
	has, kept := map[uint32]bool{}, map[uint32]bool{}
	removed := 0
	for row, b := range bases {
		c, err := lookup(b.Class, classes, "ItemClasses")
		if err != nil {
			return nil, 0, fmt.Errorf("table BaseItemTypes row %d: %w", row, err)
		}
		has[b.ModDomain] = true
		if itemClasses[c.ID] {
			kept[b.ModDomain] = true
		}
		if c.ID == removedClass {
			removed++
		}
	}
	orphans := map[uint32]bool{}
	for d := range has {
		if modDomains[d] && !kept[d] {
			orphans[d] = true
		}
	}
	return orphans, removed, nil
}

// itemBase returns the base of row b, and false when its class is not kept. modIDs holds the
// id of each kept Mods row.
func itemBase(b baseRow, classes []itemClassRow, tags []string, modIDs map[int]string) (dataset.ItemBase, bool, error) {
	c, err := lookup(b.Class, classes, "ItemClasses")
	if err != nil || !itemClasses[c.ID] {
		return dataset.ItemBase{}, false, err
	}
	base := dataset.ItemBase{ID: b.ID, Class: c.ID}
	if base.ModDomain, err = narrow[int32](b.ModDomain, "ModDomain"); err != nil {
		return base, false, err
	}
	for _, ref := range b.Implicits {
		row, ok := ref.Row()
		if !ok {
			return base, false, fmt.Errorf("empty implicit mod")
		}
		id, kept := modIDs[row]
		if !kept {
			return base, false, fmt.Errorf("implicit mod row %d is not an item mod", row)
		}
		base.Implicits = append(base.Implicits, id)
	}
	if base.Tags, err = lookupAll(b.Tags, tags, "Tags"); err != nil {
		return base, false, err
	}
	return base, true, nil
}

// benchCrafts reads the crafting bench options that are not disabled, a craft for the mod and
// the enchantment each adds. A craft applies to the classes the option names and those of its
// categories, in ItemClasses order and limited to itemClasses; an option for none of them is
// left out. modIDs holds the id of each kept Mods row.
func benchCrafts(r ot.Reader, s *dat.Schema, classes []itemClassRow, modIDs map[int]string) ([]dataset.ModClasses, error) {
	rows, err := readRows[benchCraftRow](r, s, "CraftingBenchOptions")
	if err != nil {
		return nil, err
	}
	categories, err := readRows[craftCategoryRow](r, s, "CraftingItemClassCategories")
	if err != nil {
		return nil, err
	}
	var out []dataset.ModClasses
	for row, c := range rows {
		mods := slices.DeleteFunc([]dat.Ref{c.Mod, c.Enchantment}, func(m dat.Ref) bool { _, ok := m.Row(); return !ok })
		if len(mods) == 0 || c.Disabled {
			continue
		}
		crafts, err := benchCraft(c, mods, classes, categories, modIDs)
		if err != nil {
			return nil, fmt.Errorf("table CraftingBenchOptions row %d: %w", row, err)
		}
		out = append(out, crafts...)
	}
	return out, nil
}

// benchCraft returns a craft of c for each of mods, none when c applies to no class in
// itemClasses.
func benchCraft(c benchCraftRow, mods []dat.Ref, classes []itemClassRow, categories []craftCategoryRow, modIDs map[int]string) ([]dataset.ModClasses, error) {
	refs := slices.Clone(c.Classes)
	for _, ref := range c.Categories {
		cat, err := lookup(ref, categories, "CraftingItemClassCategories")
		if err != nil {
			return nil, err
		}
		refs = append(refs, cat.Classes...)
	}
	ids, err := keptClasses(refs, classes)
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	var out []dataset.ModClasses
	for _, m := range mods {
		row, _ := m.Row()
		id, ok := modIDs[row]
		if !ok {
			return nil, fmt.Errorf("mod row %d is not an item mod", row)
		}
		out = append(out, dataset.ModClasses{Mod: id, Classes: ids})
	}
	return out, nil
}

// keptClasses returns the ItemClasses.Ids of refs that are in itemClasses, once each and in
// table order.
func keptClasses(refs []dat.Ref, classes []itemClassRow) ([]string, error) {
	var rows []int
	for _, ref := range refs {
		row, err := rowOf(ref, len(classes), "ItemClasses")
		if err != nil {
			return nil, err
		}
		if itemClasses[classes[row].ID] && !slices.Contains(rows, row) {
			rows = append(rows, row)
		}
	}
	slices.Sort(rows)
	var ids []string
	for _, row := range rows {
		ids = append(ids, classes[row].ID)
	}
	return ids, nil
}

// essenceMods reads the mods essences add, each once with the classes of every essence that
// adds it, in the order essences first add them. Classes are in ItemClasses order; a class not
// in ItemClasses is left out. classMods names only classes in itemClasses. modIDs holds the id
// of each kept Mods row.
func essenceMods(r ot.Reader, s *dat.Schema, classes []itemClassRow, modIDs map[int]string) ([]dataset.ModClasses, error) {
	rows, err := readRows[essenceRow](r, s, "Essences")
	if err != nil {
		return nil, err
	}
	order := map[string]int{}
	for i, c := range classes {
		order[c.ID] = i
	}
	var out []dataset.ModClasses
	index := map[string]int{} // mod id to index in out
	for row, e := range rows {
		for _, cm := range e.classMods() {
			m, ok := cm.mod.Row()
			if !ok {
				continue
			}
			id, ok := modIDs[m]
			if !ok {
				return nil, fmt.Errorf("table Essences row %d: mod row %d is not an item mod", row, m)
			}
			i, seen := index[id]
			if !seen {
				i, index[id] = len(out), len(out)
				out = append(out, dataset.ModClasses{Mod: id})
			}
			for _, k := range cm.classes {
				if _, ok := order[k]; ok && !slices.Contains(out[i].Classes, k) {
					out[i].Classes = append(out[i].Classes, k)
				}
			}
		}
	}
	for i := range out {
		slices.SortFunc(out[i].Classes, func(a, b string) int { return order[a] - order[b] })
	}
	return slices.DeleteFunc(out, func(m dataset.ModClasses) bool { return len(m.Classes) == 0 }), nil
}

// templateTagger returns a function giving the tags a base inherits from the item template at a
// path, which must all be in tags. It reads each template once.
func templateTagger(r ot.Reader, tags []string) func(string) ([]string, error) {
	known := map[string]bool{}
	for _, t := range tags {
		known[t] = true
	}
	byPath := map[string][]string{}
	return func(path string) ([]string, error) {
		if path == "" {
			return nil, errors.New("no item template")
		}
		if t, ok := byPath[path]; ok {
			return t, nil
		}
		t, err := ot.ItemTags(r, path)
		if err != nil {
			return nil, fmt.Errorf("item template %s: %w", path, err)
		}
		for _, tag := range t {
			if !known[tag] {
				return nil, fmt.Errorf("item template %s: tag %q is not in Tags", path, tag)
			}
		}
		byPath[path] = t
		return t, nil
	}
}

// baseValues sets the armour, shield, weapon and flask values of bases. kept maps a
// BaseItemTypes row to its index in bases; rows of other bases are skipped. nBases is the
// number of BaseItemTypes rows.
func baseValues(r ot.Reader, s *dat.Schema, bases []dataset.ItemBase, kept map[int]int, nBases int) error {
	// base returns the kept base a values row belongs to, or nil.
	base := func(table string, row int, ref dat.Ref, set bool) (*dataset.ItemBase, error) {
		b, ok := ref.Row()
		switch {
		case !ok:
			return nil, fmt.Errorf("table %s row %d: no base", table, row)
		case b >= nBases:
			return nil, fmt.Errorf("table %s row %d: base row %d out of range", table, row, b)
		case set:
			return nil, fmt.Errorf("table %s row %d: base %d has a second row", table, row, b)
		}
		i, ok := kept[b]
		if !ok {
			return nil, nil
		}
		return &bases[i], nil
	}

	armour, err := readRows[armourRow](r, s, "ArmourTypes")
	if err != nil {
		return err
	}
	seen := map[dat.Ref]bool{}
	for row, a := range armour {
		b, err := base("ArmourTypes", row, a.Base, seen[a.Base])
		if err != nil {
			return err
		}
		seen[a.Base] = true
		if b != nil {
			b.Armour = &dataset.ArmourValues{
				Armour:        dataset.ValueRange{Min: a.ArmourMin, Max: a.ArmourMax},
				Evasion:       dataset.ValueRange{Min: a.EvasionMin, Max: a.EvasionMax},
				EnergyShield:  dataset.ValueRange{Min: a.ESMin, Max: a.ESMax},
				Ward:          dataset.ValueRange{Min: a.WardMin, Max: a.WardMax},
				MovementSpeed: a.MovementSpeed,
			}
		}
	}

	shields, err := readRows[shieldRow](r, s, "ShieldTypes")
	if err != nil {
		return err
	}
	clear(seen)
	for row, sh := range shields {
		b, err := base("ShieldTypes", row, sh.Base, seen[sh.Base])
		if err != nil {
			return err
		}
		seen[sh.Base] = true
		if b != nil {
			b.Block = sh.Block
		}
	}

	weapons, err := readRows[weaponRow](r, s, "WeaponTypes")
	if err != nil {
		return err
	}
	clear(seen)
	for row, w := range weapons {
		b, err := base("WeaponTypes", row, w.Base, seen[w.Base])
		if err != nil {
			return err
		}
		seen[w.Base] = true
		if b != nil {
			b.Weapon = &dataset.WeaponValues{
				PhysicalDamage: dataset.ValueRange{Min: w.DamageMin, Max: w.DamageMax},
				AttackTime:     w.Speed, CriticalChance: w.Critical, Range: w.Range,
			}
		}
	}
	return flaskValues(r, s, base)
}

// flaskValues sets the flask values of the bases base returns, joining Flasks with
// ComponentCharges on the base id.
func flaskValues(r ot.Reader, s *dat.Schema, base func(string, int, dat.Ref, bool) (*dataset.ItemBase, error)) error {
	flasks, err := readRows[flaskRow](r, s, "Flasks")
	if err != nil {
		return err
	}
	charges, err := readRows[chargesRow](r, s, "ComponentCharges")
	if err != nil {
		return err
	}
	byBase := map[string]chargesRow{}
	for row, c := range charges {
		if _, dup := byBase[c.Base]; dup {
			return fmt.Errorf("table ComponentCharges row %d: base %s has a second row", row, c.Base)
		}
		byBase[c.Base] = c
	}
	seen := map[dat.Ref]bool{}
	for row, f := range flasks {
		b, err := base("Flasks", row, f.Base, seen[f.Base])
		if err != nil {
			return err
		}
		seen[f.Base] = true
		if b == nil {
			continue
		}
		c, ok := byBase[b.ID]
		if !ok {
			return fmt.Errorf("table Flasks row %d: base %s has no ComponentCharges row", row, b.ID)
		}
		b.Flask = &dataset.FlaskValues{
			Kind: dataset.FlaskKind(f.Type), Life: f.Life, Mana: f.Mana, Duration: f.RecoveryTime,
			MaxCharges: c.Max, ChargesPerUse: c.PerCharge,
		}
	}
	return nil
}

// itemMods reads the mods of modDomains other than those of leaveOut, with the skills they grant
// (grants holds the skill of each GrantedEffectsPerLevel row), and adds their names to names.
// It returns the mods and the id of each kept Mods row, and warns with the count of each domain
// left out.
func itemMods(r ot.Reader, s *dat.Schema, tags []string, nStats int, grants []dataset.GrantedSkill,
	leaveOut map[uint32]bool, warn func(string), names *itemNames,
) ([]dataset.Mod, map[int]string, error) {
	rows, err := readRows[modRow](r, s, "Mods")
	if err != nil {
		return nil, nil, err
	}
	types, err := readRows[modTypeRow](r, s, "ModType")
	if err != nil {
		return nil, nil, err
	}
	families, err := readRows[idRow](r, s, "ModFamily")
	if err != nil {
		return nil, nil, err
	}
	var mods []dataset.Mod
	ids := map[int]string{}
	left := map[uint32]int{}
	for row, m := range rows {
		if !modDomains[m.Domain] {
			continue
		}
		if leaveOut[m.Domain] {
			left[m.Domain]++
			continue
		}
		mod, err := itemMod(m, types, families, tags, nStats)
		if err == nil {
			mod.GrantedSkills, err = lookupAll(m.GrantedSkills, grants, "GrantedEffectsPerLevel")
		}
		if err == nil {
			mod.RuthlessGrantedSkills, err = lookupAll(m.RuthlessGrants, grants, "GrantedEffectsPerLevel")
		}
		if err != nil {
			return nil, nil, fmt.Errorf("table Mods row %d: %w", row, err)
		}
		ids[row] = m.ID
		mods = append(mods, mod)
		if m.Name != "" {
			names.mods = append(names.mods, dataset.Name{ID: m.ID, Name: m.Name})
		}
	}
	for _, d := range slices.Sorted(maps.Keys(left)) {
		warn(fmt.Sprintf("table Mods: %d mods of domain %d left out; no kept base has that domain", left[d], d))
	}
	return mods, ids, nil
}

func itemMod(m modRow, types []modTypeRow, families []idRow, tags []string, nStats int) (dataset.Mod, error) {
	mod := dataset.Mod{ID: m.ID, Domain: int32(m.Domain), Generation: int32(m.Generation), Level: m.Level}
	gameMode, err := narrow[int32](m.GameMode, "GameMode")
	if err != nil {
		return mod, err
	}
	mod.GameMode = gameMode
	t, err := lookup(m.Type, types, "ModType")
	if err != nil {
		return mod, err
	}
	mod.Type = t.Name
	for _, ref := range m.Families {
		f, err := lookup(ref, families, "ModFamily")
		if err != nil {
			return mod, err
		}
		mod.Families = append(mod.Families, f.ID)
	}
	if mod.Stats, err = m.stats(nStats); err != nil {
		return mod, err
	}
	if mod.Tags, err = lookupAll(m.ImplicitTags, tags, "Tags"); err != nil {
		return mod, err
	}
	if mod.AddedTags, err = lookupAll(m.Tags, tags, "Tags"); err != nil {
		return mod, err
	}
	if mod.SpawnWeights, err = tagWeights(m.SpawnTags, m.SpawnWeights, tags); err != nil {
		return mod, fmt.Errorf("spawn weights: %w", err)
	}
	if mod.GenerationWeights, err = tagWeights(m.GenerationTags, m.GenerationWeights, tags); err != nil {
		return mod, fmt.Errorf("generation weights: %w", err)
	}
	return mod, nil
}

// tagWeights pairs tag references with their weights.
func tagWeights(refs []dat.Ref, weights []int32, tags []string) ([]dataset.TagWeight, error) {
	if len(refs) != len(weights) {
		return nil, fmt.Errorf("%d tags, %d weights", len(refs), len(weights))
	}
	ids, err := lookupAll(refs, tags, "Tags")
	if err != nil {
		return nil, err
	}
	var out []dataset.TagWeight
	for i, id := range ids {
		out = append(out, dataset.TagWeight{Tag: id, Weight: weights[i]})
	}
	return out, nil
}
