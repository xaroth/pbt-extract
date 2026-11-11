package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"

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
