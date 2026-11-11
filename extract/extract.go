// Package extract turns game files into pbt dataset tables.
package extract

import (
	"cmp"
	"fmt"
	"maps"
	"slices"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
	"github.com/xaroth/pbt-extract/statdesc"
)

// DescriptionsRoot is the stat description file of the passive tree, whose include chain datasets
// carry. It includes passive_skill_stat_descriptions.txt, which includes stat_descriptions.txt.
const DescriptionsRoot = "Metadata/StatDescriptions/passive_skill_aura_stat_descriptions.txt"

// Input is what a dataset is built from.
type Input struct {
	// Game reads game files by path.
	Game      ot.Reader
	Schema    *dat.Schema
	Trees     []Variant
	Constants []dataset.Constant
}

// Dataset builds the tables of one game version. The caller fills in the manifest.
func Dataset(in Input) (*dataset.Dataset, error) {
	r, schema := in.Game, in.Schema
	stats, err := readTable(r, schema, "Stats", "Id")
	if err != nil {
		return nil, err
	}
	d := &dataset.Dataset{Stats: make([]dataset.Stat, stats.t.Len())}
	handles := make(map[string]uint32, len(d.Stats))
	for i := range d.Stats {
		id, err := stats.str(i, "Id")
		if err != nil {
			return nil, err
		}
		d.Stats[i] = dataset.Stat{ID: id}
		handles[id] = uint32(i)
	}

	chars, err := readTable(r, schema, "Characters", "Id", "Name", "IntegerId",
		"BaseStrength", "BaseDexterity", "BaseIntelligence", "BaseMaxLife", "BaseMaxMana")
	if err != nil {
		return nil, err
	}
	// English text: the default Data/ tables and the tree export are English.
	text := dataset.Text{Language: "en"}
	classes := map[int]string{} // IntegerId, the tree's class index, to character id
	for i := range chars.t.Len() {
		c, err := character(r, chars, i, handles)
		if err != nil {
			return nil, fmt.Errorf("table Characters row %d: %w", i, err)
		}
		name, err := chars.str(i, "Name")
		if err != nil {
			return nil, err
		}
		classID, err := chars.integer(i, "IntegerId")
		if err != nil {
			return nil, err
		}
		if _, dup := classes[classID]; dup {
			return nil, fmt.Errorf("table Characters: IntegerId %d repeated", classID)
		}
		classes[classID] = c.ID
		d.Characters = append(d.Characters, c)
		text.CharacterNames = append(text.CharacterNames, dataset.Name{ID: c.ID, Name: name})
	}

	if text.Descriptions, err = statdesc.Load(r, append([]string{DescriptionsRoot}, SkillDescriptions()...)...); err != nil {
		return nil, err
	}
	d.Constants = in.Constants
	if d.Quests, err = quests(r, schema, len(d.Stats)); err != nil {
		return nil, err
	}
	var names itemNames
	if d.Items, d.Mods, names, err = items(r, schema, len(d.Stats)); err != nil {
		return nil, err
	}
	text.ItemClassNames, text.BaseNames, text.ModNames = names.classes, names.bases, names.mods
	if len(in.Trees) > 0 {
		tt, err := readTreeTables(r, schema, d.Stats)
		if err != nil {
			return nil, err
		}
		nodeNames := map[uint32]string{}
		for _, v := range in.Trees {
			t, err := tt.build(v, classes)
			if err != nil {
				return nil, err
			}
			d.Trees = append(d.Trees, t)
			if err := addNames(&text, nodeNames, v); err != nil {
				return nil, err
			}
		}
		for _, id := range slices.Sorted(maps.Keys(nodeNames)) {
			text.NodeNames = append(text.NodeNames, dataset.NodeName{ID: id, Name: nodeNames[id]})
		}
	}
	d.Texts = []dataset.Text{text}
	return d, nil
}

// addNames adds the node and ascendancy names of one tree export. A node keeps one name in
// every variant; a different name is an error.
func addNames(text *dataset.Text, nodes map[uint32]string, v Variant) error {
	for _, id := range slices.Sorted(maps.Keys(v.Export.Nodes)) {
		n := v.Export.Nodes[id]
		if old, ok := nodes[n.Skill]; ok && old != n.Name {
			return fmt.Errorf("tree %s node %d: name %q differs from %q in another variant", v.Name, n.Skill, n.Name, old)
		}
		nodes[n.Skill] = n.Name
	}
	for _, c := range v.Export.Classes {
		for _, a := range c.Ascendancies {
			text.AscendancyNames = append(text.AscendancyNames, dataset.AscendancyName{Variant: v.Name, ID: a.ID, Name: a.Name})
		}
	}
	for _, a := range v.Export.AlternateAscendancies {
		text.AscendancyNames = append(text.AscendancyNames, dataset.AscendancyName{Variant: v.Name, ID: a.ID, Name: a.Name})
	}
	return nil
}

