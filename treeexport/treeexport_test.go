package treeexport_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/xaroth/pbt-extract/treeexport"
)

const sample = `{
	"tree": "Default",
	"classes": [{"name": "Scion", "base_str": 20, "base_dex": 20, "base_int": 20,
		"ascendancies": [{"id": "Ascendant", "name": "Ascendant", "flavourText": "x"}]}],
	"alternate_ascendancies": [{"id": "Aul", "name": "Aul"}],
	"nodes": {
		"root": {"out": ["58833"]},
		"58833": {"skill": 58833, "name": "SEVEN", "classStartIndex": 0, "out": ["4"], "in": [],
			"group": 1, "orbit": 0, "orbitIndex": 0, "icon": "Art/start.png"},
		"4": {"skill": 4, "name": "Mastery", "isMastery": true, "stats": [],
			"masteryEffects": [{"effect": 7, "stats": ["+1 to Strength"]}], "in": ["58833"],
			"icon": "Art/mastery.png", "inactiveIcon": "Art/inactive.png", "activeIcon": "Art/active.png",
			"activeEffectImage": "Art/effect.png", "group": 2, "orbit": 4, "orbitIndex": 39}
	},
	"groups": {
		"1": {"x": 3.435, "y": 6.815, "orbits": [0], "nodes": ["58833"]},
		"2": {"x": -10156.7, "y": -75.185, "isProxy": true, "orbits": [4],
			"background": {"image": "PSGroupBackground1", "isHalfImage": true}, "nodes": ["4"]}
	},
	"constants": {"PSSCentreInnerRadius": 130, "skillsPerOrbit": [1, 6, 16, 16, 40], "orbitRadii": [0, 82, 162, 335, 493]},
	"extraImages": {"1": {"x": -4560.69, "y": 292.425, "image": "Art/2DArt/BaseClassIllustrations/Str.png"}},
	"min_x": -14159, "min_y": -10689, "max_x": 12430.5, "max_y": 10900,
	"jewelSlots": [4],
	"points": {"totalPoints": 123, "ascendancyPoints": 8},
	"sprites": {"ignored": true}
}`

func TestParse(t *testing.T) {
	t.Parallel()
	got, err := treeexport.Parse([]byte(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	start := 0
	want := &treeexport.Export{
		Tree: "Default",
		Classes: []treeexport.Class{{
			Name: "Scion", BaseStrength: 20, BaseDexterity: 20, BaseIntelligence: 20,
			Ascendancies: []treeexport.Ascendancy{{ID: "Ascendant", Name: "Ascendant"}},
		}},
		AlternateAscendancies: []treeexport.Ascendancy{{ID: "Aul", Name: "Aul"}},
		Nodes: map[string]treeexport.Node{
			"58833": {Skill: 58833, Name: "SEVEN", ClassStartIndex: &start, Out: []string{"4"}, In: []string{}, Group: 1, Icon: "Art/start.png"},
			"4": {
				Skill: 4, Name: "Mastery", IsMastery: true, Stats: []string{},
				MasteryEffects: []treeexport.MasteryEffect{{Effect: 7, Stats: []string{"+1 to Strength"}}},
				In:             []string{"58833"},
				Icon:           "Art/mastery.png", InactiveIcon: "Art/inactive.png", ActiveIcon: "Art/active.png", ActiveEffectImage: "Art/effect.png",
				Group: 2, Orbit: 4, OrbitIndex: 39,
			},
		},
		JewelSlots: []uint32{4},
		Groups: map[string]treeexport.Group{
			"1": {X: 3.435, Y: 6.815, Orbits: []uint8{0}},
			"2": {X: -10156.7, Y: -75.185, IsProxy: true, Orbits: []uint8{4}, Background: &treeexport.Background{Image: "PSGroupBackground1", IsHalfImage: true}},
		},
		ExtraImages: map[string]treeexport.ExtraImage{"1": {X: -4560.69, Y: 292.425, Image: "Art/2DArt/BaseClassIllustrations/Str.png"}},
		MinX:        -14159, MinY: -10689, MaxX: 12430.5, MaxY: 10900,
	}
	want.Points.Total, want.Points.Ascendancy = 123, 8
	want.Constants.SkillsPerOrbit = []uint32{1, 6, 16, 16, 40}
	want.Constants.OrbitRadii = []float64{0, 82, 162, 335, 493}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Parse mismatch (-want +got):\n%s", diff)
	}
}

func TestParseErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		src  string
	}{
		{name: "not json", src: `{`},
		{name: "no nodes", src: `{"nodes": {"root": {}}}`},
		{name: "skill differs from key", src: `{"nodes": {"5": {"skill": 6}}}`},
		{name: "bad node", src: `{"nodes": {"5": {"skill": "x"}}}`},
		{name: "orbit out of range", src: `{"nodes": {"5": {"skill": 5, "orbit": 256}}}`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := treeexport.Parse([]byte(testCase.src)); err == nil {
				t.Errorf("Parse: got nil error, want an error")
			}
		})
	}
}
