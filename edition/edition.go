// Package edition holds what pbt-extract knows about a game edition that its game data does
// not say, such as constants, names for columns dat-schema leaves unnamed, totem links and how
// older game files number a column's values, each with a note and a source. Each edition has a
// folder: <game>/constants.json, <game>/columns.json, <game>/totems.json and <game>/values.json.
package edition

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"

	"github.com/xaroth/pbt-engine/dataset"

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
	Values    []Values
}

// Values is how the game files of a league and those before it number a column whose values
// the dat-schema enumeration numbers otherwise.
type Values struct {
	Table  string `json:"table"`
	Column string `json:"column"`
	// Until is the last league whose game files number the column this way, such as 3.27.
	Until string `json:"until"`
	// Map gives the schema's value for each value those game files hold; a value it lacks is
	// an error.
	Map    map[uint32]uint32 `json:"map"`
	Note   string            `json:"note"`
	Source string            `json:"source"`
}

// ValuesFor returns the value maps that apply to game version, such as 3.27.0.10.
func (e *Edition) ValuesFor(version string) ([]Values, error) {
	v, err := dataset.ParseGameVersion(version)
	if err != nil {
		return nil, err
	}
	var out []Values
	for _, x := range e.Values {
		until, err := dataset.ParseGameVersion(x.Until)
		if err != nil {
			return nil, fmt.Errorf("%s/values.json: %s.%s: %w", e.Name, x.Table, x.Column, err)
		}
		if v.League().Compare(until) <= 0 {
			out = append(out, x)
		}
	}
	return out, nil
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
// source; every column name a table, after, gap, position, name, note and source; every value
// map a table, column, league, map, note and source.
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
	var v struct {
		Values []Values `json:"values"`
	}
	if err := readJSON(game, "values.json", &v); err != nil {
		return nil, err
	}
	for _, k := range v.Values {
		if k.Table == "" || k.Column == "" || len(k.Map) == 0 || k.Note == "" || k.Source == "" {
			return nil, fmt.Errorf("%s/values.json: values of %s.%s need table, column, until, map, note and source", game, k.Table, k.Column)
		}
		if l, err := dataset.ParseGameVersion(k.Until); err != nil || l.League() != l {
			return nil, fmt.Errorf("%s/values.json: values of %s.%s: until %q is not a league", game, k.Table, k.Column, k.Until)
		}
	}
	e.Values = v.Values
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
