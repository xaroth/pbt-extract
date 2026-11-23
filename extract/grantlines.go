package extract

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strings"

	"github.com/xaroth/pbt-engine/dataset"
	"github.com/xaroth/pbt-engine/statdesc"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
)

// itemScope is the stat description scope of item mods.
const itemScope = "Metadata/StatDescriptions/stat_descriptions.txt"

// The ClientStrings ids of the lines the game builds from a mod's granted effect, from 3.30 on:
// "Grants Level {1} {0}" and "Socketed Gems are Supported by Level {1} {0}", {0} the effect's name
// and {1} its level.
const (
	skillGrantString   = "ModGrantsEffectWithLevel"
	supportGrantString = "ModGrantsSupportEffectWithLevel"
)

type clientStringRow struct {
	ID   string `dat:"Id"`
	Text string `dat:"Text"`
}

type grantNameRow struct {
	ID          string  `dat:"Id"`
	IsSupport   bool    `dat:"IsSupport"`
	ActiveSkill dat.Ref `dat:"ActiveSkill" ref:"ActiveSkills"`
	SupportName string  `dat:"SupportName"`
}

type activeNameRow struct {
	Name string `dat:"DisplayedName"`
}

// grantName is the name a grant line gives a granted effect.
type grantName struct {
	name    string
	support bool
}

// grantTemplates are the ClientStrings texts of the grant lines.
type grantTemplates struct {
	skill, support string
}

// grantLines gives item mods the grant lines of a client that builds them from ClientStrings
// rather than describing their stats (3.30 on), see rebuildGrantLines. A client without the
// templates describes the stats, and nothing changes.
func grantLines(r ot.Reader, s *dat.Schema, mods []dataset.Mod, stats []dataset.Stat, scopes []statdesc.Scope, warn func(string)) error {
	strs, err := readRows[clientStringRow](r, s, "ClientStrings")
	if err != nil {
		return err
	}
	var t grantTemplates
	for _, c := range strs {
		switch c.ID {
		case skillGrantString:
			t.skill = c.Text
		case supportGrantString:
			t.support = c.Text
		}
	}
	if t.skill == "" && t.support == "" {
		return nil
	}
	if t.skill == "" || t.support == "" {
		return fmt.Errorf("table ClientStrings: has one of %s and %s, not both", skillGrantString, supportGrantString)
	}
	i := slices.IndexFunc(scopes, func(sc statdesc.Scope) bool { return sc.Name == itemScope })
	if i < 0 {
		return fmt.Errorf("no stat description scope %s", itemScope)
	}
	names, err := grantNames(r, s)
	if err != nil {
		return err
	}
	rebuildGrantLines(mods, stats, &scopes[i], names, t, warn)
	return nil
}

// grantNames returns the name a grant line gives each granted effect, by id: an active skill's
// displayed name, or a support's name without " Support", as the descriptions up to 3.29 name
// it ("Supported by Level 20 Concentrated Effect"). An effect without a name, such as the
// triggered part of Cast On Critical Strike, is left out.
func grantNames(r ot.Reader, s *dat.Schema) (map[string]grantName, error) {
	effects, err := readRows[grantNameRow](r, s, "GrantedEffects")
	if err != nil {
		return nil, err
	}
	actives, err := readRows[activeNameRow](r, s, "ActiveSkills")
	if err != nil {
		return nil, err
	}
	out := map[string]grantName{}
	for i, e := range effects {
		n := grantName{support: e.IsSupport}
		if e.IsSupport {
			n.name = strings.TrimSuffix(e.SupportName, " Support")
		} else if _, ok := e.ActiveSkill.Row(); ok {
			a, err := lookup(e.ActiveSkill, actives, "ActiveSkills")
			if err != nil {
				return nil, fmt.Errorf("table GrantedEffects row %d: %w", i, err)
			}
			n.name = a.Name
		}
		if n.name != "" {
			out[e.ID] = n
		}
	}
	return out, nil
}

