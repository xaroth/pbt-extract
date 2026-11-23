package main

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/xaroth/pbt-engine/dataset"
	"github.com/xaroth/pbt-engine/statdesc"

	"github.com/xaroth/pbt-extract/edition"
	"github.com/xaroth/pbt-extract/extract"
)

func TestEditionInput(t *testing.T) {
	t.Parallel()
	ed, err := edition.Load("poe1")
	if err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		version string
		want    []extract.ColumnValues
	}{
		{version: "3.27.0.10", want: []extract.ColumnValues{{Table: "GemEffects", Column: "ItemColor", Map: map[uint32]uint32{1: 1, 2: 2, 3: 3, 4: 5}}}},
		{version: "3.28.0.16"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.version, func(t *testing.T) {
			t.Parallel()
			got, err := editionInput(ed, testCase.version)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(testCase.want, got.Values); diff != "" {
				t.Errorf("Values mismatch (-want +got):\n%s", diff)
			}
			if len(got.EditionTotems) != len(ed.Totems) || len(got.Constants) != len(ed.Constants) {
				t.Errorf("got %d totems and %d constants, want %d and %d", len(got.EditionTotems), len(got.Constants), len(ed.Totems), len(ed.Constants))
			}
		})
	}
	if _, err := editionInput(ed, "3.x"); err == nil {
		t.Error("editionInput(3.x): got nil error, want an error")
	}
}

// realDataset builds the dataset of version from the input directory $PBT_INPUT_<league>, such as
// $PBT_INPUT_3_29; ooz is $PBT_OOZ or ooz from PATH. It skips without the input directory.
func realDataset(t *testing.T, version string) *dataset.Store {
	t.Helper()
	league := strings.Join(strings.Split(version, ".")[:2], "_")
	in := os.Getenv("PBT_INPUT_" + league)
	if in == "" {
		t.Skip("PBT_INPUT_" + league + " not set")
	}
	out := t.TempDir()
	if err := run([]string{"-ooz", os.Getenv("PBT_OOZ"), "-version", version, "dataset", in, out}); err != nil {
		t.Fatalf("dataset: %v", err)
	}
	s, err := dataset.Open(os.DirFS(out))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TestRealTimelessNames checks that every timeless replacement passive of every tree of the real
// 3.29 dataset has its English name.
func TestRealTimelessNames(t *testing.T) {
	t.Parallel()
	s := realDataset(t, "3.29.3.3")
	names, err := s.Names("en")
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]string{}
	for _, n := range names.TimelessNames {
		byID[n.ID] = n.Name
	}
	const want = 182
	if len(names.TimelessNames) != want {
		t.Errorf("got %d timeless names, want %d", len(names.TimelessNames), want)
	}
	tree, err := s.Tree("standard")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range tree.TimelessVersions {
		for _, r := range v.Replacements {
			if byID[r.ID] == "" {
				t.Errorf("version %s replacement %s has no name", v.ID, r.ID)
			}
		}
	}
	for id, name := range map[string]string{
		"vaal_keystone_1":    "Divine Flesh",
		"vaal_keystone_2_v2": "Immortal Ambition",
		"kalguur_keystone_3": "The Unbreaking Circle",
	} {
		if byID[id] != name {
			t.Errorf("name of %s: got %q, want %q", id, byID[id], name)
		}
	}
}

// TestRealRuthlessGrants checks the mods of the real 3.29 dataset that grant a skill of their own
// in Ruthless: 15, of which 14 grant another skill than outside Ruthless, such as the HardMode
// Grace of GraceSkillCorrupted.
func TestRealRuthlessGrants(t *testing.T) {
	t.Parallel()
	mods, err := realDataset(t, "3.29.3.3").Mods()
	if err != nil {
		t.Fatal(err)
	}
	filled, differ := 0, 0
	for _, m := range mods {
		if len(m.RuthlessGrantedSkills) == 0 {
			continue
		}
		filled++
		if !slices.Equal(m.GrantedSkills, m.RuthlessGrantedSkills) {
			differ++
		}
		if m.ID == "GraceSkillCorrupted" {
			if diff := cmp.Diff([]dataset.GrantedSkill{{Effect: "GraceHardMode", Level: 16}}, m.Grants(true)); diff != "" {
				t.Errorf("GraceSkillCorrupted in Ruthless (-want +got):\n%s", diff)
			}
		}
	}
	if filled != 15 || differ != 14 {
		t.Errorf("got %d mods with Ruthless grants, %d differing; want 15 and 14", filled, differ)
	}
}

