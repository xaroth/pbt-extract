package extract

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/treeexport"
)

// layout places the nodes of one tree export for drawing.
func layout(v Variant) (dataset.Layout, error) {
	e := v.Export
	l := dataset.Layout{Variant: v.Name, MinX: e.MinX, MinY: e.MinY, MaxX: e.MaxX, MaxY: e.MaxY}
	c := e.Constants
	if len(c.SkillsPerOrbit) != len(c.OrbitRadii) {
		return l, fmt.Errorf("tree %s: %d orbit node counts but %d orbit radii", v.Name, len(c.SkillsPerOrbit), len(c.OrbitRadii))
	}
	for i, n := range c.SkillsPerOrbit {
		if n == 0 {
			return l, fmt.Errorf("tree %s: orbit %d has %d nodes", v.Name, i, n)
		}
		l.Orbits = append(l.Orbits, dataset.Orbit{Radius: c.OrbitRadii[i], Slots: n})
	}
	var err error
	if l.Groups, err = layoutGroups(e.Groups); err != nil {
		return l, fmt.Errorf("tree %s: %w", v.Name, err)
	}
	for _, id := range slices.Sorted(maps.Keys(e.Nodes)) {
		n := e.Nodes[id]
		l.Nodes = append(l.Nodes, dataset.LayoutNode{
			ID: n.Skill, Group: n.Group, Orbit: n.Orbit, OrbitIndex: n.OrbitIndex, Icon: n.Icon,
			MasteryInactiveIcon: n.InactiveIcon, MasteryActiveIcon: n.ActiveIcon, MasteryActiveEffect: n.ActiveEffectImage,
		})
	}
	slices.SortFunc(l.Nodes, func(a, b dataset.LayoutNode) int { return cmp.Compare(a.ID, b.ID) })
	for _, k := range slices.Sorted(maps.Keys(e.ExtraImages)) {
		class, err := strconv.ParseUint(k, 10, 32)
		if err != nil {
			return l, fmt.Errorf("tree %s: extra image key %q: %w", v.Name, k, err)
		}
		img := e.ExtraImages[k]
		l.ClassArt = append(l.ClassArt, dataset.ClassArt{Class: uint32(class), Image: img.Image, X: img.X, Y: img.Y})
	}
	slices.SortFunc(l.ClassArt, func(a, b dataset.ClassArt) int { return cmp.Compare(a.Class, b.Class) })
	return l, nil
}

func layoutGroups(groups map[string]treeexport.Group) ([]dataset.LayoutGroup, error) {
	out := make([]dataset.LayoutGroup, 0, len(groups))
	for k, g := range groups {
		id, err := strconv.ParseUint(k, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("group key %q: %w", k, err)
		}
		lg := dataset.LayoutGroup{ID: uint32(id), X: g.X, Y: g.Y, Orbits: g.Orbits, Proxy: g.IsProxy}
		if b := g.Background; b != nil {
			// The dataset has no offset, so fail rather than draw the background in the wrong place.
			if b.OffsetX != 0 || b.OffsetY != 0 {
				return nil, fmt.Errorf("group %s: background offset (%v, %v) not supported", k, b.OffsetX, b.OffsetY)
			}
			lg.Background = &dataset.GroupBackground{Image: b.Image, HalfImage: b.IsHalfImage}
		}
		out = append(out, lg)
	}
	slices.SortFunc(out, func(a, b dataset.LayoutGroup) int { return cmp.Compare(a.ID, b.ID) })
	return out, nil
}
