// Package dat reads .datc64 game tables using dat-schema table definitions
// (https://github.com/poe-tool-dev/dat-schema).
package dat

import (
	"encoding/json"
	"fmt"
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
