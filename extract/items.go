package extract

import (
	"fmt"

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
// and the mods of modDomains. All are in table order. nStats is the number of rows in Stats.
func items(r ot.Reader, s *dat.Schema, nStats int) (*dataset.Items, []dataset.Mod, itemNames, error) {
	var names itemNames
	tags, err := readIDs(r, s, "Tags")
	if err != nil {
		return nil, nil, names, err
	}
	it := &dataset.Items{Tags: tags}
	mods, modIDs, err := itemMods(r, s, it.Tags, nStats, &names)
	if err != nil {
		return nil, nil, names, err
	}
	bases, err := readRows[baseRow](r, s, "BaseItemTypes")
	if err != nil {
		return nil, nil, names, err
	}
	classes, err := readRows[itemClassRow](r, s, "ItemClasses")
	if err != nil {
		return nil, nil, names, err
	}
	for _, c := range classes {
		if itemClasses[c.ID] {
			it.Classes = append(it.Classes, dataset.ItemClass{ID: c.ID})
			names.classes = append(names.classes, dataset.Name{ID: c.ID, Name: c.Name})
		}
	}
	kept := map[int]int{} // BaseItemTypes row to index in it.Bases
	for row, b := range bases {
		base, keep, err := itemBase(b, classes, it.Tags, modIDs)
		if err != nil {
			return nil, nil, names, fmt.Errorf("table BaseItemTypes row %d: %w", row, err)
		}
		if !keep {
			continue
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
	return it, mods, names, nil
}

// itemBase returns the base of row b, and false when its class is not kept. modIDs holds the
// id of each kept Mods row.
func itemBase(b baseRow, classes []itemClassRow, tags []string, modIDs map[int]string) (dataset.ItemBase, bool, error) {
	c, err := lookup(b.Class, classes, "ItemClasses")
	if err != nil || !itemClasses[c.ID] {
		return dataset.ItemBase{}, false, err
	}
	base := dataset.ItemBase{ID: b.ID, Class: c.ID}
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

// itemMods reads the mods of modDomains and adds their names to names. It returns the mods and
// the id of each kept Mods row.
func itemMods(r ot.Reader, s *dat.Schema, tags []string, nStats int, names *itemNames) ([]dataset.Mod, map[int]string, error) {
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
	for row, m := range rows {
		if !modDomains[m.Domain] {
			continue
		}
		mod, err := itemMod(m, types, families, tags, nStats)
		if err != nil {
			return nil, nil, fmt.Errorf("table Mods row %d: %w", row, err)
		}
		ids[row] = m.ID
		mods = append(mods, mod)
		if m.Name != "" {
			names.mods = append(names.mods, dataset.Name{ID: m.ID, Name: m.Name})
		}
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
