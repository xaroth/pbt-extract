// Package build turns game files into pbt dataset tables.
package build

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
)

// Dataset reads the tables of one game version from r using schema. The caller fills in the
// manifest.
func Dataset(r ot.Reader, schema *dat.Schema) (*dataset.Dataset, error) {
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

	chars, err := readTable(r, schema, "Characters", "Id", "Name",
		"BaseStrength", "BaseDexterity", "BaseIntelligence", "BaseMaxLife", "BaseMaxMana")
	if err != nil {
		return nil, err
	}
	for i := range chars.t.Len() {
		c, err := character(r, chars, i, handles)
		if err != nil {
			return nil, fmt.Errorf("table Characters row %d: %w", i, err)
		}
		d.Characters = append(d.Characters, c)
	}
	return d, nil
}

func character(r ot.Reader, t table, row int, handles map[string]uint32) (dataset.Character, error) {
	var c dataset.Character
	var err error
	if c.ID, err = t.str(row, "Id"); err != nil {
		return c, err
	}
	if c.Name, err = t.str(row, "Name"); err != nil {
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
	v, err := t.t.Value(row, t.cols[col])
	if err != nil {
		return "", err
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("column %s: got %T, want string", col, v)
	}
	return s, nil
}

func (t table) i32(row int, col string) (int32, error) {
	v, err := t.t.Value(row, t.cols[col])
	if err != nil {
		return 0, err
	}
	n, ok := v.(int32)
	if !ok {
		return 0, fmt.Errorf("column %s: got %T, want int32", col, v)
	}
	return n, nil
}
