package extract

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
)

type overrideRow struct {
	ID          string    `dat:"Id"`
	Name        string    `dat:"Name"`
	Icon        string    `dat:"NodeIcon"`
	Stats       []dat.Ref `dat:"Stats" ref:"Stats"`
	Values      []int32   `dat:"StatValues"`
	Type        dat.Ref   `dat:"Type" ref:"PassiveSkillOverrideTypes"`
	MinAdjacent int32     `dat:"RequiresAdjacent"`
	MaxAdjacent int32     `dat:"MaxAdjacent"`
	Passive     dat.Ref   `dat:"AllocatedPassiveSkill" ref:"PassiveSkills"`
	Blocking    []dat.Ref `dat:"TattooBlockingPassive" ref:"PassiveSkills"`
}

type tattooRow struct {
	Override dat.Ref `dat:"Override" ref:"PassiveSkillOverrides"`
	Set      dat.Ref `dat:"Set" ref:"PassiveSkillTattooTargetSets"`
	// OverrideType is set on a tattoo that grants a random override of that type; its own
	// Override is a placeholder shown on the item.
	OverrideType dat.Ref `dat:"OverrideType" ref:"PassiveSkillOverrideTypes"`
}

type overridePassiveRow struct {
	GraphID uint16    `dat:"PassiveSkillGraphId"`
	Name    string    `dat:"Name"`
	Icon    string    `dat:"Icon_DDSFile"`
	Stats   []dat.Ref `dat:"Stats" ref:"Stats"`
	Value1  int32     `dat:"Stat1Value"`
	Value2  int32     `dat:"Stat2Value"`
	Value3  int32     `dat:"Stat3Value"`
	Value4  int32     `dat:"Stat4Value"`
	Value5  int32     `dat:"Stat5Value"`
	Buffs   []dat.Ref `dat:"PassiveSkillBuffs" ref:"BuffTemplates"`

	HasHardmode bool      `dat:"HasHardmodeAlternate"`
	StatsHard   []dat.Ref `dat:"StatsHardmode" ref:"Stats"`
	Value1Hard  int32     `dat:"Stat1ValueHardmode"`
	Value2Hard  int32     `dat:"Stat2ValueHardmode"`
	Value3Hard  int32     `dat:"Stat3ValueHardmode"`
	Value4Hard  int32     `dat:"Stat4ValueHardmode"`
	Value5Hard  int32     `dat:"Stat5ValueHardmode"`
	BuffsHard   []dat.Ref `dat:"BuffsHardmode" ref:"BuffTemplates"`
}

// stats returns the passive's stat references, values and buffs, the ruthless ones for
// hardmode when it has them.
func (p overridePassiveRow) stats(hardmode bool) ([]dat.Ref, []int32, []dat.Ref) {
	if hardmode && p.HasHardmode {
		return p.StatsHard, []int32{p.Value1Hard, p.Value2Hard, p.Value3Hard, p.Value4Hard, p.Value5Hard}, p.BuffsHard
	}
	return p.Stats, []int32{p.Value1, p.Value2, p.Value3, p.Value4, p.Value5}, p.Buffs
}

// overrideTargets maps PassiveSkillTattooTargetSets ids to override targets.
var overrideTargets = map[string]dataset.OverrideTarget{
	"SmallStrength":     dataset.SmallStrength,
	"SmallDexterity":    dataset.SmallDexterity,
	"SmallIntelligence": dataset.SmallIntelligence,
	"SmallAnyAttribute": dataset.SmallAttribute,
	"BigStrength":       dataset.StrengthNotable,
	"BigDexterity":      dataset.DexterityNotable,
	"BigIntelligence":   dataset.IntelligenceNotable,
	"Keystone":          dataset.KeystoneTarget,
	"Mastery":           dataset.MasteryTarget,
	"Ascendancy":        dataset.AscendancyNotable,
}

// overrideTables are the tables node overrides are read from.
type overrideTables struct {
	overrides  []overrideRow
	types      []string
	tattoos    []tattooRow
	targetSets []string
	passives   []overridePassiveRow
}

