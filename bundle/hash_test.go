package bundle_test

import (
	"testing"

	"github.com/xaroth/pbt-extract/bundle"
)

func TestPathHash(t *testing.T) {
	t.Parallel()
	// Expected values match the hashes in the 3.29.3.3 index.
	testCases := []struct {
		path string
		want uint64
	}{
		{"", 0xf42a94e69cff42fe},
		{"a", 0xf8d232a19e90f23c},
		{"Data/Stats.datc64", 0x50915701cc2396d8},
		{"data/stats.datc64", 0x50915701cc2396d8},
		{"Metadata/Characters/Character.ot", 0x1308246a6abc47da},
	}
	for _, testCase := range testCases {
		t.Run(testCase.path, func(t *testing.T) {
			t.Parallel()
			if got := bundle.PathHash(testCase.path); got != testCase.want {
				t.Errorf("got %#x, want %#x", got, testCase.want)
			}
		})
	}
}
