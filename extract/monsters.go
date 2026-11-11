package extract

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
)

// MonsterTemplate is the object template whose Stats block every monster starts from.
const MonsterTemplate = "Metadata/Monsters/Monster"

// monsterDomain is the Mods.Domain of monster mods, as the game numbers it.
const monsterDomain = 3

type monsterResistanceRow struct {
	ID                 string `dat:"Id"`
	FireNormal         int32  `dat:"FireNormal"`
	ColdNormal         int32  `dat:"ColdNormal"`
	LightningNormal    int32  `dat:"LightningNormal"`
	ChaosNormal        int32  `dat:"ChaosNormal"`
	FireCruel          int32  `dat:"FireCruel"`
	ColdCruel          int32  `dat:"ColdCruel"`
	LightningCruel     int32  `dat:"LightningCruel"`
	ChaosCruel         int32  `dat:"ChaosCruel"`
	FireMerciless      int32  `dat:"FireMerciless"`
	ColdMerciless      int32  `dat:"ColdMerciless"`
	LightningMerciless int32  `dat:"LightningMerciless"`
	ChaosMerciless     int32  `dat:"ChaosMerciless"`
}

type monsterTypeRow struct {
	ID                           string  `dat:"Id"`
	Accuracy                     int32   `dat:"Accuracy"`
	PlayerMinion                 bool    `dat:"IsPlayerMinion"`
	Armour                       int32   `dat:"Armour"`
	Evasion                      int32   `dat:"Evasion"`
	EnergyShield                 int32   `dat:"EnergyShield"`
	DamageSpread                 int32   `dat:"DamageSpread"`
	Resistances                  dat.Ref `dat:"Resistances" ref:"MonsterResistances"`
	AltLife1                     bool    `dat:"AltLife1"`
	AltLife2                     bool    `dat:"AltLife2"`
	BaseDamageIgnoresAttackSpeed bool    `dat:"BaseDamageIgnoresAttackSpeed"`
}

type monsterVarietyRow struct {
	ID               string    `dat:"Id"`
	Type             dat.Ref   `dat:"MonsterTypesKey" ref:"MonsterTypes"`
	Mods             []dat.Ref `dat:"ModsKeys" ref:"Mods"`
	Tags             []dat.Ref `dat:"TagsKeys" ref:"Tags"`
	GrantedEffects   []dat.Ref `dat:"GrantedEffectsKeys" ref:"GrantedEffects"`
	Mods2            []dat.Ref `dat:"ModsKeys2" ref:"Mods"`
	Name             string    `dat:"Name"`
	DamageMultiplier int32     `dat:"DamageMultiplier"`
	LifeMultiplier   int32     `dat:"LifeMultiplier"`
	AttackTime       int32     `dat:"AttackSpeed"`
	MainHand         dat.Ref   `dat:"MainHand_ItemClassesKey" ref:"ItemClasses"`
	OffHand          dat.Ref   `dat:"OffHand_ItemClassesKey" ref:"ItemClasses"`
	SpecialMods      []dat.Ref `dat:"Special_ModsKeys" ref:"Mods"`
	Part1Mods        []dat.Ref `dat:"Part1_ModsKeys" ref:"Mods"`
	Part2Mods        []dat.Ref `dat:"Part2_ModsKeys" ref:"Mods"`
	EndgameMods      []dat.Ref `dat:"Endgame_ModsKeys" ref:"Mods"`
	AilmentThreshold int32     `dat:"AilmentThreshold"`
}

type monsterLevelRow struct {
	// Level is the level as text, such as "70".
	Level              string  `dat:"DisplayLevel"`
	Damage             float32 `dat:"Damage"`
	Evasion            int32   `dat:"Evasion"`
	Accuracy           int32   `dat:"Accuracy"`
	Life               int32   `dat:"MonsterLife"`
	AllyLife           int32   `dat:"MinionLife"`
	Armour             int32   `dat:"Armour"`
	Difficulty         int32   `dat:"Difficulty"`
	AllyDamage         float32 `dat:"MinionDamage"`
	AltLife1           int32   `dat:"AltLife1"`
	AltDamage1         float32 `dat:"AltDamage1"`
	AltDamage2         float32 `dat:"AltDamage2"`
	AltLife2           int32   `dat:"AltLife2"`
	EvasiveEvasion     int32   `dat:"EvasiveEvasion"`
	AilmentThreshold   int32   `dat:"AilmentThreshold"`
	PhysicalConversion int32   `dat:"MonsterPhysConversionMulti"`
}