func readOverrideTables(r ot.Reader, s *dat.Schema) (overrideTables, error) {
	var t overrideTables
	var err error
	if t.overrides, err = readRows[overrideRow](r, s, "PassiveSkillOverrides"); err != nil {
		return t, err
	}
	if t.types, err = readIDs(r, s, "PassiveSkillOverrideTypes"); err != nil {
		return t, err
	}
	if t.tattoos, err = readRows[tattooRow](r, s, "PassiveSkillTattoos"); err != nil {
		return t, err
	}
	if t.targetSets, err = readIDs(r, s, "PassiveSkillTattooTargetSets"); err != nil {
		return t, err
	}
	t.passives, err = readRows[overridePassiveRow](r, s, "PassiveSkills")
	return t, err
}

// build returns every tattoo and runegraft of PassiveSkillOverrides, sorted by id, with
// their English names, for a tree of the standard or, with hardmode, the ruthless stat
// columns. ExcludedNodes may name nodes a tree lacks; overridesIn drops those.
//
// A tattoo that grants a random override of a type (a keystone or ascendancy tattoo) gives
// its target to every override of that type, and its own placeholder override is left out.
// An override without a name of its own takes the name, icon and stats of its
// AllocatedPassiveSkill. One that cannot be placed or shown (no target, no icon, a stat that
// does not resolve) or that the game marks unused is left out with a warning; a missing name
// or passive buffs, which the dataset cannot hold, are a warning only.
func (t overrideTables) build(hardmode bool, nStats int, warn func(string)) ([]dataset.NodeOverride, []dataset.Name, error) {
	byOverride, byType, placeholders, err := t.targets()
	if err != nil {
		return nil, nil, err
	}
	var out []dataset.NodeOverride
	var names []dataset.Name
	for i, row := range t.overrides {
		if placeholders[i] {
			continue
		}
		o, name, problem, err := t.override(row, hardmode, nStats, warn)
		if err != nil {
			return nil, nil, fmt.Errorf("PassiveSkillOverrides %s: %w", row.ID, err)
		}
		target, ok := byOverride[i]
		if !ok {
			typ, _ := row.Type.Row()
			target, ok = byType[typ]
		}
		switch {
		case !ok:
			problem = "no PassiveSkillTattoos row gives it a target"
		case strings.HasPrefix(name, "[DNT"):
			problem = fmt.Sprintf("named %q, unused", name)
		}
		if problem != "" {
			warn(fmt.Sprintf("override %s: %s; left out", row.ID, problem))
			continue
		}
		o.Target = target
		out = append(out, o)
		if name == "" {
			warn(fmt.Sprintf("override %s: no English name", row.ID))
		} else {
			names = append(names, dataset.Name{ID: o.ID, Name: name})
		}
	}
	slices.SortFunc(out, func(a, b dataset.NodeOverride) int { return cmp.Compare(a.ID, b.ID) })
	slices.SortFunc(names, func(a, b dataset.Name) int { return cmp.Compare(a.ID, b.ID) })
	for i := 1; i < len(out); i++ {
		if out[i-1].ID == out[i].ID {
			return nil, nil, fmt.Errorf("PassiveSkillOverrides: id %s repeated", out[i].ID)
		}
	}
	return out, names, nil
}

// targets returns the target of each override row that has a PassiveSkillTattoos row, the
// target of each override type a random tattoo grants, and the placeholder rows of those
// tattoos.
func (t overrideTables) targets() (byOverride, byType map[int]dataset.OverrideTarget, placeholders map[int]bool, err error) {
	byOverride, byType, placeholders = map[int]dataset.OverrideTarget{}, map[int]dataset.OverrideTarget{}, map[int]bool{}
	for i, tr := range t.tattoos {
		set, err := lookup(tr.Set, t.targetSets, "PassiveSkillTattooTargetSets")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("PassiveSkillTattoos row %d: %w", i, err)
		}
		target, ok := overrideTargets[set]
		if !ok {
			return nil, nil, nil, fmt.Errorf("PassiveSkillTattoos row %d: target set %s has no override target", i, set)
		}
		o, err := rowOf(tr.Override, len(t.overrides), "PassiveSkillOverrides")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("PassiveSkillTattoos row %d: %w", i, err)
		}
		typ, random := tr.OverrideType.Row()
		if !random {
			if _, dup := byOverride[o]; dup {
				return nil, nil, nil, fmt.Errorf("PassiveSkillOverrides %s: two PassiveSkillTattoos rows", t.overrides[o].ID)
			}
			byOverride[o] = target
			continue
		}
		if old, dup := byType[typ]; dup && old != target {
			return nil, nil, nil, fmt.Errorf("PassiveSkillTattoos row %d: override type %d has targets %s and %s", i, typ, old, target)
		}
		byType[typ] = target
		placeholders[o] = true
	}
	return byOverride, byType, placeholders, nil
}

