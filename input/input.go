// Package input reads the input directory of a dataset build:
//
//	source.json   {"game": "poe1", "version": "3.29.3.3", "source": {...}}
//	schema.json   the dat-schema build for the version
//	tree/         data.json, ruthless.json and alternate.json of the skilltree export
//	Bundles2/     the game's bundles: _.index.bin and the bundles the build reads
//
// How the files got there is the caller's business; source is copied into the dataset's
// manifest as given.
package input

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Names of the input directory's entries.
const (
	SourceFile = "source.json"
	SchemaFile = "schema.json"
	TreeDir    = "tree"
)

var errSource = errors.New("source.json needs game and version")

// Source is source.json.
type Source struct {
	Game    string `json:"game"`
	Version string `json:"version"`
	// Source says where the inputs came from, such as the Steam depot manifest; it goes into
	// the manifest unread.
	Source map[string]string `json:"source"`
}

// Dir is an input directory.
type Dir struct {
	Path string
	Source
}

// Open reads dir's source.json.
func Open(dir string) (*Dir, error) {
	b, err := os.ReadFile(filepath.Join(dir, SourceFile))
	if err != nil {
		return nil, err
	}
	d := &Dir{Path: dir}
	if err := json.Unmarshal(b, &d.Source); err != nil {
		return nil, fmt.Errorf("%s: %w", SourceFile, err)
	}
	if d.Game == "" || d.Version == "" {
		return nil, errSource
	}
	return d, nil
}

// Schema returns schema.json.
func (d *Dir) Schema() ([]byte, error) { return os.ReadFile(filepath.Join(d.Path, SchemaFile)) }

// TreeFile returns the tree export file name, such as data.json.
func (d *Dir) TreeFile(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(d.Path, TreeDir, name))
}
