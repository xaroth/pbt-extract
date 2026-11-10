// Package edition holds what pbt-extract knows about a game edition that its game data does
// not say, such as constants, each with a note and a source. <game>.json holds it per edition.
package edition

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed *.json
var files embed.FS

// Edition is one game edition, such as poe1.
type Edition struct {
	Name      string
	Constants []Constant
}

// Constant is an edition constant, as Value per Per.
type Constant struct {
	ID     string `json:"id"`
	Value  int32  `json:"value"`
	Per    int32  `json:"per"`
	Note   string `json:"note"`
	Source string `json:"source"`
}

// Load returns the edition game from <game>.json. Every constant needs an id, per, note and
// source.
func Load(game string) (*Edition, error) {
	b, err := files.ReadFile(game + ".json")
	if err != nil {
		return nil, fmt.Errorf("edition %s: %w", game, err)
	}
	var f struct {
		Constants []Constant `json:"constants"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s.json: %w", game, err)
	}
	for _, c := range f.Constants {
		if c.ID == "" || c.Per == 0 || c.Note == "" || c.Source == "" {
			return nil, fmt.Errorf("%s.json: constant %q needs id, per, note and source", game, c.ID)
		}
	}
	return &Edition{Name: game, Constants: f.Constants}, nil
}
