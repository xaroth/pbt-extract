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

// ColumnName is our name for a column dat-schema leaves unnamed, with its note and source.
type ColumnName struct {
	dat.ColumnName
	Note   string
	Source string
}

// ColumnNames returns the column names of the edition's tables, for dat.Schema.NameColumns.
func (e *Edition) ColumnNames() []dat.ColumnName {
	out := make([]dat.ColumnName, len(e.Columns))
	for i, c := range e.Columns {
		out[i] = c.ColumnName
	}
	return out
}

// Load returns the edition game from <game>.json. Every constant needs an id, per, note and
// source; every column name a table, after, gap, position, name, note and source.
func Load(game string) (*Edition, error) {
	b, err := files.ReadFile(game + ".json")
	if err != nil {
		return nil, fmt.Errorf("edition %s: %w", game, err)
	}
	var f struct {
		Constants []Constant `json:"constants"`
		Columns   []struct {
			Table    string   `json:"table"`
			After    string   `json:"after"`
			Gap      []string `json:"gap"`
			Position *int     `json:"position"`
			Name     string   `json:"name"`
			Note     string   `json:"note"`
			Source   string   `json:"source"`
		} `json:"columns"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s.json: %w", game, err)
	}
	for _, c := range f.Constants {
		if c.ID == "" || c.Per == 0 || c.Note == "" || c.Source == "" {
			return nil, fmt.Errorf("%s.json: constant %q needs id, per, note and source", game, c.ID)
		}
	}
	e := &Edition{Name: game, Constants: f.Constants}
	for _, c := range f.Columns {
		if c.Table == "" || c.After == "" || len(c.Gap) == 0 || c.Position == nil || c.Name == "" || c.Note == "" || c.Source == "" {
			return nil, fmt.Errorf("%s.json: column name %q needs table, after, gap, position, name, note and source", game, c.Name)
		}
		e.Columns = append(e.Columns, ColumnName{
			Table: c.Table, After: c.After, Gap: c.Gap, Position: *c.Position, Name: c.Name,
			Note:   c.Note,
			Source: c.Source,
		})
	}
	return e, nil
}
