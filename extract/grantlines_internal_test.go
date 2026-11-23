package extract

import (
	"math"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/xaroth/pbt-engine/dataset"
	"github.com/xaroth/pbt-engine/statdesc"
)

var testGrantTemplates = grantTemplates{
	skill:   "Grants Level {1} {0}",
	support: "Socketed Gems are Supported by Level {1} {0}",
}

func grantOf(effect string, level int32) []dataset.GrantedSkill {
	return []dataset.GrantedSkill{{Effect: effect, Level: level}}
}

func statAt(h uint32, v int32) []dataset.ModStat {
	return []dataset.ModStat{{Stat: h, Min: v, Max: v, RuthlessMin: v, RuthlessMax: v}}
}

func checkRanges(t *testing.T, mods []dataset.Mod, want map[string]int32) {
	t.Helper()
	for _, m := range mods {
		ms := m.Stats[0]
		w := want[m.ID]
		if ms.Min != w || ms.Max != w || ms.RuthlessMin != w || ms.RuthlessMax != w {
			t.Errorf("mod %s: got range %+v, want %d throughout", m.ID, ms, w)
		}
	}
}

func TestRebuildGrantLines(t *testing.T) {
	t.Parallel()
	stats := []dataset.Stat{
		{ID: "grants_envy"}, {ID: "supported_by_pierce"}, {ID: "described"}, {ID: "shared"}, {ID: "harbinger"},
		{ID: "cast_on_crit"}, {ID: "grants_icestorm"}, {ID: "sound_flag"}, {ID: "earthbreaker"},
	}
	names := map[string]grantName{
		"Envy":                        {name: "Envy"},
		"SupportPierce":               {name: "Pierce", support: true},
		"SupportCastOnCriticalStrike": {name: "Cast On Critical Strike", support: true},
		"HarbingerOfTime":             {name: "Summon Harbinger of Time"},
		"HarbingerOfFocus":            {name: "Summon Harbinger of Focus"},
		"Icestorm":                    {name: "Icestorm"},
		"SupportEarthbreaker":         {name: "Earthbreaker", support: true},
	}
	mods := []dataset.Mod{
		{ID: "Envy25", Stats: statAt(0, 0), GrantedSkills: grantOf("Envy", 25)},
		// already at its level, as in a few 3.30 mods
		{ID: "Envy15", Stats: statAt(0, 15), GrantedSkills: grantOf("Envy", 15)},
		{ID: "Pierce", Stats: statAt(1, 0), GrantedSkills: grantOf("SupportPierce", 15)},
		// a mod without a grant holds the stat at 0 and shows no line
		{ID: "NoGrantPierce", Stats: statAt(1, 0)},
		{ID: "Described", Stats: statAt(2, 0), GrantedSkills: grantOf("Envy", 1)},
		// a mod without a grant that rolls the stat: not a grant stat
		{ID: "SharedGrant", Stats: statAt(3, 0), GrantedSkills: grantOf("Envy", 1)},
		{ID: "SharedPlain", Stats: statAt(3, 30)},
		// different effects: keyed in mod order
		{ID: "Time", Stats: statAt(4, 0), GrantedSkills: grantOf("HarbingerOfTime", 1)},
		{ID: "Focus", Stats: statAt(4, 0), GrantedSkills: grantOf("HarbingerOfFocus", 1)},
		{ID: "Time2", Stats: statAt(4, 0), GrantedSkills: grantOf("HarbingerOfTime", 1)},
		// the triggered half has no name
		{ID: "CastOnCrit", Stats: statAt(5, 0), GrantedSkills: []dataset.GrantedSkill{
			{Effect: "SupportCastOnCriticalStrike", Level: 20}, {Effect: "SupportCastOnCritTriggered", Level: 20},
		}},
		// an undescribed flag at the grant's level beside the grant stat, as on The Whispering Ice
		{ID: "Icestorm", Stats: append(statAt(6, 0), statAt(7, 1)...), GrantedSkills: grantOf("Icestorm", 1)},
		// a mod granting the same effect at another value, as OneAncestorTotemBuffUnique__1's Earthbreaker
		{ID: "Earthbreaker20", Stats: statAt(8, 0), GrantedSkills: grantOf("SupportEarthbreaker", 20)},
		{ID: "Earthbreaker25", Stats: statAt(8, 1), GrantedSkills: grantOf("SupportEarthbreaker", 25)},
	}
	described := statdesc.Description{Stats: []string{"described"}, Variants: []statdesc.Variant{{Text: "kept"}}}
	item := statdesc.Scope{Name: itemScope, Descriptions: []statdesc.Description{described}}
	rebuildGrantLines(mods, stats, &item, names, testGrantTemplates)

	line := func(id, text string) statdesc.Description {
		return statdesc.Description{Stats: []string{id}, Variants: []statdesc.Variant{{Limits: []statdesc.Limit{{Min: 1, Max: math.MaxInt32}}, Text: text}}}
	}
	keyed := func(k int32, text string) statdesc.Variant {
		return statdesc.Variant{Limits: []statdesc.Limit{{Min: k, Max: k}}, Text: text}
	}
	wantDescriptions := []statdesc.Description{
		described,
		line("grants_envy", "Grants Level {0} Envy"),
		line("supported_by_pierce", "Socketed Gems are Supported by Level {0} Pierce"),
		{Stats: []string{"harbinger"}, Variants: []statdesc.Variant{
			keyed(1, "Grants Level 1 Summon Harbinger of Time"),
			keyed(2, "Grants Level 1 Summon Harbinger of Focus"),
		}},
		line("cast_on_crit", "Socketed Gems are Supported by Level {0} Cast On Critical Strike"),
		line("grants_icestorm", "Grants Level {0} Icestorm"),
		line("earthbreaker", "Socketed Gems are Supported by Level {0} Earthbreaker"),
	}
	if diff := cmp.Diff(wantDescriptions, item.Descriptions); diff != "" {
		t.Errorf("descriptions mismatch (-want +got):\n%s", diff)
	}
	checkRanges(t, mods, map[string]int32{
		"Envy25": 25, "Envy15": 15, "Pierce": 15, "NoGrantPierce": 0, "Described": 0,
		"SharedGrant": 0, "SharedPlain": 30, "Time": 1, "Focus": 2, "Time2": 1, "CastOnCrit": 20, "Icestorm": 1,
		"Earthbreaker20": 20, "Earthbreaker25": 25,
	})
	if flag := mods[len(mods)-3].Stats[1]; flag.Min != 1 || flag.Max != 1 {
		t.Errorf("mod Icestorm flag: got range %+v, want 1", flag)
	}
}

