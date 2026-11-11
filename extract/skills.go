package extract

import (
	"cmp"
	"fmt"
	"math"
	"slices"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
)

// SkillDescriptions returns the stat description files of skills and gems that datasets carry,
// with the files they include. Which of them describes a given skill is not in the skill tables.
func SkillDescriptions() []string {
	return []string{
		sdDir + "active_skill_gem_stat_descriptions.txt",
		sdDir + "aura_skill_stat_descriptions.txt",
		sdDir + "banner_aura_skill_stat_descriptions.txt",
		sdDir + "beam_skill_stat_descriptions.txt",
		sdDir + "brand_skill_stat_descriptions.txt",
		sdDir + "buff_skill_stat_descriptions.txt",
		sdDir + "curse_skill_stat_descriptions.txt",
		sdDir + "debuff_skill_stat_descriptions.txt",
		sdDir + "gem_stat_descriptions.txt",
		sdDir + "minion_attack_skill_stat_descriptions.txt",
		sdDir + "minion_skill_stat_descriptions.txt",
		sdDir + "minion_spell_damage_skill_stat_descriptions.txt",
		sdDir + "minion_spell_skill_stat_descriptions.txt",
		sdDir + "offering_skill_stat_descriptions.txt",
		sdDir + "secondary_debuff_skill_stat_descriptions.txt",
		sdDir + "single_minion_spell_skill_stat_descriptions.txt",
		sdDir + "skill_stat_descriptions.txt",
		sdDir + "variable_duration_skill_stat_descriptions.txt",
	}
}

const sdDir = "Metadata/StatDescriptions/"

type skillGemRow struct {
	BaseItemType        dat.Ref   `dat:"BaseItemTypesKey" ref:"BaseItemTypes"`
	StrengthPercent     int32     `dat:"StrengthRequirementPercent"`
	DexterityPercent    int32     `dat:"DexterityRequirementPercent"`
	IntelligencePercent int32     `dat:"IntelligenceRequirementPercent"`
	VaalVariant         dat.Ref   `dat:"VaalVariant_BaseItemTypesKey" ref:"BaseItemTypes"`
	IsVaal              bool      `dat:"IsVaalVariant"`
	IsSupport           bool      `dat:"IsSupport"`
	AwakenedVariant     dat.Ref   `dat:"AwakenedVariant" ref:"SkillGems"`
	RegularVariant      dat.Ref   `dat:"RegularVariant" ref:"SkillGems"`
	Colour              int32     `dat:"GemColour"`
	Effects             []dat.Ref `dat:"GemVariants" ref:"GemEffects"`
}

type baseItemRow struct {
	ID   string `dat:"Id"`
	Name string `dat:"Name"`
}

type gemEffectRow struct {
	ID                     string    `dat:"Id"`
	GrantedEffect          dat.Ref   `dat:"GrantedEffect" ref:"GrantedEffects"`
	GrantedEffectHardmode  dat.Ref   `dat:"GrantedEffectHardmode" ref:"GrantedEffects"`
	GrantedEffect2         dat.Ref   `dat:"GrantedEffect2" ref:"GrantedEffects"`
	GrantedEffect2Hardmode dat.Ref   `dat:"GrantedEffect2Hardmode" ref:"GrantedEffects"`
	Tags                   []dat.Ref `dat:"GemTags" ref:"GemTags"`
	Style                  uint32    `dat:"ItemColor"`
}

