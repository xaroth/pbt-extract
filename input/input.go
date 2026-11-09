// Package input reads the input directory of a dataset build:
//
//	schema.json   the dat-schema build for the game version
//	tree/         data.json, ruthless.json and alternate.json of the skilltree export
//	Bundles2/     the game's bundles: _.index.bin and the bundles the build reads
//
// How the files got there is the caller's business.
package input

import (
	"os"
	"path/filepath"
)

// Names of the input directory's entries.
const (
	SchemaFile = "schema.json"
	TreeDir    = "tree"
)

// Dir is an input directory.
type Dir string

// Schema returns schema.json.
func (d Dir) Schema() ([]byte, error) { return os.ReadFile(filepath.Join(string(d), SchemaFile)) }

// TreeFile returns the tree export file name, such as data.json.
func (d Dir) TreeFile(name string) ([]byte, error) {
	return os.ReadFile(filepath.Join(string(d), TreeDir, name))
}
