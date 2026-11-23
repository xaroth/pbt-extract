package extract

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/xaroth/pbt-engine/dataset"
	"github.com/xaroth/pbt-engine/statdesc"
)

func TestRebuildGrantLines(t *testing.T) {
	t.Parallel()
	stats := []dataset.Stat{{ID: "grants_envy"}, {ID: "supported_by_pierce"}, {ID: "described"}, {ID: "shared"}, {ID: "harbinger"}, {ID: "cast_on_crit"}, {ID: "grants_icestorm"}, {ID: "sound_flag"}}
	names := map[string]grantName{
		"Envy":                        {name: "Envy"},
		"SupportPierce":               {name: "Pierce", support: true},
		"SupportCastOnCriticalStrike": {name: "Cast On Critical Strike", support: true},
		"HarbingerOfTime":             {name: "Summon Harbinger of Time"},
		"HarbingerOfFocus":            {name: "Summon Harbinger of Focus"},
		"Icestorm":                    {name: "Icestorm"},
	}
	grant := func(effect string, level int32) []dataset.GrantedSkill {
		return []dataset.GrantedSkill{{Effect: effect, Level: level}}
	}
	stat := func(h uint32, v int32) []dataset.ModStat {
		return []dataset.ModStat{{Stat: h, Min: v, Max: v, RuthlessMin: v, RuthlessMax: v}}
	}
	mods := []dataset.Mod{
		{ID: "Envy25", Stats: stat(0, 0), GrantedSkills: grant("Envy", 25)},
		// already at its level, as in a few 3.30 mods
		{ID: "Envy15", Stats: stat(0, 15), GrantedSkills: grant("Envy", 15)},
		{ID: "Pierce", Stats: stat(1, 0), GrantedSkills: grant("SupportPierce", 15)},
		// a mod without a grant holds the stat at 0 and shows no line
		{ID: "NoGrantPierce", Stats: stat(1, 0)},
		{ID: "Described", Stats: stat(2, 0), GrantedSkills: grant("Envy", 1)},
		// a mod without a grant that rolls the stat: not a grant stat
		{ID: "SharedGrant", Stats: stat(3, 0), GrantedSkills: grant("Envy", 1)},
		{ID: "SharedPlain", Stats: stat(3, 30)},
		{ID: "Time", Stats: stat(4, 0), GrantedSkills: grant("HarbingerOfTime", 1)},
		{ID: "Focus", Stats: stat(4, 0), GrantedSkills: grant("HarbingerOfFocus", 1)},
		// the triggered half has no name
		{ID: "CastOnCrit", Stats: stat(5, 0), GrantedSkills: []dataset.GrantedSkill{
			{Effect: "SupportCastOnCriticalStrike", Level: 20}, {Effect: "SupportCastOnCritTriggered", Level: 20},
		}},
		// an undescribed flag at the grant's level beside the grant stat, as on The Whispering Ice
		{ID: "Icestorm", Stats: append(stat(6, 0), stat(7, 1)...), GrantedSkills: grant("Icestorm", 1)},
	}
	described := statdesc.Description{Stats: []string{"described"}, Variants: []statdesc.Variant{{Text: "kept"}}}
	item := statdesc.Scope{Name: itemScope, Descriptions: []statdesc.Description{described}}
	var warnings []string
	rebuildGrantLines(mods, stats, &item, names, grantTemplates{
		skill:   "Grants Level {1} {0}",
		support: "Socketed Gems are Supported by Level {1} {0}",
	}, func(msg string) { warnings = append(warnings, msg) })

	line := func(id, text string) statdesc.Description {
		return statdesc.Description{Stats: []string{id}, Variants: []statdesc.Variant{{Limits: []statdesc.Limit{{Min: 1, Max: math.MaxInt32}}, Text: text}}}
	}
	wantDescriptions := []statdesc.Description{
		described,
		line("grants_envy", "Grants Level {0} Envy"),
		line("supported_by_pierce", "Socketed Gems are Supported by Level {0} Pierce"),
		line("cast_on_crit", "Socketed Gems are Supported by Level {0} Cast On Critical Strike"),
		line("grants_icestorm", "Grants Level {0} Icestorm"),
	}
	if diff := cmp.Diff(wantDescriptions, item.Descriptions); diff != "" {
		t.Errorf("descriptions mismatch (-want +got):\n%s", diff)
	}
	wantRanges := map[string]int32{
		"Envy25": 25, "Envy15": 15, "Pierce": 15, "NoGrantPierce": 0, "Described": 0,
		"SharedGrant": 0, "SharedPlain": 30, "Time": 0, "Focus": 0, "CastOnCrit": 20, "Icestorm": 1,
	}
	for _, m := range mods {
		ms := m.Stats[0]
		want := wantRanges[m.ID]
		if ms.Min != want || ms.Max != want || ms.RuthlessMin != want || ms.RuthlessMax != want {
			t.Errorf("mod %s: got range %+v, want %d throughout", m.ID, ms, want)
		}
	}
	if flag := mods[len(mods)-1].Stats[1]; flag.Min != 1 || flag.Max != 1 {
		t.Errorf("mod Icestorm flag: got range %+v, want 1", flag)
	}
	wantWarnings := []string{"stat harbinger: mods granting different effects share it; no grant line"}
	if diff := cmp.Diff(wantWarnings, warnings); diff != "" {
		t.Errorf("warnings mismatch (-want +got):\n%s", diff)
	}
}
