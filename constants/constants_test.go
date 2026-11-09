package constants_test

import (
	"testing"

	"github.com/xaroth/pbt-extract/constants"
)

func TestLoad(t *testing.T) {
	t.Parallel()
	cs, err := constants.Load("poe1")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs) == 0 {
		t.Fatal("got no constants, want the poe1 constants")
	}
	if _, err := constants.Load("nope"); err == nil {
		t.Error("Load(nope): got nil error, want an error")
	}
}