type displayMinionRow struct {
	ID      int32   `dat:"Id"`
	Variety dat.Ref `dat:"MonsterVarietiesKey" ref:"MonsterVarieties"`
}

type totemVariationRow struct {
	Key     uint32  `dat:"SkillTotemsKey"`
	Skin    int32   `dat:"TotemSkinId"`
	Variety dat.Ref `dat:"MonsterVarietiesKey" ref:"MonsterVarieties"`
}

type spectreOverrideRow struct {
	Monster dat.Ref `dat:"Monster" ref:"MonsterVarieties"`
	Spectre dat.Ref `dat:"Spectre" ref:"MonsterVarieties"`
}

type minionTypeRow struct {
	ID          string  `dat:"Id"`
	LimitStat   dat.Ref `dat:"LimitStat" ref:"Stats"`
	ActiveCount dat.Ref `dat:"ActiveCountStat" ref:"Stats"`
}

// monsters reads the monsters group, every list in table order except levels, sorted by level,
// and totems, by key, and the English monster names. handles maps stat ids to stat handles;
// totems are the edition's links the game data does not hold.
func monsters(r ot.Reader, s *dat.Schema, handles map[string]uint32, totems []dataset.EditionTotem) (*dataset.Monsters, []dataset.Name, error) {
	m := &dataset.Monsters{EditionTotems: totems}
	var err error
	if m.BaseStats, err = templateStats(r, MonsterTemplate, handles); err != nil {
		return nil, nil, err
	}
	resistances, err := readRows[monsterResistanceRow](r, s, "MonsterResistances")
	if err != nil {
		return nil, nil, err
	}
	for _, x := range resistances {
		m.Resistances = append(m.Resistances, dataset.MonsterResistance{
			ID:        x.ID,
			Normal:    dataset.Resistances{Fire: x.FireNormal, Cold: x.ColdNormal, Lightning: x.LightningNormal, Chaos: x.ChaosNormal},
			Cruel:     dataset.Resistances{Fire: x.FireCruel, Cold: x.ColdCruel, Lightning: x.LightningCruel, Chaos: x.ChaosCruel},
			Merciless: dataset.Resistances{Fire: x.FireMerciless, Cold: x.ColdMerciless, Lightning: x.LightningMerciless, Chaos: x.ChaosMerciless},
		})
	}
	types, err := readRows[monsterTypeRow](r, s, "MonsterTypes")
	if err != nil {
		return nil, nil, err
	}
	resistanceIDs := ids(resistances, func(x monsterResistanceRow) string { return x.ID })
	for i, x := range types {
		t := dataset.MonsterType{
			ID: x.ID, Armour: x.Armour, Evasion: x.Evasion, EnergyShield: x.EnergyShield, Accuracy: x.Accuracy,
			DamageSpread: x.DamageSpread, PlayerMinion: x.PlayerMinion, BaseDamageIgnoresAttackSpeed: x.BaseDamageIgnoresAttackSpeed,
			AltLife1: x.AltLife1, AltLife2: x.AltLife2,
		}
		if t.Resistances, err = optionalID(x.Resistances, resistanceIDs, "MonsterResistances"); err != nil {
			return nil, nil, fmt.Errorf("table MonsterTypes row %d: %w", i, err)
		}
		m.Types = append(m.Types, t)
	}
	if m.Levels, err = monsterLevels(r, s); err != nil {
		return nil, nil, err
	}
	tags, err := readIDs(r, s, "Tags")
	if err != nil {
		return nil, nil, err
	}
	modIDs, err := monsterMods(r, s, tags, len(handles), m)
	if err != nil {
		return nil, nil, err
	}
	varieties, err := readRows[monsterVarietyRow](r, s, "MonsterVarieties")
	if err != nil {
		return nil, nil, err
	}
	refs := varietyRefs{
		types: ids(types, func(x monsterTypeRow) string { return x.ID }),
		tags:  tags,
		mods:  modIDs,
	}
	if refs.classes, err = readIDs(r, s, "ItemClasses"); err != nil {
		return nil, nil, err
	}
	if refs.effects, err = readIDs(r, s, "GrantedEffects"); err != nil {
		return nil, nil, err
	}
	var names []dataset.Name
	for i, x := range varieties {
		v, err := refs.variety(x)
		if err != nil {
			return nil, nil, fmt.Errorf("table MonsterVarieties row %d: %w", i, err)
		}
		m.Varieties = append(m.Varieties, v)
		if x.Name != "" {
			names = append(names, dataset.Name{ID: x.ID, Name: x.Name})
		}
	}
	varietyIDs := ids(varieties, func(x monsterVarietyRow) string { return x.ID })
	if err := monsterLinks(r, s, varietyIDs, len(handles), m); err != nil {
		return nil, nil, err
	}
	return m, names, nil
}

