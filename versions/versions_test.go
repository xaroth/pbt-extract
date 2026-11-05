package versions_test

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/xaroth/pbt-extract/versions"
)

func TestAllVersionsAreComplete(t *testing.T) {
	t.Parallel()
	all, err := versions.All("poe1")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 {
		t.Fatal("got no poe1 versions")
	}
	commit := regexp.MustCompile(`^[0-9a-f]{40}$`)
	manifest := regexp.MustCompile(`^[0-9]+$`)
	for _, v := range all {
		t.Run(v.Name, func(t *testing.T) {
			t.Parallel()
			src, err := v.Source()
			if err != nil {
				t.Fatal(err)
			}
			if !commit.MatchString(src.DatSchemaCommit) {
				t.Errorf("got datSchemaCommit %q, want a full commit id", src.DatSchemaCommit)
			}
			if !manifest.MatchString(src.SteamManifest) {
				t.Errorf("got steamManifest %q, want a manifest id", src.SteamManifest)
			}
			b, err := v.Schema()
			if err != nil {
				t.Fatal(err)
			}
			var s struct {
				Tables []struct {
					ValidFor int `json:"validFor"`
				} `json:"tables"`
			}
			if err := json.Unmarshal(b, &s); err != nil {
				t.Fatal(err)
			}
			for _, tbl := range s.Tables {
				if tbl.ValidFor&1 == 0 {
					t.Fatalf("got a table with validFor %d in a poe1 schema", tbl.ValidFor)
				}
			}
		})
	}
}

func TestLookup(t *testing.T) {
	t.Parallel()
	if _, err := versions.Lookup("poe1", "3.29.3.3"); err != nil {
		t.Errorf("got %v, want nil", err)
	}
	if _, err := versions.Lookup("poe1", "0.0.0"); err == nil {
		t.Error("got nil error for an unknown version, want an error")
	}
}
