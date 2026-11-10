// Package edition holds what pbt-extract knows about a game edition that its game data does
// not say, such as constants and names for columns dat-schema leaves unnamed, each with a note
// and a source. <game>.json holds it per edition.
package edition

import (
	"embed"
	"encoding/json"
	"fmt"

	"github.com/xaroth/pbt-extract/dat"
)

//go:embed *.json
var files embed.FS

// Edition is one game edition, such as poe1.
type Edition struct {
	Name      string
	Constants []Constant
	Columns   []ColumnName
}

// Constant is an edition constant, as Value per Per.
type Constant struct {
	ID     string `json:"id"`
	Value  int32  `json:"value"`
	Per    int32  `json:"per"`
	Note   string `json:"note"`
	Source string `json:"source"`
}

// ColumnName is our name for a column dat-schema leaves unnamed, at Index of Table, with
// dat-schema type Type.
type ColumnName struct {
	Table  string `json:"table"`
	Index  int    `json:"index"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Note   string `json:"note"`
	Source string `json:"source"`
}

// NameColumns gives the columns of s our names. It fails when dat-schema names one of them or
// its type differs (see dat.Schema.NameColumn).
func (e *Edition) NameColumns(s *dat.Schema) error {
	for _, c := range e.Columns {
		if err := s.NameColumn(c.Table, c.Index, c.Name, c.Type); err != nil {
			return fmt.Errorf("edition %s column name %s: %w", e.Name, c.Name, err)
		}
	}
	return nil
}

// Load returns the edition game from <game>.json. Every constant needs an id, per, note and
// source; every column name a table, type, name, note and source.
func Load(game string) (*Edition, error) {
	b, err := files.ReadFile(game + ".json")
	if err != nil {
		return nil, fmt.Errorf("edition %s: %w", game, err)
	}
	var f struct {
		Constants []Constant   `json:"constants"`
		Columns   []ColumnName `json:"columns"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s.json: %w", game, err)
	}
	for _, c := range f.Constants {
		if c.ID == "" || c.Per == 0 || c.Note == "" || c.Source == "" {
			return nil, fmt.Errorf("%s.json: constant %q needs id, per, note and source", game, c.ID)
		}
	}
	for _, c := range f.Columns {
		if c.Table == "" || c.Type == "" || c.Name == "" || c.Note == "" || c.Source == "" {
			return nil, fmt.Errorf("%s.json: column name %q needs table, type, name, note and source", game, c.Name)
		}
	}
	return &Edition{Name: game, Constants: f.Constants, Columns: f.Columns}, nil
}