type grantedEffectRow struct {
	ID                 string    `dat:"Id"`
	IsSupport          bool      `dat:"IsSupport"`
	AllowedTypes       []dat.Ref `dat:"AllowedActiveSkillTypes" ref:"ActiveSkillType"`
	AddedTypes         []dat.Ref `dat:"AddedActiveSkillTypes" ref:"ActiveSkillType"`
	ExcludedTypes      []dat.Ref `dat:"ExcludedActiveSkillTypes" ref:"ActiveSkillType"`
	SupportsGemsOnly   bool      `dat:"SupportsGemsOnly"`
	CannotBeSupported  bool      `dat:"CannotBeSupported"`
	CastTime           int32     `dat:"CastTime"`
	ActiveSkill        dat.Ref   `dat:"ActiveSkill" ref:"ActiveSkills"`
	AddedMinionTypes   []dat.Ref `dat:"AddedMinionActiveSkillTypes" ref:"ActiveSkillType"`
	WeaponRestrictions []dat.Ref `dat:"SupportWeaponRestrictions" ref:"ItemClasses"`
	StatSet            dat.Ref   `dat:"StatSet1" ref:"GrantedEffectStatSets"`
	MoreStatSets       []dat.Ref `dat:"StatSet2" ref:"GrantedEffectStatSets"`
}

type effectLevelRow struct {
	GrantedEffect          dat.Ref   `dat:"GrantedEffect" ref:"GrantedEffects"`
	Level                  int32     `dat:"Level"`
	PlayerLevel            float32   `dat:"PlayerLevelReq"`
	CostMultiplier         int32     `dat:"CostMultiplier"`
	StoredUses             int32     `dat:"StoredUses"`
	Cooldown               int32     `dat:"Cooldown"`
	VaalSouls              int32     `dat:"VaalSouls"`
	VaalStoredUses         int32     `dat:"VaalStoredUses"`
	CooldownGroup          int32     `dat:"CooldownGroup"`
	SoulGainPrevention     int32     `dat:"SoulGainPreventionDuration"`
	AttackSpeedMultiplier  int32     `dat:"AttackSpeedMultiplier"`
	CostAmounts            []int32   `dat:"CostAmounts"`
	CostTypes              []dat.Ref `dat:"CostTypes" ref:"CostTypes"`
	ManaReservationFlat    int32     `dat:"ManaReservationFlat"`
	ManaReservationPercent int32     `dat:"ManaReservationPercent"`
	LifeReservationFlat    int32     `dat:"LifeReservationFlat"`
	LifeReservationPercent int32     `dat:"LifeReservationPercent"`
	AttackTime             int32     `dat:"AttackTime"`
}

type qualityRow struct {
	GrantedEffect dat.Ref   `dat:"GrantedEffectsKey" ref:"GrantedEffects"`
	Stats         []dat.Ref `dat:"StatsKeys" ref:"Stats"`
	Values        []int32   `dat:"StatsValuesPermille"`
}

type statSetRow struct {
	ID                       string    `dat:"Id"`
	ImplicitStats            []dat.Ref `dat:"ImplicitStats" ref:"Stats"`
	ConstantStats            []dat.Ref `dat:"ConstantStats" ref:"Stats"`
	ConstantValues           []int32   `dat:"ConstantStatsValues"`
	BaseEffectiveness        float32   `dat:"BaseEffectiveness"`
	IncrementalEffectiveness float32   `dat:"IncrementalEffectiveness"`
}

type statSetLevelRow struct {
	StatSet             dat.Ref   `dat:"StatSet" ref:"GrantedEffectStatSets"`
	Level               int32     `dat:"GemLevel"`
	PlayerLevel         float32   `dat:"PlayerLevelReq"`
	SpellCritChance     int32     `dat:"SpellCritChance"`
	AttackCritChance    int32     `dat:"AttackCritChance"`
	BaseMultiplier      int32     `dat:"BaseMultiplier"`
	DamageEffectiveness int32     `dat:"DamageEffectiveness"`
	ResolvedValues      []int32   `dat:"BaseResolvedValues"`
	StatValues          []int32   `dat:"AdditionalStatsValues"`
	Flags               []dat.Ref `dat:"AdditionalFlags" ref:"Stats"`
	FloatStats          []dat.Ref `dat:"FloatStats" ref:"Stats"`
	Bases               []dat.Ref `dat:"InterpolationBases" ref:"EffectivenessCostConstants"`
	Stats               []dat.Ref `dat:"AdditionalStats" ref:"Stats"`
	// Interpolations has one entry per float stat, then one per stat.
	Interpolations []uint32  `dat:"StatInterpolations"`
	FloatValues    []float32 `dat:"FloatStatsValues"`
}

