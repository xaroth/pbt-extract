package extract

import (
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
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

// line returns the grant line naming n, with level in place of the level, such as "{0}".
func (t grantTemplates) line(n grantName, level string) string {
	template := t.skill
	if n.support {
		template = t.support
	}
	return strings.NewReplacer("{0}", n.name, "{1}", level).Replace(template)
}

// grantLines gives item mods the grant lines of a client that builds them from ClientStrings
// rather than describing their stats (3.30 on), see levelDescribedGrants and rebuildGrantLines. A client without the
// templates describes the stats, and nothing changes.
func grantLines(r ot.Reader, s *dat.Schema, mods []dataset.Mod, stats []dataset.Stat, scopes []statdesc.Scope) error {
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
	levelDescribedGrants(mods, stats, &scopes[i], names, t)
	rebuildGrantLines(mods, stats, &scopes[i], names, t)
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
// a value that reads as their grant, as the descriptions up to 3.29 have them. A mod that grants
// one named effect has at most one grant stat (see grantStat). A stat is given a line when it is
// the grant stat of some mods and every other mod with it holds it at [0, 0] or grants the same
// effect as all those mods:
//
//   - When they all grant the same effect, the line names it with the level as the value ("Grants
//     Level {0} Envy" for local_display_grants_level_X_envy), and each mod with the stat that
//     grants the effect gets its grant's level as the range, whatever it held: 3.30 holds
//     Earthbreaker's at 1 on OneAncestorTotemBuffUnique__1, whose grant is level 25.
//   - When they grant different effects, such as local_display_summon_harbinger_x_on_equip, the
//     line has one variant per effect and level, keyed 1, 2, ... in mod order ("Grants Level 1
//     Summon Harbinger of Time" at 2), and each mod gets its key as the range.
func rebuildGrantLines(mods []dataset.Mod, stats []dataset.Stat, item *statdesc.Scope, names map[string]grantName, t grantTemplates) {
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
	type grant struct {
		name  grantName
		level int32
	}
	for _, h := range slices.Sorted(maps.Keys(uses)) {
		var owners, others []use
		effects := map[grantName]bool{}
		for _, u := range uses[h] {
			if own[u] {
				owners = append(owners, u)
				_, n, _ := namedGrant(mods[u.mod].GrantedSkills, names)
				effects[n] = true
			} else {
				others = append(others, u)
			}
		}
		if len(owners) == 0 {
			continue
		}
		ok := true
		for _, u := range others {
			ms := mods[u.mod].Stats[u.stat]
			if ms.Min == 0 && ms.Max == 0 {
				continue
			}
			_, n, one := namedGrant(mods[u.mod].GrantedSkills, names)
			if len(effects) == 1 && one && effects[n] {
				owners = append(owners, u)
				continue
			}
			ok = false
		}
		if !ok {
			continue
		}
		id := stats[h].ID
		if len(effects) == 1 {
			n := slices.Collect(maps.Keys(effects))[0]
			text := t.line(n, "{0}")
			item.Descriptions = append(item.Descriptions, statdesc.Description{
				Stats:    []string{id},
				Variants: []statdesc.Variant{{Limits: []statdesc.Limit{{Min: 1, Max: math.MaxInt32}}, Text: text}},
			})
			for _, u := range owners {
				g, _, _ := namedGrant(mods[u.mod].GrantedSkills, names)
				setGrantValue(&mods[u.mod].Stats[u.stat], g.Level)
			}
			continue
		}
		keys := map[grant]int32{}
		d := statdesc.Description{Stats: []string{id}}
		for _, u := range owners {
			g, n, _ := namedGrant(mods[u.mod].GrantedSkills, names)
			k, seen := keys[grant{n, g.Level}]
			if !seen {
				k = int32(len(keys) + 1)
				keys[grant{n, g.Level}] = k
				text := t.line(n, strconv.Itoa(int(g.Level)))
				d.Variants = append(d.Variants, statdesc.Variant{Limits: []statdesc.Limit{{Min: k, Max: k}}, Text: text})
			}
			setGrantValue(&mods[u.mod].Stats[u.stat], k)
		}
		item.Descriptions = append(item.Descriptions, d)
	}
}

// levelDescribedGrants gives a mod that grants one named effect its grant's level as the range
// of a stat at [0, 0] whose description reads the grant line the templates build for it, such
// as local_display_grants_skill_frostbite_level ("Grants Level {0} [Frostbite::{0}|Frostbite]"):
// 3.30 still describes a few grant stats but holds them at 0, as it does the ones it no longer
// describes.
func levelDescribedGrants(mods []dataset.Mod, stats []dataset.Stat, item *statdesc.Scope, names map[string]grantName, t grantTemplates) {
	texts := map[string][]string{}
	for _, d := range item.Descriptions {
		if len(d.Stats) != 1 {
			continue
		}
		for _, v := range d.Variants {
			texts[d.Stats[0]] = append(texts[d.Stats[0]], tagText(v.Text))
		}
	}
	for i, m := range mods {
		g, n, one := namedGrant(m.GrantedSkills, names)
		if !one {
			continue
		}
		for j, ms := range m.Stats {
			if ms.Min != 0 || ms.Max != 0 {
				continue
			}
			if slices.ContainsFunc(texts[stats[ms.Stat].ID], func(s string) bool { return strings.Contains(s, t.line(n, "{0}")) }) {
				setGrantValue(&mods[i].Stats[j], g.Level)
			}
		}
	}
}

// tagText returns text with each [tag|shown] or [tag] as its shown part.
func tagText(text string) string {
	var b strings.Builder
	for {
		open := strings.IndexByte(text, '[')
		if open < 0 {
			break
		}
		end := strings.IndexByte(text[open:], ']')
		if end < 0 {
			break
		}
		tag := text[open+1 : open+end]
		if bar := strings.LastIndexByte(tag, '|'); bar >= 0 {
			tag = tag[bar+1:]
		}
		b.WriteString(text[:open])
		b.WriteString(tag)
		text = text[open+end+1:]
	}
	b.WriteString(text)
	return b.String()
}

// setGrantValue sets ms to v, and its Ruthless range too where that is [0, 0] or the same as
// the range it replaces, as for a mod without Ruthless values.
func setGrantValue(ms *dataset.ModStat, v int32) {
	if ms.RuthlessMin == 0 && ms.RuthlessMax == 0 || ms.RuthlessMin == ms.Min && ms.RuthlessMax == ms.Max {
		ms.RuthlessMin, ms.RuthlessMax = v, v
	}
	ms.Min, ms.Max = v, v
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
