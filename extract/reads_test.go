package extract_test

import (
	"testing"
	"testing/fstest"

	"github.com/xaroth/pbt-extract/extract"
)

func TestReads(t *testing.T) {
	t.Parallel()
	for path, want := range map[string]bool{
		"Data/Stats.datc64":           true,
		"data/passiveskills.datc64":   true,
		"Data/BaseItemTypes.datc64":   true,
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

// recorder records the files read from fs.
type recorder struct {
	fs   fstest.MapFS
	read []string
}

func (r *recorder) ReadFile(name string) ([]byte, error) {
	r.read = append(r.read, name)
	return r.fs.ReadFile(name)
}

// TestReadsCoversDataset checks that every file a build reads is one Reads declares.
func TestReadsCoversDataset(t *testing.T) {
	t.Parallel()
	r := &recorder{fs: game(t)}
	in := input(t, r.fs, export())
	in.Game = r
	if _, err := extract.Dataset(in); err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	for _, p := range r.read {
		if !extract.Reads(p) {
			t.Errorf("Dataset read %s, which Reads does not declare", p)
		}
	}
}