type activeSkillRow struct {
	ID                 string    `dat:"Id"`
	Name               string    `dat:"DisplayedName"`
	Types              []dat.Ref `dat:"ActiveSkillTypes" ref:"ActiveSkillType"`
	WeaponRestrictions []dat.Ref `dat:"WeaponRestriction_ItemClassesKeys" ref:"ItemClasses"`
	Totem              uint32    `dat:"SkillTotemId"`
	MinionTypes        []dat.Ref `dat:"MinionActiveSkillTypes" ref:"ActiveSkillType"`
	Mine               int32     `dat:"SkillMine"`
	TransfigureBase    dat.Ref   `dat:"TransfigureBase" ref:"ActiveSkills"`
}

// skillNames are the English gem and skill names of the skills group.
type skillNames struct {
	gems, skills []dataset.Name
}

// gemStyleColumn is the column of gem styles, which older game files number otherwise.
const gemStyleColumn = "GemEffects.ItemColor"

// skillTables holds the ids of the tables skill rows refer to.
type skillTables struct {
	nStats                                                                           int
	types, itemClasses, gemTags, costTypes, bases, effects, sets, actives, baseItems []string
	// styles maps the game files' gem styles to the schema's, nil when they agree.
	styles map[uint32]uint32
}

// skills reads the skills group: every gem, granted effect, stat set and active skill, in table
// order. nStats is the number of rows in Stats; styles, when not nil, maps the game files' gem
// styles to the schema's.
func skills(r ot.Reader, schema *dat.Schema, nStats int, styles map[uint32]uint32) (*dataset.Skills, skillNames, error) {
	var names skillNames
	t := skillTables{nStats: nStats, styles: styles}
	var err error
	for _, x := range []struct {
		table string
		dst   *[]string
	}{
		{"ActiveSkillType", &t.types}, {"ItemClasses", &t.itemClasses}, {"GemTags", &t.gemTags},
		{"CostTypes", &t.costTypes}, {"EffectivenessCostConstants", &t.bases},
	} {
		if *x.dst, err = readIDs(r, schema, x.table); err != nil {
			return nil, names, err
		}
	}
	effects, err := readRows[grantedEffectRow](r, schema, "GrantedEffects")
	if err != nil {
		return nil, names, err
	}
	sets, err := readRows[statSetRow](r, schema, "GrantedEffectStatSets")
	if err != nil {
		return nil, names, err
	}
	actives, err := readRows[activeSkillRow](r, schema, "ActiveSkills")
	if err != nil {
		return nil, names, err
	}
	t.effects = ids(effects, func(e grantedEffectRow) string { return e.ID })
	t.sets = ids(sets, func(s statSetRow) string { return s.ID })
	t.actives = ids(actives, func(a activeSkillRow) string { return a.ID })

	s := &dataset.Skills{Types: t.types}
	if s.Gems, names.gems, err = t.gems(r, schema); err != nil {
		return nil, names, err
	}
	if s.GrantedEffects, err = t.grantedEffects(r, schema, effects); err != nil {
		return nil, names, err
	}
	if s.StatSets, err = t.statSets(r, schema, sets); err != nil {
		return nil, names, err
	}
	for i, a := range actives {
		as, err := t.activeSkill(a)
		if err != nil {
			return nil, names, fmt.Errorf("table ActiveSkills row %d: %w", i, err)
		}
		s.ActiveSkills = append(s.ActiveSkills, as)
		if a.Name != "" {
			names.skills = append(names.skills, dataset.Name{ID: a.ID, Name: a.Name})
		}
	}
	return s, names, nil
}