// override builds one override without its target, and returns its English name. problem
// says why it cannot be used; err is a broken reference.
func (t overrideTables) override(row overrideRow, hardmode bool, nStats int, warn func(string)) (o dataset.NodeOverride, name, problem string, err error) {
	typ, err := lookup(row.Type, t.types, "PassiveSkillOverrideTypes")
	if err != nil {
		return o, "", "", err
	}
	o = dataset.NodeOverride{ID: row.ID, Kind: dataset.Tattoo, MinNeighbours: row.MinAdjacent, MaxNeighbours: row.MaxAdjacent}
	if typ == "AlternateMastery" {
		o.Kind = dataset.Runegraft
	}
	name, icon := row.Name, row.Icon
	if len(row.Stats) != len(row.Values) {
		return o, name, fmt.Sprintf("%d stats with %d values", len(row.Stats), len(row.Values)), nil
	}
	refs, values := row.Stats, row.Values
	if _, ok := row.Passive.Row(); ok {
		p, err := lookup(row.Passive, t.passives, "PassiveSkills")
		if err != nil {
			return o, "", "", err
		}
		name, icon = cmp.Or(name, p.Name), cmp.Or(icon, p.Icon)
		ps, pv, buffs := p.stats(hardmode)
		if len(ps) > len(pv) {
			return o, name, fmt.Sprintf("passive %s has %d stats", p.Name, len(ps)), nil
		}
		refs, values = append(slices.Clone(refs), ps...), append(slices.Clone(values), pv[:len(ps)]...)
		if len(buffs) > 0 {
			warn(fmt.Sprintf("override %s: the buffs of passive %s are not carried", row.ID, p.Name))
		}
	}
	for i, ref := range refs {
		h, ok := ref.Row()
		if !ok || h >= nStats {
			return o, name, fmt.Sprintf("stat %d does not resolve", i+1), nil
		}
		o.Stats = append(o.Stats, dataset.StatValue{Stat: uint32(h), Value: values[i]})
	}
	slices.SortFunc(o.Stats, func(a, b dataset.StatValue) int { return cmp.Compare(a.Stat, b.Stat) })
	for i := 1; i < len(o.Stats); i++ {
		if o.Stats[i-1].Stat == o.Stats[i].Stat {
			return o, name, fmt.Sprintf("stat row %d repeated", o.Stats[i].Stat), nil
		}
	}
	if icon == "" {
		return o, name, "no icon", nil
	}
	// The tree export, which layout icons come from, names the game's .dds art as .png.
	o.Icon = strings.TrimSuffix(icon, ".dds") + ".png"
	blocking, err := lookupAll(row.Blocking, t.passives, "PassiveSkills")
	if err != nil {
		return o, "", "", err
	}
	for _, p := range blocking {
		o.ExcludedNodes = append(o.ExcludedNodes, uint32(p.GraphID))
	}
	slices.Sort(o.ExcludedNodes)
	o.ExcludedNodes = slices.Compact(o.ExcludedNodes)
	return o, name, "", nil
}

// overridesIn returns overrides for tree t: a node t lacks cannot be replaced, so it is
// dropped from ExcludedNodes.
func overridesIn(t dataset.Tree, overrides []dataset.NodeOverride) []dataset.NodeOverride {
	out := make([]dataset.NodeOverride, len(overrides))
	for i, o := range overrides {
		o.ExcludedNodes = slices.DeleteFunc(slices.Clone(o.ExcludedNodes), func(id uint32) bool {
			_, ok := slices.BinarySearchFunc(t.Nodes, id, func(n dataset.Node, id uint32) int { return cmp.Compare(n.ID, id) })
			return !ok
		})
		out[i] = o
	}
	return out
}