// rebuildGrantLines gives each grant stat a description built from the templates, and its mods
// the level of their grant as its range, as the descriptions up to 3.29 have them ("Grants Level
// {0} Envy Skill" for local_display_grants_level_X_envy at [25, 25]). A mod that grants one named
// effect has at most one grant stat (see grantStat). A stat is given a line when it is the grant
// stat of some mods, all granting the same effect, and every other mod with it holds it at
// [0, 0]. A stat that mods granting different effects share, such as
// local_display_summon_harbinger_x_on_equip, cannot name them in one line: it is left as it is,
// with a warning.
func rebuildGrantLines(mods []dataset.Mod, stats []dataset.Stat, item *statdesc.Scope, names map[string]grantName, t grantTemplates, warn func(string)) {
	described := map[string]bool{}
	for _, d := range item.Descriptions {
		for _, id := range d.Stats {
			described[id] = true
		}
	}
	type use struct{ mod, stat int }
	uses := map[uint32][]use{}
	own := map[use]bool{} // the grant stat of its mod
	for i, m := range mods {
		for j, ms := range m.Stats {
			if !described[stats[ms.Stat].ID] {
				uses[ms.Stat] = append(uses[ms.Stat], use{i, j})
			}
		}
		if g, _, one := namedGrant(m.GrantedSkills, names); one {
			if j := grantStat(m, g.Level, stats, described); j >= 0 {
				own[use{i, j}] = true
			}
		}
	}
	for _, h := range slices.Sorted(maps.Keys(uses)) {
		var line *grantName
		levels := map[use]int32{}
		ok, differ := true, false
		for _, u := range uses[h] {
			ms := mods[u.mod].Stats[u.stat]
			if !own[u] {
				ok = ok && ms.Min == 0 && ms.Max == 0
				continue
			}
			g, n, _ := namedGrant(mods[u.mod].GrantedSkills, names)
			if line != nil && *line != n {
				differ = true
			}
			line, levels[u] = &n, g.Level
		}
		if !ok || line == nil {
			continue
		}
		id := stats[h].ID
		if differ {
			warn(fmt.Sprintf("stat %s: mods granting different effects share it; no grant line", id))
			continue
		}
		template := t.skill
		if line.support {
			template = t.support
		}
		text := strings.NewReplacer("{0}", line.name, "{1}", "{0}").Replace(template)
		item.Descriptions = append(item.Descriptions, statdesc.Description{
			Stats:    []string{id},
			Variants: []statdesc.Variant{{Limits: []statdesc.Limit{{Min: 1, Max: math.MaxInt32}}, Text: text}},
		})
		for u, level := range levels {
			ms := &mods[u.mod].Stats[u.stat]
			ms.Min, ms.Max = level, level
			if ms.RuthlessMin == 0 && ms.RuthlessMax == 0 {
				ms.RuthlessMin, ms.RuthlessMax = level, level
			}
		}
	}
}

// grantStat returns the index in m.Stats of the stat that carries m's grant: its one undescribed
// stat at [0, 0], as 3.30 holds them, or without one its only undescribed stat if that is at the
// grant's level, as a few 3.30 mods have it. Otherwise -1: an undescribed flag such as
// sound_on_rare_kill_ice_whisper at 1 beside the grant stat is not one.
func grantStat(m dataset.Mod, level int32, stats []dataset.Stat, described map[string]bool) int {
	var zero, other []int
	for j, ms := range m.Stats {
		switch {
		case described[stats[ms.Stat].ID]:
		case ms.Min == 0 && ms.Max == 0:
			zero = append(zero, j)
		default:
			other = append(other, j)
		}
	}
	switch {
	case len(zero) == 1:
		return zero[0]
	case len(zero) == 0 && len(other) == 1 && m.Stats[other[0]].Min == level && m.Stats[other[0]].Max == level:
		return other[0]
	}
	return -1
}

// namedGrant returns the one grant of gs that has a name, and reports whether there is exactly
// one.
func namedGrant(gs []dataset.GrantedSkill, names map[string]grantName) (dataset.GrantedSkill, grantName, bool) {
	var g dataset.GrantedSkill
	var n grantName
	count := 0
	for _, x := range gs {
		if xn, ok := names[x.Effect]; ok {
			g, n = x, xn
			count++
		}
	}
	return g, n, count == 1
}