// TestRealTimelessVersionNames checks that the real 3.30 dataset names the faction and unique
// jewel of each version of the six unique timeless jewels, and the conqueror of each of their
// keystones.
func TestRealTimelessVersionNames(t *testing.T) {
	t.Parallel()
	s := realDataset(t, "3.30.0.1")
	names, err := s.Names("en")
	if err != nil {
		t.Fatal(err)
	}
	factions, jewels, conquerors := map[string]string{}, map[string]string{}, map[string]string{}
	for _, n := range names.TimelessVersionNames {
		factions[n.ID] = n.Name
	}
	for _, n := range names.TimelessJewelNames {
		jewels[n.ID] = n.Name
	}
	for _, n := range names.ConquerorNames {
		conquerors[n.ID] = n.Name
	}
	wantFactions := map[string]string{
		"Vaal": "Vaal", "Karui": "Karui", "Maraketh": "Maraketh", "Templar": "Templars", "Eternal": "Eternal Empire", "Kalguuran": "Kalguur",
	}
	if diff := cmp.Diff(wantFactions, factions); diff != "" {
		t.Errorf("faction names mismatch (-want +got):\n%s", diff)
	}
	wantJewels := map[string]string{
		"Vaal": "Glorious Vanity", "Karui": "Lethal Pride", "Maraketh": "Brutal Restraint", "Templar": "Militant Faith",
		"Eternal": "Elegant Hubris", "Kalguuran": "Heroic Tragedy",
	}
	if diff := cmp.Diff(wantJewels, jewels); diff != "" {
		t.Errorf("jewel names mismatch (-want +got):\n%s", diff)
	}
	tree, err := s.Tree("standard")
	if err != nil {
		t.Fatal(err)
	}
	keystones := 0
	for _, v := range tree.TimelessVersions {
		for _, r := range v.Replacements {
			if r.Conqueror == (dataset.TimelessConqueror{}) {
				continue
			}
			keystones++
			if conquerors[r.ID] == "" {
				t.Errorf("version %s keystone %s has no conqueror name", v.ID, r.ID)
			}
		}
	}
	if keystones == 0 {
		t.Error("no conqueror keystones")
	}
	// 3.29's descriptions named vaal_keystone_2_v2's conqueror Ahuana; 3.30's game files name Mazcotl.
	for id, name := range map[string]string{"vaal_keystone_1": "Xibaqua", "vaal_keystone_2_v2": "Mazcotl", "vaal_keystone_3": "Doryani"} {
		if conquerors[id] != name {
			t.Errorf("conqueror of %s: got %q, want %q", id, conquerors[id], name)
		}
	}
}

// TestRealAnimatedGuardianStats checks that the Animated Guardian of the real 3.29 and 3.30
// datasets holds the stats of its own template, AnimatedArmour.ot, which no mod gives it.
func TestRealAnimatedGuardianStats(t *testing.T) {
	t.Parallel()
	want := map[string]int32{
		"main_hand_local_minimum_added_physical_damage": 2,
		"main_hand_local_maximum_added_physical_damage": 6,
		"dual_wield_inherent_attack_speed_+%_final":     10,
		"block_while_dual_wielding_%":                   15,
	}
	for _, version := range []string{"3.29.3.3", "3.30.0.1"} {
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			s := realDataset(t, version)
			stats, err := s.Stats()
			if err != nil {
				t.Fatal(err)
			}
			m, err := s.Monsters()
			if err != nil {
				t.Fatal(err)
			}
			i := slices.IndexFunc(m.Varieties, func(v dataset.MonsterVariety) bool {
				return v.ID == "Metadata/Monsters/AnimatedItem/AnimatedArmour"
			})
			if i < 0 {
				t.Fatal("no Animated Guardian variety")
			}
			got := map[string]int32{}
			for _, sv := range m.Varieties[i].Stats {
				got[stats[sv.Stat].ID] = sv.Value
			}
			for id, v := range want {
				if got[id] != v {
					t.Errorf("stat %s: got %d, want %d", id, got[id], v)
				}
			}
		})
	}
}

// TestRealGrantLines checks that the real 3.30 dataset has the lines of item mods that grant a
// skill or support, which 3.30's game files build from ClientStrings rather than describe, with
// the grant's level as the stat's range.
func TestRealGrantLines(t *testing.T) {
	t.Parallel()
	s := realDataset(t, "3.30.0.1")
	stats, err := s.Stats()
	if err != nil {
		t.Fatal(err)
	}
	mods, err := s.Mods()
	if err != nil {
		t.Fatal(err)
	}
	text, err := s.Text("en")
	if err != nil {
		t.Fatal(err)
	}
	set, err := statdesc.NewSet(text.Descriptions)
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{
		"GrantsEnvyUnique__1":                              "Grants Level 25 Envy",
		"ItemActsAsConcentratedAOESupportUniqueHelmetInt4": "Socketed Gems are Supported by Level 20 Concentrated Effect",
	} {
		i := slices.IndexFunc(mods, func(m dataset.Mod) bool { return m.ID == id })
		if i < 0 {
			t.Errorf("no mod %s", id)
			continue
		}
		m := mods[i]
		if len(m.Stats) != 1 || len(m.GrantedSkills) == 0 {
			t.Errorf("mod %s: got %d stats and grants %v, want one stat and a grant", id, len(m.Stats), m.GrantedSkills)
			continue
		}
		ms, level := m.Stats[0], m.GrantedSkills[0].Level
		if ms.Min != level || ms.Max != level {
			t.Errorf("mod %s: got range [%d, %d], want [%d, %d]", id, ms.Min, ms.Max, level, level)
		}
		lines, err := set.Describe("Metadata/StatDescriptions/stat_descriptions.txt", map[string]int32{stats[ms.Stat].ID: ms.Min})
		if err != nil {
			t.Fatal(err)
		}
		if len(lines) != 1 || lines[0].Text != want {
			t.Errorf("mod %s: got lines %+v, want %q", id, lines, want)
		}
	}
}
