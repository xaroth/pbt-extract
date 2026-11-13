package extract

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
	"github.com/xaroth/pbt-extract/treeexport"
)

// Variant is one tree export to include, such as "standard" from data.json. Hardmode selects
// the ruthless stat columns of PassiveSkills.
type Variant struct {
	Name     string
	Export   *treeexport.Export
	Hardmode bool
}

// treeTables holds the tables trees are joined with.
type treeTables struct {
	passives, mastery, masteryGroups, buffTemplates, buffDefs table
	byGraphID, byMasteryHash                                  map[int]int
	stats                                                     []dataset.Stat
	warn                                                      func(string)
}

func readTreeTables(r ot.Reader, schema *dat.Schema, stats []dataset.Stat) (*treeTables, error) {
	t := &treeTables{stats: stats}
	var err error
	if t.passives, err = readTable(r, schema, "PassiveSkills", "PassiveSkillGraphId", "Stats",
		"Stat1Value", "Stat2Value", "Stat3Value", "Stat4Value", "Stat5Value", "PassiveSkillBuffs", "SkillPointsGranted",
		"HasHardmodeAlternate", "StatsHardmode", "Stat1ValueHardmode", "Stat2ValueHardmode", "Stat3ValueHardmode",
		"Stat4ValueHardmode", "Stat5ValueHardmode", "BuffsHardmode", "SkillPointsGrantedHardmode", "MasteryGroup"); err != nil {
		return nil, err
	}
	if t.masteryGroups, err = readTable(r, schema, "PassiveSkillMasteryGroups", "MasteryCountStat"); err != nil {
		return nil, err
	}
	if t.mastery, err = readTable(r, schema, "PassiveSkillMasteryEffects", "HASH16", "Stats",
		"Stat1Value", "Stat2Value", "Stat3Value"); err != nil {
		return nil, err
	}
	if t.buffTemplates, err = readTable(r, schema, "BuffTemplates", "BuffDefinitionsKey", "Buff_StatValues"); err != nil {
		return nil, err
	}
	if t.buffDefs, err = readTable(r, schema, "BuffDefinitions", "StatsKeys"); err != nil {
		return nil, err
	}
	if t.byGraphID, err = index(t.passives, "PassiveSkillGraphId"); err != nil {
		return nil, err
	}
	if t.byMasteryHash, err = index(t.mastery, "HASH16"); err != nil {
		return nil, err
	}
	return t, nil
}

// index maps an integer key column to row numbers; keys must be unique.
func index(t table, col string) (map[int]int, error) {
	out := make(map[int]int, t.t.Len())
	for r := range t.t.Len() {
		k, err := t.integer(r, col)
		if err != nil {
			return nil, err
		}
		if _, dup := out[k]; dup {
			return nil, fmt.Errorf("%s %d repeated", col, k)
		}
		out[k] = r
	}
	return out, nil
}

// build joins one tree export with the game tables. classes maps the export's class index to
// the character id.
func (tt *treeTables) build(v Variant, classes map[int]string) (dataset.Tree, error) {
	e := v.Export
	t := dataset.Tree{
		Variant:               v.Name,
		TotalPoints:           e.Points.Total,
		AscendancyPoints:      e.Points.Ascendancy,
		AlternateAscendancies: ascendancies(e.AlternateAscendancies),
		JewelSlots:            slices.Clone(e.JewelSlots),
	}
	starts := map[int]uint32{}
	// Sorted so errors and duplicate checks do not depend on map order.
	for _, id := range slices.Sorted(maps.Keys(e.Nodes)) {
		n := e.Nodes[id]
		nd, err := tt.node(n, e, v.Hardmode)
		if errors.Is(err, errNoPassive) && n.IsProxy {
			// New proxy nodes can reach the tree export before the game files have their rows.
			// Without stats they still connect the tree and match the layout.
			if tt.warn != nil {
				tt.warn(fmt.Sprintf("tree %s node %s: proxy node has no PassiveSkills row; carried without stats", v.Name, id))
			}
			nd, err = edges(nd, n, e)
		}
		if err != nil {
			return t, fmt.Errorf("tree %s node %s: %w", v.Name, id, err)
		}
		if n.ClassStartIndex != nil {
			if _, dup := starts[*n.ClassStartIndex]; dup {
				return t, fmt.Errorf("tree %s: class %d has two start nodes", v.Name, *n.ClassStartIndex)
			}
			starts[*n.ClassStartIndex] = n.Skill
		}
		t.Nodes = append(t.Nodes, nd)
	}
	slices.SortFunc(t.Nodes, func(a, b dataset.Node) int { return cmp.Compare(a.ID, b.ID) })
	for i, c := range e.Classes {
		start, ok := starts[i]
		if !ok {
			return t, fmt.Errorf("tree %s: class %s has no start node", v.Name, c.Name)
		}
		char, ok := classes[i]
		if !ok {
			return t, fmt.Errorf("tree %s: class %d (%s) has no Characters row with that IntegerId", v.Name, i, c.Name)
		}
		t.Classes = append(t.Classes, dataset.Class{
			Character:        char,
			BaseStrength:     c.BaseStrength,
			BaseDexterity:    c.BaseDexterity,
			BaseIntelligence: c.BaseIntelligence,
			StartNode:        start,
			Ascendancies:     ascendancies(c.Ascendancies),
		})
	}
	return t, nil
}