func character(r ot.Reader, t table, row int, handles map[string]uint32) (dataset.Character, error) {
	var c dataset.Character
	var err error
	if c.ID, err = t.str(row, "Id"); err != nil {
		return c, err
	}
	for _, f := range []struct {
		col string
		dst *int32
	}{
		{"BaseStrength", &c.BaseStrength},
		{"BaseDexterity", &c.BaseDexterity},
		{"BaseIntelligence", &c.BaseIntelligence},
		{"BaseMaxLife", &c.BaseMaxLife},
		{"BaseMaxMana", &c.BaseMaxMana},
	} {
		if *f.dst, err = t.i32(row, f.col); err != nil {
			return c, err
		}
	}

	stats, err := ot.Stats(r, c.ID)
	if err != nil {
		return c, err
	}
	for id, v := range stats {
		h, ok := handles[id]
		if !ok {
			return c, fmt.Errorf("%s.ot: stat %s not in Stats", c.ID, id)
		}
		if int(int32(v)) != v {
			return c, fmt.Errorf("%s.ot: stat %s value %d overflows int32", c.ID, id, v)
		}
		c.Stats = append(c.Stats, dataset.StatValue{Stat: h, Value: int32(v)})
	}
	slices.SortFunc(c.Stats, func(a, b dataset.StatValue) int { return cmp.Compare(a.Stat, b.Stat) })
	return c, nil
}

type table struct {
	t    *dat.Table
	cols map[string]int
}

// readTable reads Data/<name>.datc64 and checks that the named columns exist.
func readTable(r ot.Reader, schema *dat.Schema, name string, cols ...string) (table, error) {
	def, ok := schema.Table(name)
	if !ok {
		return table{}, fmt.Errorf("table %s not in schema", name)
	}
	t := table{cols: map[string]int{}}
	for _, c := range cols {
		i, ok := def.Column(c)
		if !ok {
			return table{}, fmt.Errorf("table %s: no column %s", name, c)
		}
		t.cols[c] = i
	}
	b, err := r.ReadFile("Data/" + name + ".datc64")
	if err != nil {
		return table{}, err
	}
	if t.t, err = dat.Read(def, b); err != nil {
		return table{}, fmt.Errorf("%s: %w", name, err)
	}
	return t, nil
}

func (t table) str(row int, col string) (string, error) {
	v, err := t.value(row, col)
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("column %s: got %T, want string", col, v)
	}
	return s, nil
}

func (t table) value(row int, col string) (any, error) {
	i, ok := t.cols[col]
	if !ok {
		return nil, fmt.Errorf("column %s not read", col)
	}
	return t.t.Value(row, i)
}

// integer reads any integer column as an int.
func (t table) integer(row int, col string) (int, error) {
	v, err := t.value(row, col)
	if err != nil {
		return 0, err
	}
	switch n := v.(type) {
	case int:
		return n, nil
	case int16:
		return int(n), nil
	case uint16:
		return int(n), nil
	case int32:
		return int(n), nil
	case uint32:
		return int(n), nil
	}
	return 0, fmt.Errorf("column %s: got %T, want an integer", col, v)
}

func (t table) boolean(row int, col string) (bool, error) {
	v, err := t.value(row, col)
	if err != nil {
		return false, err
	}
	b, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("column %s: got %T, want bool", col, v)
	}
	return b, nil
}

// ref reads a row or foreignrow column; a null reference is -1.
func (t table) ref(row int, col string) (int, error) {
	v, err := t.value(row, col)
	if err != nil || v == nil {
		return -1, err
	}
	n, ok := v.(int)
	if !ok {
		return -1, fmt.Errorf("column %s: got %T, want a row reference", col, v)
	}
	return n, nil
}

// ints reads a list of row references; null entries are an error.
func (t table) ints(row int, col string) ([]int, error) {
	v, err := t.value(row, col)
	if err != nil {
		return nil, err
	}
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("column %s: got %T, want a list", col, v)
	}
	out := make([]int, len(list))
	for i, x := range list {
		if out[i], ok = x.(int); !ok {
			return nil, fmt.Errorf("column %s item %d: got %T, want a row reference", col, i, x)
		}
	}
	return out, nil
}

func (t table) i32s(row int, col string) ([]int32, error) {
	v, err := t.value(row, col)
	if err != nil {
		return nil, err
	}
	list, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("column %s: got %T, want a list", col, v)
	}
	out := make([]int32, len(list))
	for i, x := range list {
		if out[i], ok = x.(int32); !ok {
			return nil, fmt.Errorf("column %s item %d: got %T, want int32", col, i, x)
		}
	}
	return out, nil
}

func (t table) i32(row int, col string) (int32, error) {
	v, err := t.value(row, col)
	if err != nil {
		return 0, err
	}
	n, ok := v.(int32)
	if !ok {
		return 0, fmt.Errorf("column %s: got %T, want int32", col, v)
	}
	return n, nil
}