func monsterLevels(r ot.Reader, s *dat.Schema) ([]dataset.MonsterLevel, error) {
	rows, err := readRows[monsterLevelRow](r, s, "DefaultMonsterStats")
	if err != nil {
		return nil, err
	}
	var out []dataset.MonsterLevel
	for i, x := range rows {
		level, err := strconv.ParseInt(x.Level, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("table DefaultMonsterStats row %d: DisplayLevel %q is not a level", i, x.Level)
		}
		out = append(out, dataset.MonsterLevel{
			Level: int32(level), Difficulty: x.Difficulty,
			Life: x.Life, AllyLife: x.AllyLife, AltLife1: x.AltLife1, AltLife2: x.AltLife2,
			Damage: x.Damage, AllyDamage: x.AllyDamage, AltDamage1: x.AltDamage1, AltDamage2: x.AltDamage2,
			Armour: x.Armour, Evasion: x.Evasion, EvasiveEvasion: x.EvasiveEvasion,
			Accuracy: x.Accuracy, AilmentThreshold: x.AilmentThreshold, PhysicalConversion: x.PhysicalConversion,
		})
	}
	slices.SortStableFunc(out, func(a, b dataset.MonsterLevel) int { return cmp.Compare(a.Level, b.Level) })
	return out, nil
}

// monsterMods adds the mods of the monster domain to m and returns the id of each Mods row, ""
// for rows of other domains.
func monsterMods(r ot.Reader, s *dat.Schema, tags []string, nStats int, m *dataset.Monsters) ([]string, error) {
	rows, err := readRows[modRow](r, s, "Mods")
	if err != nil {
		return nil, err
	}
	types, err := readRows[modTypeRow](r, s, "ModType")
	if err != nil {
		return nil, err
	}
	families, err := readRows[idRow](r, s, "ModFamily")
	if err != nil {
		return nil, err
	}
	modIDs := make([]string, len(rows))
	for i, x := range rows {
		if x.Domain != monsterDomain {
			continue
		}
		mod, err := itemMod(x, types, families, tags, nStats)
		if err != nil {
			return nil, fmt.Errorf("table Mods row %d: %w", i, err)
		}
		m.Mods = append(m.Mods, mod)
		modIDs[i] = x.ID
	}
	return modIDs, nil
}

// varietyRefs holds the ids of the tables MonsterVarieties refers to.
type varietyRefs struct {
	types, tags, classes, effects []string
	// mods has the id of each Mods row of the monster domain, "" for other rows.
	mods []string
}

