// Package dat reads .datc64 game tables using dat-schema table definitions
// (https://github.com/poe-tool-dev/dat-schema).
package dat

import (
	"encoding/json"
	"fmt"
	"slices"
)

// Schema is a parsed dat-schema build for one game.
type Schema struct {
	tables map[string]TableDef
}

// TableDef describes the columns of one table, in file order.
type TableDef struct {
	Name    string   `json:"name"`
	Columns []Column `json:"columns"`
}

// Column is one column of a table. Type is one of bool, i16, u16, i32, u32, f32, string,
// row (index into the same table), foreignrow (index into References.Table), enumrow
// (enumeration index) or array (list of unknown element type).
type Column struct {
	Name       *string    `json:"name"`
	Type       string     `json:"type"`
	Array      bool       `json:"array"`
	Interval   bool       `json:"interval"`
	Localized  bool       `json:"localized"`
	References *Reference `json:"references"`
}

// Reference names the table, and optionally the column, a row column points into.
type Reference struct {
	Table  string  `json:"table"`
	Column *string `json:"column"`
}

var scalarSizes = map[string]int{
	"bool": 1, "i16": 2, "u16": 2, "i32": 4, "u32": 4, "f32": 4,
	"string": 8, "row": 8, "foreignrow": 16, "enumrow": 4,
}

const listSize = 16

// ParseSchema parses dat-schema JSON (schema.min.json or a per-game file). It rejects column
// types it does not know, so a format change fails here instead of misreading tables.
func ParseSchema(b []byte) (*Schema, error) {
	var raw struct {
		Version int        `json:"version"`
		Tables  []TableDef `json:"tables"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	if raw.Version != 7 {
		return nil, fmt.Errorf("dat-schema format version %d, want 7", raw.Version)
	}
	s := &Schema{tables: make(map[string]TableDef, len(raw.Tables))}
	for _, t := range raw.Tables {
		for i, c := range t.Columns {
			if _, err := c.size(); err != nil {
				return nil, fmt.Errorf("table %s column %d: %w", t.Name, i, err)
			}
		}
		if _, dup := s.tables[t.Name]; dup {
			return nil, fmt.Errorf("table %s defined twice", t.Name)
		}
		s.tables[t.Name] = t
	}
	return s, nil
}

// Table returns the definition of the named table.
func (s *Schema) Table(name string) (TableDef, bool) {
	t, ok := s.tables[name]
	return t, ok
}

func (c Column) size() (int, error) {
	if c.Array {
		if c.Interval {
			return 0, fmt.Errorf("interval list not supported")
		}
		if c.Type == "array" {
			return listSize, nil
		}
	}
	n, ok := scalarSizes[c.Type]
	if !ok {
		return 0, fmt.Errorf("unknown column type %q", c.Type)
	}
	switch {
	case c.Array:
		return listSize, nil
	case c.Interval:
		return 2 * n, nil
	}
	return n, nil
}

// RowSize is the byte size of one row in the fixed section.
func (t TableDef) RowSize() int {
	n := 0
	for _, c := range t.Columns {
		s, _ := c.size() // validated in ParseSchema
		n += s
	}
	return n
}

// Column returns the index of the column with the given name.
func (t TableDef) Column(name string) (int, bool) {
	for i, c := range t.Columns {
		if c.Name != nil && *c.Name == name {
			return i, true
		}
	}
	return 0, false
}

// Kind returns the column's type as dat-schema writes it: "i32", "[i32]" for a list, "interval
// i32" for an interval.
func (c Column) Kind() string {
	switch {
	case c.Array:
		return "[" + c.Type + "]"
	case c.Interval:
		return "interval " + c.Type
	}
	return c.Type
}

// ColumnName is our name for a column dat-schema leaves unnamed. Unnamed columns sit in gaps
// between named ones, so a column is located by the named column After that starts its gap and
// its Position (from 0) in that gap; Gap lists the Kind of every column in the gap. Unlike a
// column index, this holds across patches that add or remove columns elsewhere in the table.
type ColumnName struct {
	Table    string
	After    string
	Gap      []string
	Position int
	Name     string
}

// NameColumns gives unnamed columns our names. Every column is located before any is named, so
// on error s is unchanged. It fails when a table or After column is missing, when the gap after
// After differs from Gap (dat-schema named, added, removed or retyped a column in it), or when
// a name is taken.
func (s *Schema) NameColumns(names []ColumnName) error {
	index := make([]int, len(names))
	for i, n := range names {
		var err error
		if index[i], err = s.locate(n); err != nil {
			return fmt.Errorf("table %s: column name %s: %w", n.Table, n.Name, err)
		}
	}
	for i, n := range names {
		t := s.tables[n.Table]
		if _, taken := t.Column(n.Name); taken {
			return fmt.Errorf("table %s: column name %s already taken", n.Table, n.Name)
		}
		t.Columns[index[i]].Name = &n.Name
	}
	return nil
}

func (s *Schema) locate(n ColumnName) (int, error) {
	t, ok := s.tables[n.Table]
	if !ok {
		return 0, fmt.Errorf("table not in schema")
	}
	after, ok := t.Column(n.After)
	if !ok {
		return 0, fmt.Errorf("no column %s", n.After)
	}
	if n.Position < 0 || n.Position >= len(n.Gap) {
		return 0, fmt.Errorf("position %d outside a gap of %d", n.Position, len(n.Gap))
	}
	var gap []string
	for _, c := range t.Columns[after+1:] {
		if c.Name != nil {
			break
		}
		gap = append(gap, c.Kind())
	}
	if !slices.Equal(gap, n.Gap) {
		return 0, fmt.Errorf("unnamed columns after %s are %v, want %v", n.After, gap, n.Gap)
	}
	return after + 1 + n.Position, nil
}
