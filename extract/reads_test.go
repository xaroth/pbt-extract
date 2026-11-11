package extract_test

import (
	"testing"

	"github.com/xaroth/pbt-extract/extract"
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
		if got := extract.Reads(path); got != want {
			t.Errorf("Reads(%q) = %v, want %v", path, got, want)
		}
	}
	for _, path := range append([]string{extract.DescriptionsRoot}, extract.SkillDescriptions()...) {
		if !extract.Reads(path) {
			t.Errorf("Reads(%q) = false, want true", path)
		}
	}
}
