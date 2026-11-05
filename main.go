// Command pbt-extract reads game files from a Path of Exile install or bundle archive.
//
//	pbt-extract [-ooz path] get <dir> <path>
//
// get writes the file at path (for example Data/Stats.datc64) from the Bundles2 archive under
// dir to stdout.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/xaroth/pbt-extract/bundle"
	"github.com/xaroth/pbt-extract/ooz"
)

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
	if fl.NArg() != 3 || fl.Arg(0) != "get" {
		return fmt.Errorf("usage: pbt-extract [-ooz path] get <dir> <path>")
	}
	a, err := bundle.Open(os.DirFS(fl.Arg(1)), ooz.Ooz{Path: *oozPath})
	if err != nil {
		return err
	}
	b, err := a.ReadFile(fl.Arg(2))
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(b)
	return err
}
