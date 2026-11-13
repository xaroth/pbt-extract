package main

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/edition"
	"github.com/xaroth/pbt-extract/extract"
)

func TestEditionInput(t *testing.T) {
	t.Parallel()
	ed, err := edition.Load("poe1")
	if err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		version string
		want    []extract.ColumnValues
	}{
		{version: "3.27.0.10", want: []extract.ColumnValues{{Table: "GemEffects", Column: "ItemColor", Map: map[uint32]uint32{1: 1, 2: 2, 3: 3, 4: 5}}}},
		{version: "3.28.0.16"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.version, func(t *testing.T) {
			t.Parallel()
			got, err := editionInput(ed, testCase.version)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(testCase.want, got.Values); diff != "" {
				t.Errorf("Values mismatch (-want +got):\n%s", diff)
			}
			if len(got.EditionTotems) != len(ed.Totems) || len(got.Constants) != len(ed.Constants) {
				t.Errorf("got %d totems and %d constants, want %d and %d", len(got.EditionTotems), len(got.Constants), len(ed.Totems), len(ed.Constants))
			}
		})
	}
	if _, err := editionInput(ed, "3.x"); err == nil {
		t.Error("editionInput(3.x): got nil error, want an error")
	}
}

// TestRealTimelessNames builds the dataset of the 3.29 input directory $PBT_INPUT_3_29 and checks
// that every timeless replacement passive of every tree has its English name. It skips without
// $PBT_INPUT_3_29; ooz is $PBT_OOZ or ooz from PATH.
func TestRealTimelessNames(t *testing.T) {
	t.Parallel()
	in := os.Getenv("PBT_INPUT_3_29")
	if in == "" {
		t.Skip("PBT_INPUT_3_29 not set")
	}
	out := t.TempDir()
	if err := run([]string{"-ooz", os.Getenv("PBT_OOZ"), "-version", "3.29.3.3", "dataset", in, out}); err != nil {
		t.Fatalf("dataset: %v", err)
	}
	s, err := dataset.Open(os.DirFS(out))
	if err != nil {
		t.Fatal(err)
	}
	names, err := s.Names("en")
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]string{}
	for _, n := range names.TimelessNames {
		byID[n.ID] = n.Name
	}
	const want = 182
	if len(names.TimelessNames) != want {
		t.Errorf("got %d timeless names, want %d", len(names.TimelessNames), want)
	}
	tree, err := s.Tree("standard")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range tree.TimelessVersions {
		for _, r := range v.Replacements {
			if byID[r.ID] == "" {
				t.Errorf("version %s replacement %s has no name", v.ID, r.ID)
			}
		}
	}
	for id, name := range map[string]string{
		"vaal_keystone_1":    "Divine Flesh",
		"vaal_keystone_2_v2": "Immortal Ambition",
		"kalguur_keystone_3": "The Unbreaking Circle",
	} {
		if byID[id] != name {
			t.Errorf("name of %s: got %q, want %q", id, byID[id], name)
		}
	}
}
