package edition_test

import (
	"testing"

	"github.com/xaroth/pbt-extract/edition"
)

func TestLoad(t *testing.T) {
	t.Parallel()
	e, err := edition.Load("poe1")
	if err != nil {
		t.Fatal(err)
	}
	if e.Name != "poe1" {
		t.Errorf("got name %q, want poe1", e.Name)
	}
	if len(e.Constants) == 0 {
		t.Error("got no constants, want the poe1 constants")
	}
	if _, err := edition.Load("nope"); err == nil {
		t.Error("Load(nope): got nil error, want an error")
	}
}