func ascendancies(as []treeexport.Ascendancy) []dataset.Ascendancy {
	var out []dataset.Ascendancy
	for _, a := range as {
		out = append(out, dataset.Ascendancy{ID: a.ID})
	}
	return out
}

var errNoPassive = errors.New("no PassiveSkills row")

func (tt *treeTables) node(n treeexport.Node, e *treeexport.Export, hardmode bool) (dataset.Node, error) {
	nd := dataset.Node{
		ID:                   n.Skill,
		Kind:                 kind(n),
		Ascendancy:           n.AscendancyName,
		MultipleChoice:       n.IsMultipleChoice,
		MultipleChoiceOption: n.IsMultipleChoiceOpt,
	}
	row, ok := tt.byGraphID[int(n.Skill)]
	if !ok {
		return nd, errNoPassive
	}
	p := tt.passives
	hard := false
	if hardmode {
		var err error
		if hard, err = p.boolean(row, "HasHardmodeAlternate"); err != nil {
			return nd, err
		}
	}
	statsCol, valueSuffix, buffsCol, pointsCol := "Stats", "Value", "PassiveSkillBuffs", "SkillPointsGranted"
	if hard {
		statsCol, valueSuffix, buffsCol, pointsCol = "StatsHardmode", "ValueHardmode", "BuffsHardmode", "SkillPointsGrantedHardmode"
	}
	var err error
	if nd.Stats, err = tt.statValues(p, row, statsCol, "Stat%d"+valueSuffix); err != nil {
		return nd, err
	}
	if nd.Stats, err = tt.masteryCount(row, nd.Stats); err != nil {
		return nd, err
	}
	slices.SortFunc(nd.Stats, func(a, b dataset.StatValue) int { return cmp.Compare(a.Stat, b.Stat) })
	for i := 1; i < len(nd.Stats); i++ {
		if nd.Stats[i-1].Stat == nd.Stats[i].Stat {
			return nd, fmt.Errorf("stat %s repeated", tt.stats[nd.Stats[i].Stat].ID)
		}
	}
	if nd.SkillPoints, err = p.i32(row, pointsCol); err != nil {
		return nd, err
	}
	buffs, err := p.ints(row, buffsCol)
	if err != nil {
		return nd, err
	}
	for _, b := range buffs {
		svs, err := tt.buff(b)
		if err != nil {
			return nd, err
		}
		if svs != nil {
			nd.Buffs = append(nd.Buffs, svs)
		}
	}
	for _, me := range n.MasteryEffects {
		mr, ok := tt.byMasteryHash[int(me.Effect)]
		if !ok {
			return nd, fmt.Errorf("mastery effect %d not in PassiveSkillMasteryEffects", me.Effect)
		}
		svs, err := tt.statValues(tt.mastery, mr, "Stats", "Stat%dValue")
		if err != nil {
			return nd, err
		}
		nd.MasteryEffects = append(nd.MasteryEffects, dataset.MasteryEffect{ID: me.Effect, Stats: svs})
	}
	return edges(nd, n, e)
}