func (t *skillTables) gems(r ot.Reader, schema *dat.Schema) ([]dataset.Gem, []dataset.Name, error) {
	rows, err := readRows[skillGemRow](r, schema, "SkillGems")
	if err != nil {
		return nil, nil, err
	}
	bases, err := readRows[baseItemRow](r, schema, "BaseItemTypes")
	if err != nil {
		return nil, nil, err
	}
	t.baseItems = ids(bases, func(b baseItemRow) string { return b.ID })
	effects, err := readRows[gemEffectRow](r, schema, "GemEffects")
	if err != nil {
		return nil, nil, err
	}
	var gems []dataset.Gem
	var names []dataset.Name
	for i, row := range rows {
		g, err := t.gem(rows, effects, row)
		if err != nil {
			return nil, nil, fmt.Errorf("table SkillGems row %d: %w", i, err)
		}
		gems = append(gems, g)
		if b, _ := row.BaseItemType.Row(); bases[b].Name != "" {
			names = append(names, dataset.Name{ID: g.ID, Name: bases[b].Name})
		}
	}
	return gems, names, nil
}

func (t *skillTables) gem(gems []skillGemRow, effects []gemEffectRow, row skillGemRow) (dataset.Gem, error) {
	g := dataset.Gem{
		StrengthPercent:     row.StrengthPercent,
		DexterityPercent:    row.DexterityPercent,
		IntelligencePercent: row.IntelligencePercent,
		Support:             row.IsSupport,
		Vaal:                row.IsVaal,
	}
	var err error
	if g.ID, err = lookup(row.BaseItemType, t.baseItems, "BaseItemTypes"); err != nil {
		return g, err
	}
	if g.Colour, err = narrow[dataset.GemColour](row.Colour, "GemColour"); err != nil {
		return g, err
	}
	if g.VaalGem, err = optionalID(row.VaalVariant, t.baseItems, "BaseItemTypes"); err != nil {
		return g, err
	}
	for _, v := range []struct {
		ref dat.Ref
		dst *string
	}{{row.RegularVariant, &g.RegularGem}, {row.AwakenedVariant, &g.AwakenedGem}} {
		// dat checks that a row reference is inside its table.
		gem, ok := v.ref.Row()
		if !ok {
			continue
		}
		if *v.dst, err = optionalID(gems[gem].BaseItemType, t.baseItems, "BaseItemTypes"); err != nil {
			return g, err
		}
	}
	for _, ref := range row.Effects {
		e, err := rowOf(ref, len(effects), "GemEffects")
		if err != nil {
			return g, err
		}
		ge, err := t.gemEffect(effects[e])
		if err != nil {
			return g, fmt.Errorf("GemEffects row %d: %w", e, err)
		}
		g.Effects = append(g.Effects, ge)
	}
	return g, nil
}

func (t *skillTables) gemEffect(row gemEffectRow) (dataset.GemEffect, error) {
	e := dataset.GemEffect{ID: row.ID}
	var err error
	for _, x := range []struct {
		ref dat.Ref
		dst *string
	}{
		{row.GrantedEffect, &e.GrantedEffect}, {row.GrantedEffect2, &e.SecondaryGrantedEffect},
		{row.GrantedEffectHardmode, &e.HardmodeGrantedEffect}, {row.GrantedEffect2Hardmode, &e.HardmodeSecondaryGrantedEffect},
	} {
		if *x.dst, err = optionalID(x.ref, t.effects, "GrantedEffects"); err != nil {
			return e, err
		}
	}
	if e.Tags, err = lookupAll(row.Tags, t.gemTags, "GemTags"); err != nil {
		return e, err
	}
	style := row.Style
	if t.styles != nil {
		var ok bool
		if style, ok = t.styles[row.Style]; !ok {
			return e, fmt.Errorf("column ItemColor: %d not in the edition's values", row.Style)
		}
	}
	e.Style, err = narrow[dataset.GemStyle](style, "ItemColor")
	return e, err
}

