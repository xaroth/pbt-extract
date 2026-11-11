package extract

import (
	"fmt"
	"slices"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
)

type buffDefinitionRow struct {
	ID             string    `dat:"Id"`
	Name           string    `dat:"Name"`
	Category       int32     `dat:"BuffCategory"`
	ID2            string    `dat:"Id2"`
	Stats          []dat.Ref `dat:"StatsKeys" ref:"Stats"`
	ExtraStats     []dat.Ref `dat:"Stats2" ref:"Stats"`
	GrantedStats   []dat.Ref `dat:"GrantedStats" ref:"Stats"`
	Flags          []dat.Ref `dat:"GrantedFlags" ref:"Stats"`
	ConditionStats []dat.Ref `dat:"ConditionStats" ref:"Stats"`
	MaximumStat    dat.Ref   `dat:"Maximum_StatsKey" ref:"Stats"`
	CurrentStat    dat.Ref   `dat:"Current_StatsKey" ref:"Stats"`
	MinimumStat    dat.Ref   `dat:"MinStat" ref:"Stats"`
	Limit          int32     `dat:"BuffLimit"`
	SkillBuff      bool      `dat:"IsSkillBuff"`
	Invisible      bool      `dat:"Invisible"`
	Removable      bool      `dat:"Removable"`
	CancelOnDeath  bool      `dat:"CancelOnDeath"`
	Recovery       bool      `dat:"IsRecovery"`
}

// flaskBuffRow reads BuffStatValues and not BuffStatValues2, which equals it on every row.
type flaskBuffRow struct {
	Base   dat.Ref `dat:"BaseItemTypesKey" ref:"BaseItemTypes"`
	Buff   dat.Ref `dat:"BuffDefinitionsKey" ref:"BuffDefinitions"`
	Values []int32 `dat:"BuffStatValues"`
}

type tinctureRow struct {
	Base           dat.Ref `dat:"BaseItem" ref:"BaseItemTypes"`
	DebuffInterval int32   `dat:"DebuffInterval"`
	Cooldown       int32   `dat:"Cooldown"`
}

// buffs reads the buffs group: every BuffDefinitions row, the Flasks rows with a buff and
// every Tinctures row, in table order, and the buff names. nStats is the number of rows in
// Stats; flasks and tinctures must be bases of items.
func buffs(r ot.Reader, s *dat.Schema, nStats int, items *dataset.Items) (*dataset.Buffs, []dataset.Name, error) {
	rows, err := readRows[buffDefinitionRow](r, s, "BuffDefinitions")
	if err != nil {
		return nil, nil, err
	}
	b := &dataset.Buffs{}
	var names []dataset.Name
	for i, row := range rows {
		d, err := buffDefinition(row, nStats)
		if err != nil {
			return nil, nil, fmt.Errorf("table BuffDefinitions row %d: %w", i, err)
		}
		b.Definitions = append(b.Definitions, d)
		if row.Name != "" {
			names = append(names, dataset.Name{ID: row.ID, Name: row.Name})
		}
	}
	bases, err := readIDs(r, s, "BaseItemTypes")
	if err != nil {
		return nil, nil, err
	}
	kept := map[string]bool{}
	for _, base := range items.Bases {
		kept[base.ID] = true
	}
	if b.Flasks, err = flaskBuffs(r, s, b.Definitions, bases, kept); err != nil {
		return nil, nil, err
	}
	if b.Tinctures, err = tinctures(r, s, bases, kept); err != nil {
		return nil, nil, err
	}
	return b, names, nil
}

