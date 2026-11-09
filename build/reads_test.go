package build_test

import (
	"testing"

	"github.com/xaroth/pbt-extract/build"
)

func TestReads(t *testing.T) {
	t.Parallel()
	for path, want := range map[string]bool{
		"Data/Stats.datc64":           true,
		"data/passiveskills.datc64":   true,
		"Data/French/Stats.datc64":    false,
		"Metadata/Characters/Str.ot":  true,
		"Metadata/Items/Item.it":      true,
		"Metadata/StatDescriptions/x": true,
		"Art/Textures/thing.dds":      false,
		"Data/Stats.dat":              false,
	} {
		if got := build.Reads(path); got != want {
			t.Errorf("Reads(%q) = %v, want %v", path, got, want)
		}
	}
}
