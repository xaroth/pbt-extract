package extract_test

import (
	"math"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/extract"
	"github.com/xaroth/pbt-extract/treeexport"
)

func TestLayouts(t *testing.T) {
	t.Parallel()
	got, err := extract.Dataset(input(t, game(t), export()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	if _, err := dataset.Write(t.TempDir(), got); err != nil {
		t.Fatalf("Write: %v", err)
	}
	want := dataset.Layout{
		Variant: "standard",
		MinX:    -14159, MinY: -10689, MaxX: 12430, MaxY: 10900,
		Orbits: []dataset.Orbit{
			{Radius: 0, Slots: 1}, {Radius: 82, Slots: 6}, {Radius: 162, Slots: 16},
			{Radius: 335, Slots: 16}, {Radius: 493, Slots: 40}, {Radius: 662, Slots: 72},
		},
		// sorted by id, not by key
		Groups: []dataset.LayoutGroup{
			{ID: 2, X: -2824.29, Y: 1642.43, Orbits: []uint8{}},
			{ID: 10, X: 100, Y: -50, Orbits: []uint8{0, 2}, Background: &dataset.GroupBackground{Image: "PSGroupBackground3", HalfImage: true}, Proxy: true},
		},
		Nodes: []dataset.LayoutNode{
			{ID: 1, Group: 2, Icon: "Art/start.png"},
			{ID: 2, Group: 10, Orbit: 2, OrbitIndex: 2, Icon: "Art/might.png"},
			{
				ID: 3, Icon: "Art/mastery.png",
				MasteryInactiveIcon: "Art/inactive.png", MasteryActiveIcon: "Art/active.png", MasteryActiveEffect: "Art/effect.png",
			},
		},
		ClassArt: []dataset.ClassArt{{Class: 0, Image: "Art/2DArt/BaseClassIllustrations/Str.png", X: -4560.69, Y: 292.425}},
	}
	if len(got.Layouts) != 2 {
		t.Fatalf("got %d layouts, want 2", len(got.Layouts))
	}
	if diff := cmp.Diff(want, got.Layouts[0], cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("standard layout mismatch (-want +got):\n%s", diff)
	}
	if got.Layouts[1].Variant != "ruthless" {
		t.Errorf("second layout: got variant %s, want ruthless", got.Layouts[1].Variant)
	}
	// orbit 2 index 2 is 45 degrees clockwise from up
	x, y, ok := got.Layouts[0].Position(2)
	if wx, wy := 100+162/math.Sqrt2, -50-162/math.Sqrt2; !ok || math.Abs(x-wx) > 1e-9 || math.Abs(y-wy) > 1e-9 {
		t.Errorf("Position(2): got (%v, %v, %v), want (%v, %v, true)", x, y, ok, wx, wy)
	}
}

func TestLayoutsErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name, want string
		export     func(*treeexport.Export)
	}{
		{name: "radii and counts differ", want: "6 orbit node counts but 5 orbit radii", export: func(e *treeexport.Export) {
			e.Constants.OrbitRadii = e.Constants.OrbitRadii[:5]
		}},
		{name: "empty orbit", want: "orbit 1 has 0 nodes", export: func(e *treeexport.Export) { e.Constants.SkillsPerOrbit[1] = 0 }},
		{name: "group key not a number", want: `group key "x"`, export: func(e *treeexport.Export) { e.Groups["x"] = treeexport.Group{} }},
		{name: "background offset", want: "background offset (5, 0) not supported", export: func(e *treeexport.Export) {
			e.Groups["10"].Background.OffsetX = 5
		}},
		{name: "extra image key not a number", want: `extra image key "x"`, export: func(e *treeexport.Export) {
			e.ExtraImages["x"] = treeexport.ExtraImage{}
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			e := export()
			testCase.export(e)
			_, err := extract.Dataset(input(t, game(t), e))
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Errorf("Dataset: got %v, want an error containing %q", err, testCase.want)
			}
		})
	}
}
