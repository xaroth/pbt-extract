// Package versions holds, per game version, the dat-schema build and where the game files
// come from. Add a version with scripts/add-version.sh.
package versions

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
)

//go:embed poe1
var files embed.FS

// Source says where a version's game files and schema come from.
type Source struct {
	SteamManifest   string `json:"steamManifest"`
	DatSchemaCommit string `json:"datSchemaCommit"`
	// DatSchemaNote says why DatSchemaCommit is not the commit closest after the patch.
	DatSchemaNote string `json:"datSchemaNote,omitempty"`
	// TreeExport is the tag of https://github.com/grindinggear/skilltree-export to use.
	TreeExport string `json:"treeExport"`
	// TreeExportCommit is the commit TreeExport pointed at when the version was added.
	TreeExportCommit string `json:"treeExportCommit"`
	// TreeExportFiles maps each export file used (data.json, ...) to its SHA-256.
	TreeExportFiles map[string]string `json:"treeExportFiles"`
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

// CheckTreeExport compares a game version with the skilltree-export tag it uses. A tag from
// another league (major.minor) is an error. A tag older than the version returns a warning:
// GGG does not tag an export for every patch, so the graph may miss a hotfix change.
func CheckTreeExport(version, tag string) (warning string, err error) {
	v, err := parseVersion(version)
	if err != nil {
		return "", err
	}
	t, err := parseVersion(tag)
	if err != nil {
		return "", fmt.Errorf("tree export tag: %w", err)
	}
	if len(v) < 2 || len(t) < 2 || v[0] != t[0] || v[1] != t[1] {
		return "", fmt.Errorf("tree export %s is not from the league of game version %s", tag, version)
	}
	if slices.Compare(t, v) < 0 {
		return fmt.Sprintf("tree export %s is older than game version %s; graph changes after %s are missing", tag, version, tag), nil
	}
	return "", nil
}

func parseVersion(s string) ([]int, error) {
	var out []int
	for p := range strings.SplitSeq(s, ".") {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("version %q: want numbers separated by dots", s)
		}
		out = append(out, n)
	}
	return out, nil
}