func (t *skillTables) grantedEffects(r ot.Reader, schema *dat.Schema, rows []grantedEffectRow) ([]dataset.GrantedEffect, error) {
	levels, err := readRows[effectLevelRow](r, schema, "GrantedEffectsPerLevel")
	if err != nil {
		return nil, err
	}
	quality, err := readRows[qualityRow](r, schema, "GrantedEffectQualityStats")
	if err != nil {
		return nil, err
	}
	out := make([]dataset.GrantedEffect, len(rows))
	for i, row := range rows {
		if out[i], err = t.grantedEffect(row); err != nil {
			return nil, fmt.Errorf("table GrantedEffects row %d: %w", i, err)
		}
	}
	for i, l := range levels {
		ge, err := rowOf(l.GrantedEffect, len(out), "GrantedEffects")
		if err != nil {
			return nil, fmt.Errorf("table GrantedEffectsPerLevel row %d: %w", i, err)
		}
		el, err := t.effectLevel(l)
		if err != nil {
			return nil, fmt.Errorf("table GrantedEffectsPerLevel row %d: %w", i, err)
		}
		out[ge].Levels = append(out[ge].Levels, el)
	}
	for i, q := range quality {
		ge, err := rowOf(q.GrantedEffect, len(out), "GrantedEffects")
		if err != nil {
			return nil, fmt.Errorf("table GrantedEffectQualityStats row %d: %w", i, err)
		}
		stats, err := t.statValues(q.Stats, q.Values)
		if err != nil {
			return nil, fmt.Errorf("table GrantedEffectQualityStats row %d: %w", i, err)
		}
		out[ge].QualityStats = append(out[ge].QualityStats, stats...)
	}
	for i := range out {
		slices.SortStableFunc(out[i].Levels, func(a, b dataset.EffectLevel) int { return cmp.Compare(a.Level, b.Level) })
	}
	return out, nil
}

func (t *skillTables) grantedEffect(row grantedEffectRow) (dataset.GrantedEffect, error) {
	g := dataset.GrantedEffect{
		ID:                row.ID,
		Support:           row.IsSupport,
		CastTime:          row.CastTime,
		SupportsGemsOnly:  row.SupportsGemsOnly,
		CannotBeSupported: row.CannotBeSupported,
	}
	var err error
	if g.ActiveSkill, err = optionalID(row.ActiveSkill, t.actives, "ActiveSkills"); err != nil {
		return g, err
	}
	if g.StatSets, err = lookupAll(append([]dat.Ref{row.StatSet}, row.MoreStatSets...), t.sets, "GrantedEffectStatSets"); err != nil {
		return g, err
	}
	for _, x := range []struct {
		refs []dat.Ref
		dst  *[]uint16
	}{
		{row.AllowedTypes, &g.AllowedTypes}, {row.ExcludedTypes, &g.ExcludedTypes},
		{row.AddedTypes, &g.AddedTypes}, {row.AddedMinionTypes, &g.AddedMinionTypes},
	} {
		if *x.dst, err = t.skillTypes(x.refs); err != nil {
			return g, err
		}
	}
	g.SupportWeaponRestrictions, err = lookupAll(row.WeaponRestrictions, t.itemClasses, "ItemClasses")
	return g, err
}

