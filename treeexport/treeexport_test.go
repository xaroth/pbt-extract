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
		"58833": {"skill": 58833, "name": "SEVEN", "classStartIndex": 0, "out": ["4"], "in": []},
		"4": {"skill": 4, "name": "Mastery", "isMastery": true, "stats": [],
			"masteryEffects": [{"effect": 7, "stats": ["+1 to Strength"]}], "in": ["58833"]}
	},
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
			"58833": {Skill: 58833, Name: "SEVEN", ClassStartIndex: &start, Out: []string{"4"}, In: []string{}},
			"4": {
				Skill: 4, Name: "Mastery", IsMastery: true, Stats: []string{},
				MasteryEffects: []treeexport.MasteryEffect{{Effect: 7, Stats: []string{"+1 to Strength"}}},
				In:             []string{"58833"},
			},
		},
		JewelSlots: []uint32{4},
	}
	want.Points.Total, want.Points.Ascendancy = 123, 8
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
