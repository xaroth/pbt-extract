// Command pbt-extract builds pbt datasets from Path of Exile game files.
//
//	pbt-extract [-ooz path] [-game poe1] -version <v> dataset <in> <out>
//	pbt-extract [-ooz path] art <in> <out>
//	pbt-extract [-ooz path] bundles <in>
//	pbt-extract [-ooz path] get <in> <path>
//	pbt-extract [-ooz path] [-game poe1] table <in> <table>
//	pbt-extract [-ooz path] stats <in> <object>
//
// in is an input directory (see package input): schema.json, the tree export under tree/ and
// the game's Bundles2. dataset writes the dataset of game version -version into the directory
// out. art writes the tree's sprite sheets from the export's assets folder and the faces of
// the base classes and their ascendancies from the game's UI art into out, with a manifest placing
// each icon on the sheets and naming each face.
// bundles lists the bundle files under in that a dataset or art build may read, one per line, from
// Bundles2/_.index.bin alone. get writes the game file at path (for example Data/Stats.datc64)
// to stdout. table decodes Data/<table>.datc64 with schema.json, with the -game edition's names
// for the table's unnamed columns, and writes one JSON object per row. stats writes the Stats block of an object template such as Metadata/Characters/Str/Str,
// merged over its extends chain, as one JSON object.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/art"
	"github.com/xaroth/pbt-extract/bundle"
	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/edition"
	"github.com/xaroth/pbt-extract/extract"
	"github.com/xaroth/pbt-extract/input"
	"github.com/xaroth/pbt-extract/ooz"
	"github.com/xaroth/pbt-extract/ot"
	"github.com/xaroth/pbt-extract/treeexport"
)

const usage = "usage: pbt-extract [-ooz path] [-game poe1] -version <v> dataset <in> <out>\n       pbt-extract [-ooz path] art <in> <out>\n       pbt-extract [-ooz path] bundles <in>\n       pbt-extract [-ooz path] get <in> <path>\n       pbt-extract [-ooz path] [-game poe1] table <in> <table>\n       pbt-extract [-ooz path] stats <in> <object>"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "pbt-extract:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fl := flag.NewFlagSet("pbt-extract", flag.ContinueOnError)
	oozPath := fl.String("ooz", "", "path to the ooz binary (default: ooz from PATH)")
	game := fl.String("game", "poe1", "game edition of the input (dataset, table)")
	version := fl.String("version", "", "game version of the input, such as 3.29.3.3 (dataset)")
	if err := fl.Parse(args); err != nil {
		return err
	}
	cmd := fl.Args()
	if len(cmd) < 2 {
		return errors.New(usage)
	}
	open := func() (*bundle.Archive, error) {
		return bundle.Open(os.DirFS(cmd[1]), ooz.Ooz{Path: *oozPath})
	}
	switch {
	case cmd[0] == "get" && len(cmd) == 3:
		a, err := open()
		if err != nil {
			return err
		}
		b, err := a.ReadFile(cmd[2])
		if err != nil {
			return err
		}
		_, err = os.Stdout.Write(b)
		return err
	case cmd[0] == "table" && len(cmd) == 3:
		a, err := open()
		if err != nil {
			return err
		}
		ed, err := edition.Load(*game)
		if err != nil {
			return err
		}
		return writeTable(a, ed, input.Dir(cmd[1]), cmd[2])
	case cmd[0] == "stats" && len(cmd) == 3:
		a, err := open()
		if err != nil {
			return err
		}
		s, err := ot.Stats(a, cmd[2])
		if err != nil {
			return err
		}
		return json.NewEncoder(os.Stdout).Encode(s)
	case cmd[0] == "dataset" && len(cmd) == 3:
		a, err := open()
		if err != nil {
			return err
		}
		if *version == "" {
			return errors.New("dataset needs -version")
		}
		ed, err := edition.Load(*game)
		if err != nil {
			return err
		}
		return writeDataset(a, ed, *version, input.Dir(cmd[1]), cmd[2])
	case cmd[0] == "art" && len(cmd) == 3:
		a, err := open()
		if err != nil {
			return err
		}
		return writeArt(a, input.Dir(cmd[1]), cmd[2])
	case cmd[0] == "bundles" && len(cmd) == 2:
		a, err := open()
		if err != nil {
			return err
		}
		return writeBundles(a)
	}
	return errors.New(usage)
}

// loadSchema parses schema.json and gives its unnamed columns our names.
func loadSchema(in input.Dir, names []dat.ColumnName) (*dat.Schema, error) {
	sb, err := in.Schema()
	if err != nil {
		return nil, err
	}
	s, err := dat.ParseSchema(sb)
	if err != nil {
		return nil, err
	}
	if err := s.NameColumns(names); err != nil {
		return nil, err
	}
	return s, nil
}

func writeDataset(a *bundle.Archive, ed *edition.Edition, version string, in input.Dir, out string) error {
	schema, err := loadSchema(in, ed.ColumnNames())
	if err != nil {
		return err
	}
	bi, err := editionInput(ed, version)
	if err != nil {
		return err
	}
	bi.Game, bi.Schema = a, schema
	if bi.Trees, err = readTrees(in); err != nil {
		return err
	}
	bi.Warn = func(msg string) { fmt.Fprintf(os.Stderr, "pbt-extract: warning: %s\n", msg) }
	d, err := extract.Dataset(bi)
	if err != nil {
		return err
	}
	d.Manifest = dataset.Manifest{Game: ed.Name, Version: version}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	_, err = dataset.Write(out, d)
	return err
}

