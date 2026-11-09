// Command pbt-extract builds pbt datasets from Path of Exile game files.
//
//	pbt-extract [-ooz path] [-game poe1] dataset <in> <out>
//	pbt-extract [-ooz path] bundles <in>
//	pbt-extract [-ooz path] get <in> <path>
//	pbt-extract [-ooz path] table <in> <table>
//	pbt-extract [-ooz path] stats <in> <object>
//
// in is an input directory (see package input): source.json, schema.json, the tree export
// under tree/ and the game's Bundles2. dataset writes the dataset into the directory out, with
// source.json as the manifest's source.
// bundles lists the bundle files under in that a dataset build may read, one per line, from
// Bundles2/_.index.bin alone. get writes the game file at path (for example Data/Stats.datc64)
// to stdout. table decodes Data/<table>.datc64 with schema.json and writes one JSON object per
// row. stats writes the Stats block of an object template such as Metadata/Characters/Str/Str,
// merged over its extends chain, as one JSON object.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"
	"strconv"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/build"
	"github.com/xaroth/pbt-extract/bundle"
	"github.com/xaroth/pbt-extract/constants"
	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/input"
	"github.com/xaroth/pbt-extract/ooz"
	"github.com/xaroth/pbt-extract/ot"
	"github.com/xaroth/pbt-extract/tree"
)

const usage = "usage: pbt-extract [-ooz path] dataset <in> <out>\n       pbt-extract [-ooz path] bundles <in>\n       pbt-extract [-ooz path] get <in> <path>\n       pbt-extract [-ooz path] table <in> <table>\n       pbt-extract [-ooz path] stats <in> <object>"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "pbt-extract:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fl := flag.NewFlagSet("pbt-extract", flag.ContinueOnError)
	oozPath := fl.String("ooz", "", "path to the ooz binary (default: ooz from PATH)")
	game := fl.String("game", "poe1", "game edition of the input")
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
		return writeTable(a, cmd[1], cmd[2])
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
		return writeDataset(a, *game, cmd[1], cmd[2])
	case cmd[0] == "bundles" && len(cmd) == 2:
		a, err := open()
		if err != nil {
			return err
		}
		return writeBundles(a)
	}
	return errors.New(usage)
}

func loadSchema(in *input.Dir) (*dat.Schema, error) {
	sb, err := in.Schema()
	if err != nil {
		return nil, err
	}
	return dat.ParseSchema(sb)
}

func writeDataset(a *bundle.Archive, game, inDir, out string) error {
	in, err := input.Open(inDir)
	if err != nil {
		return err
	}
	schema, err := loadSchema(in)
	if err != nil {
		return err
	}
	bi := build.Input{Game: a, Schema: schema}
	// Ruthless uses the hardmode columns of PassiveSkills.
	treeFiles := []struct {
		variant, file string
		hardmode      bool
	}{
		{"standard", "data.json", false},
		{"ruthless", "ruthless.json", true},
		{"alternate", "alternate.json", false},
	}
	for _, tf := range treeFiles {
		b, err := in.TreeFile(tf.file)
		if err != nil {
			return err
		}
		e, err := tree.Parse(b)
		if err != nil {
			return fmt.Errorf("%s: %w", tf.file, err)
		}
		bi.Trees = append(bi.Trees, build.Variant{Name: tf.variant, Export: e, Hardmode: tf.hardmode})
	}
	consts, err := constants.Load(game)
	if err != nil {
		return err
	}
	for _, c := range consts {
		bi.Constants = append(bi.Constants, dataset.Constant{ID: c.ID, Value: c.Value, Per: c.Per})
	}
	d, err := build.Dataset(bi)
	if err != nil {
		return err
	}
	d.Manifest = dataset.Manifest{Game: game, Version: in.Version(), Source: in.Source}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	_, err = dataset.Write(out, d)
	return err
}

// writeBundles lists the bundle files holding the game files a build may read.
func writeBundles(a *bundle.Archive) error {
	paths, err := a.Paths()
	if err != nil {
		return err
	}
	var files []string
	for _, p := range paths {
		if !build.Reads(p) {
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

func writeTable(a *bundle.Archive, inDir, name string) error {
	in, err := input.Open(inDir)
	if err != nil {
		return err
	}
	schema, err := loadSchema(in)
	if err != nil {
		return err
	}
	def, ok := schema.Table(name)
	if !ok {
		return fmt.Errorf("table %s not in the %s schema", name, in.Version())
	}
	b, err := a.ReadFile("Data/" + name + ".datc64")
	if err != nil {
		return err
	}
	t, err := dat.Read(def, b)
	if err != nil {
		return err
	}

	names := make([]string, len(def.Columns))
	for i, c := range def.Columns {
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