func (t *skillTables) effectLevel(row effectLevelRow) (dataset.EffectLevel, error) {
	l := dataset.EffectLevel{
		Level:                  row.Level,
		CostMultiplier:         row.CostMultiplier,
		Cooldown:               row.Cooldown,
		CooldownGroup:          row.CooldownGroup,
		StoredUses:             row.StoredUses,
		VaalSouls:              row.VaalSouls,
		VaalStoredUses:         row.VaalStoredUses,
		SoulGainPrevention:     row.SoulGainPrevention,
		AttackSpeedMultiplier:  row.AttackSpeedMultiplier,
		AttackTime:             row.AttackTime,
		ManaReservationFlat:    row.ManaReservationFlat,
		ManaReservationPercent: row.ManaReservationPercent,
		LifeReservationFlat:    row.LifeReservationFlat,
		LifeReservationPercent: row.LifeReservationPercent,
	}
	var err error
	if l.PlayerLevel, err = wholeNumber(row.PlayerLevel, "PlayerLevelReq"); err != nil {
		return l, err
	}
	if len(row.CostTypes) != len(row.CostAmounts) {
		return l, fmt.Errorf("%d cost types, %d amounts", len(row.CostTypes), len(row.CostAmounts))
	}
	types, err := lookupAll(row.CostTypes, t.costTypes, "CostTypes")
	if err != nil {
		return l, err
	}
	for i, c := range types {
		l.Costs = append(l.Costs, dataset.Cost{Type: c, Amount: row.CostAmounts[i]})
	}
	return l, nil
}

func (t *skillTables) statSets(r ot.Reader, schema *dat.Schema, rows []statSetRow) ([]dataset.StatSet, error) {
	levels, err := readRows[statSetLevelRow](r, schema, "GrantedEffectStatSetsPerLevel")
	if err != nil {
		return nil, err
	}
	out := make([]dataset.StatSet, len(rows))
	for i, row := range rows {
		s := dataset.StatSet{ID: row.ID, BaseEffectiveness: row.BaseEffectiveness, IncrementalEffectiveness: row.IncrementalEffectiveness}
		if s.ImplicitStats, err = t.handles(row.ImplicitStats); err != nil {
			return nil, fmt.Errorf("table GrantedEffectStatSets row %d: %w", i, err)
		}
		if s.ConstantStats, err = t.statValues(row.ConstantStats, row.ConstantValues); err != nil {
			return nil, fmt.Errorf("table GrantedEffectStatSets row %d: %w", i, err)
		}
		out[i] = s
	}
	for i, l := range levels {
		set, err := rowOf(l.StatSet, len(out), "GrantedEffectStatSets")
		if err != nil {
			return nil, fmt.Errorf("table GrantedEffectStatSetsPerLevel row %d: %w", i, err)
		}
		sl, err := t.statSetLevel(l)
		if err != nil {
			return nil, fmt.Errorf("table GrantedEffectStatSetsPerLevel row %d: %w", i, err)
		}
		out[set].Levels = append(out[set].Levels, sl)
	}
	for i := range out {
		slices.SortStableFunc(out[i].Levels, func(a, b dataset.StatSetLevel) int { return cmp.Compare(a.Level, b.Level) })
	}
	return out, nil
}

func (t *skillTables) statSetLevel(row statSetLevelRow) (dataset.StatSetLevel, error) {
	l := dataset.StatSetLevel{
		Level:               row.Level,
		SpellCritChance:     row.SpellCritChance,
		AttackCritChance:    row.AttackCritChance,
		DamageEffectiveness: row.DamageEffectiveness,
		BaseMultiplier:      row.BaseMultiplier,
	}
	var err error
	if l.PlayerLevel, err = wholeNumber(row.PlayerLevel, "PlayerLevelReq"); err != nil {
		return l, err
	}
	nf, ns := len(row.FloatStats), len(row.Stats)
	if len(row.FloatValues) != nf || len(row.ResolvedValues) != nf || len(row.Bases) != nf ||
		len(row.StatValues) != ns || len(row.Interpolations) != nf+ns {
		return l, fmt.Errorf("%d float stats with %d values, %d resolved values and %d bases; %d stats with %d values; %d interpolations",
			nf, len(row.FloatValues), len(row.ResolvedValues), len(row.Bases), ns, len(row.StatValues), len(row.Interpolations))
	}
	interp := make([]dataset.Interpolation, nf+ns)
	for i, x := range row.Interpolations {
		if interp[i], err = narrow[dataset.Interpolation](x, "StatInterpolations"); err != nil {
			return l, err
		}
	}
	floats, err := t.handles(row.FloatStats)
	if err != nil {
		return l, err
	}
	bases, err := lookupAll(row.Bases, t.bases, "EffectivenessCostConstants")
	if err != nil {
		return l, err
	}
	for i, h := range floats {
		l.FloatStats = append(l.FloatStats, dataset.FloatStat{
			Stat: h, Value: row.FloatValues[i], Resolved: row.ResolvedValues[i], Interpolation: interp[i], Base: bases[i],
		})
	}
	stats, err := t.handles(row.Stats)
	if err != nil {
		return l, err
	}
	for i, h := range stats {
		l.Stats = append(l.Stats, dataset.LevelStat{Stat: h, Value: row.StatValues[i], Interpolation: interp[nf+i]})
	}
	l.Flags, err = t.handles(row.Flags)
	return l, err
}