func TestLevelDescribedGrants(t *testing.T) {
	t.Parallel()
	stats := []dataset.Stat{{ID: "frostbite"}, {ID: "fire_burst"}}
	names := map[string]grantName{
		"Frostbite":          {name: "Frostbite"},
		"TriggeredExplosion": {name: "Fire Burst"},
	}
	mods := []dataset.Mod{
		{ID: "Frostbite", Stats: statAt(0, 0), GrantedSkills: grantOf("Frostbite", 5)},
		// a line other than the template's
		{ID: "FireBurst", Stats: statAt(1, 0), GrantedSkills: grantOf("TriggeredExplosion", 1)},
	}
	item := statdesc.Scope{Name: itemScope, Descriptions: []statdesc.Description{
		{Stats: []string{"frostbite"}, Variants: []statdesc.Variant{{Text: "Grants Level {0} [Frostbite::{0}|Frostbite]"}}},
		{Stats: []string{"fire_burst"}, Variants: []statdesc.Variant{{Text: "Trigger Level {0} [TriggeredExplosion::{0}|Fire Burst] on Kill"}}},
	}}
	levelDescribedGrants(mods, stats, &item, names, testGrantTemplates)
	checkRanges(t, mods, map[string]int32{"Frostbite": 5, "FireBurst": 0})
}
