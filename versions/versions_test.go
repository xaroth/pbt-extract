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
			if src.TreeExport == "" {
				t.Errorf("got empty treeExport, want a skilltree-export tag")
			}
			if !commit.MatchString(src.TreeExportCommit) {
				t.Errorf("got treeExportCommit %q, want a full commit id", src.TreeExportCommit)
			}
			sha := regexp.MustCompile(`^[0-9a-f]{64}$`)
			for _, f := range []string{"data.json", "ruthless.json", "alternate.json"} {
				if !sha.MatchString(src.TreeExportFiles[f]) {
					t.Errorf("treeExportFiles[%s]: got %q, want a SHA-256", f, src.TreeExportFiles[f])
				}
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

func TestConstants(t *testing.T) {
	t.Parallel()
	cs, err := versions.Constants("poe1")
	if err != nil {
		t.Fatalf("Constants: %v", err)
	}
	if len(cs) == 0 {
		t.Fatalf("Constants: got none, want the poe1 constants")
	}
	if _, err := versions.Constants("nope"); err == nil {
		t.Errorf("Constants(nope): got nil error, want an error")
	}
}

func TestLookupRejectsFiles(t *testing.T) {
	t.Parallel()
	if _, err := versions.Lookup("poe1", "constants.json"); err == nil {
		t.Errorf("Lookup(constants.json): got nil error, want an error")
	}
}

func TestCheckTreeExport(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		version, tag string
		warn, err    bool
	}{
		{version: "3.29.3.3", tag: "3.29.1", warn: true},
		{version: "3.28.0", tag: "3.28.0"},
		{version: "3.28.0.16", tag: "3.28.0.16"},
		{version: "3.28.0", tag: "3.28.0.1"},
		{version: "3.29.0.1", tag: "3.28.0", err: true},
		{version: "4.29.0", tag: "3.29.0", err: true},
		{version: "3.29", tag: "x", err: true},
		{version: "3..1", tag: "3.29", err: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.version+" "+testCase.tag, func(t *testing.T) {
			t.Parallel()
			warn, err := versions.CheckTreeExport(testCase.version, testCase.tag)
			if (err != nil) != testCase.err {
				t.Fatalf("got error %v, want error %v", err, testCase.err)
			}
			if (warn != "") != testCase.warn {
				t.Errorf("got warning %q, want warning %v", warn, testCase.warn)
			}
		})
	}
}