// readTrees parses the tree export of each variant. Ruthless uses the hardmode columns of
// PassiveSkills.
func readTrees(in input.Dir) ([]extract.Variant, error) {
	treeFiles := []struct {
		variant, file string
		hardmode      bool
	}{
		{"standard", "data.json", false},
		{"ruthless", "ruthless.json", true},
		{"alternate", "alternate.json", false},
	}
	var trees []extract.Variant
	for _, tf := range treeFiles {
		b, err := in.TreeFile(tf.file)
		if err != nil {
			return nil, err
		}
		e, err := treeexport.Parse(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", tf.file, err)
		}
		trees = append(trees, extract.Variant{Name: tf.variant, Export: e, Hardmode: tf.hardmode})
	}
	return trees, nil
}

// writeArt writes the art into out: the sheets of the standard tree's sprites, which every
// variant's icons must resolve on, and the class and ascendancy faces.
func writeArt(game *bundle.Archive, in input.Dir, out string) error {
	trees, err := readTrees(in)
	if err != nil {
		return err
	}
	exports := make([]*treeexport.Export, len(trees))
	for i, t := range trees {
		exports[i] = t.Export
	}
	sprites := trees[0].Export.Sprites
	for _, kind := range slices.Sorted(maps.Keys(sprites)) {
		for _, z := range slices.Sorted(maps.Keys(sprites[kind])) {
			if bad := sprites[kind][z].Malformed; len(bad) > 0 {
				fmt.Fprintf(os.Stderr, "pbt-extract: warning: %s sprites at %s: left out malformed rectangles of %v\n", kind, z, bad)
			}
		}
	}
	a, err := art.Build(sprites, in.TreeAssets(), exports...)
	if err != nil {
		return err
	}
	schema, err := loadSchema(in, nil)
	if err != nil {
		return err
	}
	faces, err := art.ReadFaces(game, schema, trees[0].Export.Classes, func(msg string) {
		fmt.Fprintf(os.Stderr, "pbt-extract: warning: %s\n", msg)
	})
	if err != nil {
		return err
	}
	if err := a.AddFaces(faces); err != nil {
		return err
	}
	return a.Write(out)
}

// editionInput returns the part of a build's input that edition data gives for game version:
// totems, constants and value maps.
func editionInput(ed *edition.Edition, version string) (extract.Input, error) {
	var bi extract.Input
	for _, t := range ed.Totems {
		bi.EditionTotems = append(bi.EditionTotems, dataset.EditionTotem{GrantedEffect: t.GrantedEffect, Totem: t.Totem, Source: t.Source})
	}
	for _, c := range ed.Constants {
		bi.Constants = append(bi.Constants, dataset.Constant{ID: c.ID, Value: c.Value, Per: c.Per})
	}
	values, err := ed.ValuesFor(version)
	if err != nil {
		return extract.Input{}, err
	}
	for _, v := range values {
		bi.Values = append(bi.Values, extract.ColumnValues{Table: v.Table, Column: v.Column, Map: v.Map})
	}
	return bi, nil
}

// writeBundles lists the bundle files holding the game files a dataset or art build may read.
func writeBundles(a *bundle.Archive) error {
	paths, err := a.Paths()
	if err != nil {
		return err
	}
	var files []string
	for _, p := range paths {
		if !extract.Reads(p) && !art.Reads(p) {
			continue
		}
		f, ok := a.BundleFile(p)
		if !ok {
			return fmt.Errorf("index names %s but has no entry for it", p)
		}
		files = append(files, f)
	}
	slices.Sort(files)
	w := bufio.NewWriter(os.Stdout)
	for _, f := range slices.Compact(files) {
		if _, err := fmt.Fprintln(w, f); err != nil {
			return err
		}
	}
	return w.Flush()
}

// writeTable gives only the table's own unnamed columns our names, and prints dat-schema's
// names with a warning when ours no longer fit: table is how a changed schema gets inspected.
func writeTable(a *bundle.Archive, ed *edition.Edition, in input.Dir, name string) error {
	ours := slices.DeleteFunc(ed.ColumnNames(), func(c dat.ColumnName) bool { return c.Table != name })
	schema, err := loadSchema(in, ours)
	if err != nil && len(ours) > 0 {
		fmt.Fprintf(os.Stderr, "pbt-extract: warning: %v; printing dat-schema's names\n", err)
		schema, err = loadSchema(in, nil)
	}
	if err != nil {
		return err
	}
	def, ok := schema.Table(name)
	if !ok {
		return fmt.Errorf("table %s not in schema.json", name)
	}
	b, err := a.ReadFile("Data/" + name + ".datc64")
	if err != nil {
		return err
	}
	t, err := dat.Read(def, b)
	if err != nil {
		return err
	}

	names := make([]string, t.Columns())
	for i, c := range def.Columns[:t.Columns()] {
		names[i] = "_" + strconv.Itoa(i)
		if c.Name != nil {
			names[i] = *c.Name
		}
	}
	w := bufio.NewWriter(os.Stdout)
	enc := json.NewEncoder(w)
	for r := range t.Len() {
		row := make(map[string]any, len(names))
		for c, n := range names {
			v, err := t.Value(r, c)
			if err != nil {
				return fmt.Errorf("%s row %d: %w", name, r, err)
			}
			row[n] = v
		}
		if err := enc.Encode(row); err != nil {
			return err
		}
	}
	return w.Flush()
}
