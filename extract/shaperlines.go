package extract

import (
	"fmt"
	"slices"
	"strings"

	"github.com/xaroth/pbt-engine/statdesc"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
)

// shaperHandler names the Shaper Memory skill that the value of shaper_apparition_ability
// gives: from 3.30 the line "Grants Level 20 {0}, which will be used by Shaper Memory" reads it.
const shaperHandler = "shaper_apparition_ability"

type shaperAbilityRow struct {
	Name string `dat:"Name"`
}

// shaperLines gives each item description variant that reads shaperHandler one variant per
// ShaperApparitionAbility name, as the descriptions up to 3.29 have them ("1" for "Grants Level
// 20 Shaper's Devastation, ..."), since no handler of the engine names them. A client without
// the handler changes nothing.
func shaperLines(r ot.Reader, s *dat.Schema, scopes []statdesc.Scope, warn func(string)) error {
	i := slices.IndexFunc(scopes, func(sc statdesc.Scope) bool { return sc.Name == itemScope })
	if i < 0 || !readsHandler(scopes[i], shaperHandler) {
		return nil
	}
	rows, err := readOptionalRows[shaperAbilityRow](r, s, "ShaperApparitionAbility", warn)
	if err != nil {
		return err
	}
	names := make([]string, len(rows))
	for j, row := range rows {
		names[j] = row.Name
	}
	return nameHandlerValues(&scopes[i], shaperHandler, names)
}

// readsHandler reports whether a variant of sc reads handler.
func readsHandler(sc statdesc.Scope, handler string) bool {
	for _, d := range sc.Descriptions {
		for _, v := range d.Variants {
			if slices.ContainsFunc(v.Handlers, func(h statdesc.Handler) bool { return h.Name == handler }) {
				return true
			}
		}
	}
	return false
}

// nameHandlerValues replaces each variant that reads handler on its one stat with one variant per
// value its limit allows and names[value-1] has a name, the name in place of {0}.
func nameHandlerValues(sc *statdesc.Scope, handler string, names []string) error {
	for di := range sc.Descriptions {
		d := &sc.Descriptions[di]
		var out []statdesc.Variant
		for _, v := range d.Variants {
			k := slices.IndexFunc(v.Handlers, func(h statdesc.Handler) bool { return h.Name == handler })
			if k < 0 {
				out = append(out, v)
				continue
			}
			if len(d.Stats) != 1 || v.Handlers[k].Stat != 0 || len(v.Handlers[k].Args) > 0 || !strings.Contains(v.Text, "{0}") {
				return fmt.Errorf("stat %s: handler %s on a description it cannot name", strings.Join(d.Stats, " "), handler)
			}
			l := v.Limits[0]
			var others []statdesc.Handler
			for _, h := range v.Handlers {
				if h.Name != handler {
					others = append(others, h)
				}
			}
			for j, name := range names {
				value := int32(j + 1)
				if name == "" || l.Not == (value >= l.Min && value <= l.Max) {
					continue
				}
				out = append(out, statdesc.Variant{
					Limits:   []statdesc.Limit{{Min: value, Max: value}},
					Text:     strings.ReplaceAll(v.Text, "{0}", name),
					Handlers: others,
				})
			}
		}
		d.Variants = out
	}
	return nil
}
