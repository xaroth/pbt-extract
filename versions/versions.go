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
	if _, err := fs.Stat(files, path.Join(game, name)); err != nil {
		return Version{}, err
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