// edges sets nd's neighbours from the node's edges in both directions.
func edges(nd dataset.Node, n treeexport.Node, e *treeexport.Export) (dataset.Node, error) {
	seen := map[uint32]bool{}
	for _, list := range [][]string{n.Out, n.In} {
		for _, s := range list {
			id, err := strconv.ParseUint(s, 10, 32)
			if err != nil {
				return nd, fmt.Errorf("edge %q: %w", s, err)
			}
			if _, ok := e.Nodes[s]; !ok {
				return nd, fmt.Errorf("edge to unknown node %s", s)
			}
			if !seen[uint32(id)] {
				seen[uint32(id)] = true
				nd.Neighbours = append(nd.Neighbours, uint32(id))
			}
		}
	}
	slices.Sort(nd.Neighbours)
	return nd, nil
}

// masteryCount adds the count stat of the node's mastery group, at 1, to stats: the game
// counts allocated masteries of a group in that stat (PassiveSkillMasteryGroups
// MasteryCountStat, such as number_of_life_masteries), which effects like "10% more Maximum
// Life if you have at least 6 Life Masteries allocated" read. Nodes without a group are left
// as they are.
func (tt *treeTables) masteryCount(row int, stats []dataset.StatValue) ([]dataset.StatValue, error) {
	group, err := tt.passives.ref(row, "MasteryGroup")
	if err != nil || group < 0 {
		return stats, err
	}
	stat, err := tt.masteryGroups.ref(group, "MasteryCountStat")
	if err != nil || stat < 0 {
		return stats, err
	}
	if stat >= len(tt.stats) {
		return stats, fmt.Errorf("mastery count stat row %d out of range", stat)
	}
	return append(stats, dataset.StatValue{Stat: uint32(stat), Value: 1}), nil
}

func kind(n treeexport.Node) dataset.NodeKind {
	switch {
	// A proxy is never allocated, whatever else the export marks it as.
	case n.IsProxy:
		return dataset.Proxy
	case n.ClassStartIndex != nil:
		return dataset.ClassStart
	case n.IsAscendancyStart:
		return dataset.AscendancyStart
	case n.IsMastery:
		return dataset.Mastery
	case n.IsJewelSocket:
		return dataset.JewelSocket
	case n.IsKeystone:
		return dataset.Keystone
	case n.IsNotable:
		return dataset.Notable
	}
	return dataset.Normal
}

// statValues pairs the stat keys in statsCol with the value columns named by valueFmt
// (Stat1Value, Stat2Value, ...). Keys beyond the value columns are an error.
func (tt *treeTables) statValues(t table, row int, statsCol, valueFmt string) ([]dataset.StatValue, error) {
	keys, err := t.ints(row, statsCol)
	if err != nil {
		return nil, err
	}
	var out []dataset.StatValue
	for i, k := range keys {
		col := fmt.Sprintf(valueFmt, i+1)
		if _, ok := t.cols[col]; !ok {
			return nil, fmt.Errorf("%d stats but no column %s", len(keys), col)
		}
		v, err := t.i32(row, col)
		if err != nil {
			return nil, err
		}
		if k < 0 || k >= len(tt.stats) {
			return nil, fmt.Errorf("stat row %d out of range", k)
		}
		out = append(out, dataset.StatValue{Stat: uint32(k), Value: v})
	}
	return out, nil
}

// buff pairs a BuffTemplates row's values with its BuffDefinitions stats.
func (tt *treeTables) buff(row int) ([]dataset.StatValue, error) {
	def, err := tt.buffTemplates.ref(row, "BuffDefinitionsKey")
	if err != nil {
		return nil, err
	}
	if def < 0 {
		return nil, fmt.Errorf("BuffTemplates row %d has no definition", row)
	}
	keys, err := tt.buffDefs.ints(def, "StatsKeys")
	if err != nil {
		return nil, err
	}
	vals, err := tt.buffTemplates.i32s(row, "Buff_StatValues")
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, nil // visual-only buff
	}
	// Some templates carry extra trailing values with no stat; they are not displayed.
	if len(vals) < len(keys) {
		return nil, fmt.Errorf("BuffTemplates row %d: %d values for %d stats", row, len(vals), len(keys))
	}
	vals = vals[:len(keys)]
	var out []dataset.StatValue
	for i, v := range vals {
		if keys[i] < 0 || keys[i] >= len(tt.stats) {
			return nil, fmt.Errorf("stat row %d out of range", keys[i])
		}
		out = append(out, dataset.StatValue{Stat: uint32(keys[i]), Value: v})
	}
	return out, nil
}