func buffDefinition(row buffDefinitionRow, nStats int) (dataset.BuffDefinition, error) {
	d := dataset.BuffDefinition{
		ID: row.ID, Category: row.Category, ID2: row.ID2, Limit: row.Limit,
		SkillBuff: row.SkillBuff, Invisible: row.Invisible, Removable: row.Removable,
		CancelOnDeath: row.CancelOnDeath, Recovery: row.Recovery,
	}
	// The group stores StatsKeys and Stats2 for GrantedStats; this holds in every row so far.
	if !slices.Equal(row.GrantedStats, slices.Concat(row.Stats, row.ExtraStats)) {
		return d, fmt.Errorf("GrantedStats is not StatsKeys followed by Stats2")
	}
	for _, l := range []struct {
		col  string
		refs []dat.Ref
		dst  *[]uint32
	}{
		{"StatsKeys", row.Stats, &d.Stats}, {"Stats2", row.ExtraStats, &d.ExtraStats},
		{"GrantedFlags", row.Flags, &d.Flags}, {"ConditionStats", row.ConditionStats, &d.ConditionStats},
	} {
		hs, err := statHandles(l.refs, nStats)
		if err != nil {
			return d, fmt.Errorf("column %s: %w", l.col, err)
		}
		*l.dst = hs
	}
	for _, s := range []struct {
		col string
		ref dat.Ref
		dst **uint32
	}{
		{"Maximum_StatsKey", row.MaximumStat, &d.MaximumStat}, {"Current_StatsKey", row.CurrentStat, &d.CurrentStat},
		{"MinStat", row.MinimumStat, &d.MinimumStat},
	} {
		h, err := optionalStat(s.ref, nStats)
		if err != nil {
			return d, fmt.Errorf("column %s: %w", s.col, err)
		}
		*s.dst = h
	}
	return d, nil
}

// flaskBuffs returns the Flasks rows with a buff. bases are the BaseItemTypes ids and kept the
// bases of the items group by id.
func flaskBuffs(r ot.Reader, s *dat.Schema, defs []dataset.BuffDefinition, bases []string, kept map[string]bool) ([]dataset.FlaskBuff, error) {
	rows, err := readRows[flaskBuffRow](r, s, "Flasks")
	if err != nil {
		return nil, err
	}
	var out []dataset.FlaskBuff
	for i, row := range rows {
		if _, ok := row.Buff.Row(); !ok {
			continue
		}
		f, err := flaskBuff(row, defs, bases, kept)
		if err != nil {
			return nil, fmt.Errorf("table Flasks row %d: %w", i, err)
		}
		out = append(out, f)
	}
	return out, nil
}

func flaskBuff(row flaskBuffRow, defs []dataset.BuffDefinition, bases []string, kept map[string]bool) (dataset.FlaskBuff, error) {
	item, err := keptBase(row.Base, bases, kept)
	if err != nil {
		return dataset.FlaskBuff{}, err
	}
	def, err := lookup(row.Buff, defs, "BuffDefinitions")
	if err != nil {
		return dataset.FlaskBuff{}, err
	}
	if want := len(def.Stats) + len(def.ExtraStats); len(row.Values) != want {
		return dataset.FlaskBuff{}, fmt.Errorf("%d values for the %d stats of buff %s", len(row.Values), want, def.ID)
	}
	return dataset.FlaskBuff{Item: item, Buff: def.ID, Values: row.Values}, nil
}

func tinctures(r ot.Reader, s *dat.Schema, bases []string, kept map[string]bool) ([]dataset.Tincture, error) {
	rows, err := readRows[tinctureRow](r, s, "Tinctures")
	if err != nil {
		return nil, err
	}
	var out []dataset.Tincture
	for i, row := range rows {
		item, err := keptBase(row.Base, bases, kept)
		if err != nil {
			return nil, fmt.Errorf("table Tinctures row %d: %w", i, err)
		}
		out = append(out, dataset.Tincture{Item: item, DebuffInterval: row.DebuffInterval, Cooldown: row.Cooldown})
	}
	return out, nil
}

// keptBase returns the id of the base ref references, which must be a base of the items group.
func keptBase(ref dat.Ref, bases []string, kept map[string]bool) (string, error) {
	id, err := lookup(ref, bases, "BaseItemTypes")
	if err != nil {
		return "", err
	}
	if !kept[id] {
		return "", fmt.Errorf("base %s is not in the items group", id)
	}
	return id, nil
}