func (t *skillTables) activeSkill(row activeSkillRow) (dataset.ActiveSkill, error) {
	a := dataset.ActiveSkill{ID: row.ID, Mine: row.Mine}
	var err error
	if a.Types, err = t.skillTypes(row.Types); err != nil {
		return a, err
	}
	if a.MinionTypes, err = t.skillTypes(row.MinionTypes); err != nil {
		return a, err
	}
	if a.WeaponRestrictions, err = lookupAll(row.WeaponRestrictions, t.itemClasses, "ItemClasses"); err != nil {
		return a, err
	}
	if a.Totem, err = narrow[int32](row.Totem, "SkillTotemId"); err != nil {
		return a, err
	}
	a.TransfigureBase, err = optionalID(row.TransfigureBase, t.actives, "ActiveSkills")
	return a, err
}

// skillTypes returns ActiveSkillType references as indexes into Skills.Types.
func (t *skillTables) skillTypes(refs []dat.Ref) ([]uint16, error) {
	var out []uint16
	for _, r := range refs {
		i, err := rowOf(r, min(len(t.types), math.MaxUint16+1), "ActiveSkillType")
		if err != nil {
			return nil, err
		}
		out = append(out, uint16(i))
	}
	return out, nil
}

// handles returns Stats references as stat handles.
func (t *skillTables) handles(refs []dat.Ref) ([]uint32, error) {
	var out []uint32
	for _, r := range refs {
		i, err := rowOf(r, t.nStats, "Stats")
		if err != nil {
			return nil, err
		}
		out = append(out, uint32(i))
	}
	return out, nil
}

// statValues pairs stats with their values, in game order.
func (t *skillTables) statValues(stats []dat.Ref, values []int32) ([]dataset.StatValue, error) {
	if len(stats) != len(values) {
		return nil, fmt.Errorf("%d stats, %d values", len(stats), len(values))
	}
	hs, err := t.handles(stats)
	if err != nil {
		return nil, err
	}
	var out []dataset.StatValue
	for i, h := range hs {
		out = append(out, dataset.StatValue{Stat: h, Value: values[i]})
	}
	return out, nil
}

// optionalID returns the id of the row ref references, or "" for a null reference.
func optionalID(ref dat.Ref, ids []string, table string) (string, error) {
	if _, ok := ref.Row(); !ok {
		return "", nil
	}
	return lookup(ref, ids, table)
}

// wholeNumber returns f, a game value stored as a float that holds an integer, as an int32.
func wholeNumber(f float32, col string) (int32, error) {
	if f != float32(math.Trunc(float64(f))) || f < math.MinInt32 || f > math.MaxInt32 {
		return 0, fmt.Errorf("column %s: %v is not a whole number", col, f)
	}
	return int32(f), nil
}

// narrow converts the game integer v to the dataset's type T, failing if it does not fit.
func narrow[T ~uint8 | ~int32, S ~int32 | ~uint32](v S, col string) (T, error) {
	if int64(T(v)) != int64(v) {
		return 0, fmt.Errorf("column %s: %d out of range", col, v)
	}
	return T(v), nil
}
