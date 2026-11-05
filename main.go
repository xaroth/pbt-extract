// Command pbt-extract reads game files from a Path of Exile install or bundle archive.
//
//	pbt-extract [-ooz path] get <dir> <path>
//	pbt-extract [-ooz path] table <dir> <version> <table>
//
// get writes the file at path (for example Data/Stats.datc64) to stdout. table decodes
// Data/<table>.datc64 with the dat-schema build stored for version and writes one JSON object
// per row. dir is the game install or archive that holds Bundles2.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/xaroth/pbt-extract/bundle"
	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ooz"
	"github.com/xaroth/pbt-extract/versions"
)

const usage = "usage: pbt-extract [-ooz path] get <dir> <path>\n       pbt-extract [-ooz path] table <dir> <version> <table>"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "pbt-extract:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fl := flag.NewFlagSet("pbt-extract", flag.ContinueOnError)
	oozPath := fl.String("ooz", "", "path to the ooz binary (default: ooz from PATH)")
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
	case cmd[0] == "table" && len(cmd) == 4:
		a, err := open()
		if err != nil {
			return err
		}
		return writeTable(a, cmd[2], cmd[3])
	}
	return errors.New(usage)
}

func writeTable(a *bundle.Archive, version, name string) error {
	v, err := versions.Lookup("poe1", version)
	if err != nil {
		return fmt.Errorf("version %s: %w", version, err)
	}
	sb, err := v.Schema()
	if err != nil {
		return err
	}
	schema, err := dat.ParseSchema(sb)
	if err != nil {
		return err
	}
	def, ok := schema.Table(name)
	if !ok {
		return fmt.Errorf("table %s not in the %s schema", name, version)
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