func (refs varietyRefs) variety(x monsterVarietyRow) (dataset.MonsterVariety, error) {
	v := dataset.MonsterVariety{
		ID: x.ID, LifeMultiplier: x.LifeMultiplier, DamageMultiplier: x.DamageMultiplier,
		AilmentThreshold: x.AilmentThreshold, AttackTime: x.AttackTime,
	}
	var err error
	if v.Type, err = lookup(x.Type, refs.types, "MonsterTypes"); err != nil {
		return v, err
	}
	if v.MainHand, err = optionalID(x.MainHand, refs.classes, "ItemClasses"); err != nil {
		return v, err
	}
	if v.OffHand, err = optionalID(x.OffHand, refs.classes, "ItemClasses"); err != nil {
		return v, err
	}
	if v.Tags, err = lookupAll(x.Tags, refs.tags, "Tags"); err != nil {
		return v, err
	}
	if v.GrantedEffects, err = lookupAll(x.GrantedEffects, refs.effects, "GrantedEffects"); err != nil {
		return v, err
	}
	for _, l := range []struct {
		col  string
		refs []dat.Ref
		dst  *[]string
	}{
		{"ModsKeys", x.Mods, &v.Mods}, {"ModsKeys2", x.Mods2, &v.Mods2}, {"Special_ModsKeys", x.SpecialMods, &v.SpecialMods},
		{"Part1_ModsKeys", x.Part1Mods, &v.Part1Mods}, {"Part2_ModsKeys", x.Part2Mods, &v.Part2Mods},
		{"Endgame_ModsKeys", x.EndgameMods, &v.EndgameMods},
	} {
		if *l.dst, err = lookupAll(l.refs, refs.mods, "Mods"); err != nil {
			return v, fmt.Errorf("%s: %w", l.col, err)
		}
		if i := slices.Index(*l.dst, ""); i >= 0 {
			return v, fmt.Errorf("%s item %d: not a monster mod", l.col, i)
		}
	}
	return v, nil
}

// monsterLinks adds the rows that link skills and raised monsters to varieties: display
// minions, totems, spectre overrides and minion types.
func monsterLinks(r ot.Reader, s *dat.Schema, varieties []string, nStats int, m *dataset.Monsters) error {
	displays, err := readRows[displayMinionRow](r, s, "DisplayMinionMonsterType")
	if err != nil {
		return err
	}
	for i, x := range displays {
		v, err := lookup(x.Variety, varieties, "MonsterVarieties")
		if err != nil {
			return fmt.Errorf("table DisplayMinionMonsterType row %d: %w", i, err)
		}
		m.DisplayMinions = append(m.DisplayMinions, dataset.DisplayMinion{ID: x.ID, Variety: v})
	}
	if m.Totems, err = totems(r, s, varieties); err != nil {
		return err
	}
	overrides, err := readRows[spectreOverrideRow](r, s, "SpectreOverrides")
	if err != nil {
		return err
	}
	for i, x := range overrides {
		var o dataset.SpectreOverride
		if o.Monster, err = lookup(x.Monster, varieties, "MonsterVarieties"); err == nil {
			o.Spectre, err = lookup(x.Spectre, varieties, "MonsterVarieties")
		}
		if err != nil {
			return fmt.Errorf("table SpectreOverrides row %d: %w", i, err)
		}
		m.SpectreOverrides = append(m.SpectreOverrides, o)
	}
	minionTypes, err := readRows[minionTypeRow](r, s, "MinionType")
	if err != nil {
		return err
	}
	for i, x := range minionTypes {
		t := dataset.MinionType{ID: x.ID}
		if t.LimitStat, err = optionalStat(x.LimitStat, nStats); err == nil {
			t.ActiveCountStat, err = optionalStat(x.ActiveCount, nStats)
		}
		if err != nil {
			return fmt.Errorf("table MinionType row %d: %w", i, err)
		}
		m.MinionTypes = append(m.MinionTypes, t)
	}
	return nil
}

// totems groups SkillTotemVariations by key, sorted by key, with skins sorted by skin.
func totems(r ot.Reader, s *dat.Schema, varieties []string) ([]dataset.Totem, error) {
	rows, err := readRows[totemVariationRow](r, s, "SkillTotemVariations")
	if err != nil {
		return nil, err
	}
	byKey := map[int32][]dataset.TotemSkin{}
	for i, x := range rows {
		key, err := narrow[int32](x.Key, "SkillTotemsKey")
		if err != nil {
			return nil, fmt.Errorf("table SkillTotemVariations row %d: %w", i, err)
		}
		v, err := lookup(x.Variety, varieties, "MonsterVarieties")
		if err != nil {
			return nil, fmt.Errorf("table SkillTotemVariations row %d: %w", i, err)
		}
		byKey[key] = append(byKey[key], dataset.TotemSkin{Skin: x.Skin, Variety: v})
	}
	var out []dataset.Totem
	for _, key := range slices.Sorted(maps.Keys(byKey)) {
		skins := byKey[key]
		slices.SortStableFunc(skins, func(a, b dataset.TotemSkin) int { return cmp.Compare(a.Skin, b.Skin) })
		out = append(out, dataset.Totem{Key: key, Skins: skins})
	}
	return out, nil
}
