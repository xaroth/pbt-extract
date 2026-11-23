package extract

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/xaroth/pbt-engine/statdesc"
)

func TestNameHandlerValues(t *testing.T) {
	t.Parallel()
	other := statdesc.Variant{Limits: []statdesc.Limit{{Min: 0, Max: 0}}, Text: "kept"}
	sc := statdesc.Scope{Descriptions: []statdesc.Description{{
		Stats: []string{"ability"},
		Variants: []statdesc.Variant{other, {
			Limits:   []statdesc.Limit{{Min: 1, Max: math.MaxInt32}},
			Text:     "Grants {0}",
			Handlers: []statdesc.Handler{{Name: "ability"}},
		}},
	}}}
	if err := nameHandlerValues(&sc, "ability", []string{"Ire", "", "Despair"}); err != nil {
		t.Fatal(err)
	}
	want := []statdesc.Variant{
		other,
		{Limits: []statdesc.Limit{{Min: 1, Max: 1}}, Text: "Grants Ire"},
		{Limits: []statdesc.Limit{{Min: 3, Max: 3}}, Text: "Grants Despair"},
	}
	if diff := cmp.Diff(want, sc.Descriptions[0].Variants); diff != "" {
		t.Errorf("variants mismatch (-want +got):\n%s", diff)
	}
}
