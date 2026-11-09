// Package constants holds per edition the constants the game data does not contain, each with
// a note and a source.
package constants

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed *.json
var files embed.FS

// Constant is an edition constant, as Value per Per.
type Constant struct {
	ID     string `json:"id"`
	Value  int32  `json:"value"`
	Per    int32  `json:"per"`
	Note   string `json:"note"`
	Source string `json:"source"`
}

// Load returns the constants of game from <game>.json. Every constant needs an id, per, note
// and source.
func Load(game string) ([]Constant, error) {
	b, err := files.ReadFile(game + ".json")
	if err != nil {
		return nil, err
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
	return f.Constants, nil
}
