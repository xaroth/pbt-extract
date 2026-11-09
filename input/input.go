// Package input reads the input directory of a dataset build:
//
//	source.json   {"version": "3.29.3.3", ...}: the game version and where the inputs came from
//	schema.json   the dat-schema build for the version
//	tree/         data.json, ruthless.json and alternate.json of the skilltree export
//	Bundles2/     the game's bundles: _.index.bin and the bundles the build reads
//
// How the files got there is the caller's business; source.json goes into the dataset's
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

var errVersion = errors.New("source.json needs a version")

// Dir is an input directory.
type Dir struct {
	Path string
	// Source is source.json: string fields, of which version is the game version.
	Source map[string]string
}

// Version returns the game version from source.json.
func (d *Dir) Version() string { return d.Source["version"] }

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
	if d.Version() == "" {
		return nil, errVersion
	}
	return d, nil
}

// Schema returns schema.json.
func (d *Dir) Schema() ([]byte, error) { return os.ReadFile(filepath.Join(d.Path, SchemaFile)) }

// TreeFile returns the tree export file name, such as data.json.
func (d *Dir) TreeFile(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(d.Path, TreeDir, name))
}
