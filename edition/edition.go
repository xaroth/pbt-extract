// Package edition holds what pbt-extract knows about a game edition that its game data does
// not say, such as constants, names for columns dat-schema leaves unnamed and totem links, each
// with a note and a source. Each edition has a folder: <game>/constants.json,
// <game>/columns.json and <game>/totems.json.
package edition

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"

	"github.com/xaroth/pbt-extract/dat"
)

//go:embed */*.json
var files embed.FS

// Edition is one game edition, such as poe1.
type Edition struct {
	Name      string
	Constants []Constant
	Columns   []ColumnName
	Totems    []Totem
}

// Totem is a link the game data does not hold: the SkillTotemVariations key whose varieties
// the totems of a support use.
type Totem struct {
	// GrantedEffect is the support's GrantedEffects.Id.
	GrantedEffect string `json:"granted_effect"`
	Totem         int32  `json:"totem"`
	Note          string `json:"note"`
	Source        string `json:"source"`
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

// Load returns the edition game from its folder. Every constant needs an id, per, note and
// source; every column name a table, after, gap, position, name, note and source.
func Load(game string) (*Edition, error) {
	var c struct {
		Constants []Constant `json:"constants"`
	}
	if err := readJSON(game, "constants.json", &c); err != nil {
		return nil, err
	}
	for _, k := range c.Constants {
		if k.ID == "" || k.Per == 0 || k.Note == "" || k.Source == "" {
			return nil, fmt.Errorf("%s/constants.json: constant %q needs id, per, note and source", game, k.ID)
		}
	}
	var n struct {
		Columns []struct {
			Table    string   `json:"table"`
			After    string   `json:"after"`
			Gap      []string `json:"gap"`
			Position *int     `json:"position"`
			Name     string   `json:"name"`
			Note     string   `json:"note"`
			Source   string   `json:"source"`
		} `json:"columns"`
	}
	if err := readJSON(game, "columns.json", &n); err != nil {
		return nil, err
	}
	e := &Edition{Name: game, Constants: c.Constants}
	for _, k := range n.Columns {
		if k.Table == "" || k.After == "" || len(k.Gap) == 0 || k.Position == nil || k.Name == "" || k.Note == "" || k.Source == "" {
			return nil, fmt.Errorf("%s/columns.json: column name %q needs table, after, gap, position, name, note and source", game, k.Name)
		}
		e.Columns = append(e.Columns, ColumnName{
			Table: k.Table, After: k.After, Gap: k.Gap, Position: *k.Position, Name: k.Name,
			Note:   k.Note,
			Source: k.Source,
		})
	}
	var t struct {
		Totems []Totem `json:"totems"`
	}
	if err := readJSON(game, "totems.json", &t); err != nil {
		return nil, err
	}
	for _, k := range t.Totems {
		if k.GrantedEffect == "" || k.Note == "" || k.Source == "" {
			return nil, fmt.Errorf("%s/totems.json: totem of %q needs granted_effect, note and source", game, k.GrantedEffect)
		}
	}
	e.Totems = t.Totems
	return e, nil
}

func readJSON(game, name string, v any) error {
	b, err := files.ReadFile(path.Join(game, name))
	if err != nil {
		return fmt.Errorf("edition %s: %w", game, err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("%s/%s: %w", game, name, err)
	}
	return nil
}
