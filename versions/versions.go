// Package versions holds, per game version, the dat-schema build and where the game files
// come from. Add a version with scripts/add-version.sh.
package versions

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
)

//go:embed poe1
var files embed.FS

// Source says where a version's game files and schema come from.
type Source struct {
	SteamManifest   string `json:"steamManifest"`
	DatSchemaCommit string `json:"datSchemaCommit"`
	// TreeExport is the tag of https://github.com/grindinggear/skilltree-export to use.
	TreeExport string `json:"treeExport"`
}

// Version is one game version known to the extractor.
type Version struct {
	Game, Name string
}

// Schema returns the dat-schema JSON for v.
func (v Version) Schema() ([]byte, error) {
	return fs.ReadFile(files, path.Join(v.Game, v.Name, "schema.json"))
}

// Source returns where v's files come from.
func (v Version) Source() (Source, error) {
	b, err := fs.ReadFile(files, path.Join(v.Game, v.Name, "source.json"))
	if err != nil {
		return Source{}, err
	}
	var s Source
	if err := json.Unmarshal(b, &s); err != nil {
		return Source{}, fmt.Errorf("%s/%s/source.json: %w", v.Game, v.Name, err)
	}
	return s, nil
}

// Lookup returns the version with the given name, or an error wrapping fs.ErrNotExist.
func Lookup(game, name string) (Version, error) {
	st, err := fs.Stat(files, path.Join(game, name))
	if err != nil {
		return Version{}, err
	}
	if !st.IsDir() {
		return Version{}, fmt.Errorf("%s/%s is not a version: %w", game, name, fs.ErrNotExist)
	}
	return Version{Game: game, Name: name}, nil
}

// All returns every known version of game, sorted by name.
func All(game string) ([]Version, error) {
	entries, err := fs.ReadDir(files, game)
	if err != nil {
		return nil, err
	}
	var out []Version
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, Version{Game: game, Name: e.Name()})
		}
	}
	return out, nil
}

// Constant is an edition constant the game data does not hold, as Value per Per.
type Constant struct {
	ID     string `json:"id"`
	Value  int32  `json:"value"`
	Per    int32  `json:"per"`
	Note   string `json:"note"`
	Source string `json:"source"`
}

// Constants returns the edition constants of game from <game>/constants.json. Every constant
// needs a note and a source.
func Constants(game string) ([]Constant, error) {
	b, err := fs.ReadFile(files, path.Join(game, "constants.json"))
	if err != nil {
		return nil, err
	}
	var f struct {
		Constants []Constant `json:"constants"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s/constants.json: %w", game, err)
	}
	for _, c := range f.Constants {
		if c.ID == "" || c.Per == 0 || c.Note == "" || c.Source == "" {
			return nil, fmt.Errorf("%s/constants.json: constant %q needs id, per, note and source", game, c.ID)
		}
	}
	return f.Constants, nil
}
