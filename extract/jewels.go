package extract

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
	"github.com/xaroth/pbt-extract/treeexport"
)

type jewelRadiusRow struct {
	ID        string `dat:"ID"`
	Radius    int32  `dat:"Radius"`
	RingInner int32  `dat:"RingInnerRadius"`
	RingOuter int32  `dat:"RingOuterRadius"`
}

type jewelSlotRow struct {
	Slot  dat.Ref `dat:"Slot" ref:"PassiveSkills"`
	Size  dat.Ref `dat:"ClusterJewelSize" ref:"PassiveTreeExpansionJewelSizes"`
	Index int32   `dat:"ClusterIndex"`
	// Parent is the slot whose cluster jewel holds this socket.
	Parent       dat.Ref `dat:"ReplacesSlot" ref:"PassiveJewelSlots"`
	Proxy        dat.Ref `dat:"ProxySlot" ref:"PassiveSkills"`
	StartIndices []int32 `dat:"StartIndices"`
}

type clusterJewelRow struct {
	Base           dat.Ref `dat:"BaseItemTypesKey" ref:"BaseItemTypes"`
	Size           dat.Ref `dat:"PassiveTreeExpansionJewelSizesKey" ref:"PassiveTreeExpansionJewelSizes"`
	MinNodes       int32   `dat:"MinNodes"`
	MaxNodes       int32   `dat:"MaxNodes"`
	SmallIndices   []int32 `dat:"SmallIndices"`
	NotableIndices []int32 `dat:"NotableIndices"`
	SocketIndices  []int32 `dat:"SocketIndices"`
	TotalIndices   int32   `dat:"TotalIndices"`
}

type clusterSkillRow struct {
	Passive dat.Ref `dat:"PassiveSkillsKey" ref:"PassiveSkills"`
	Mastery dat.Ref `dat:"Mastery_PassiveSkillsKey" ref:"PassiveSkills"`
	Tag     dat.Ref `dat:"TagsKey" ref:"Tags"`
	Size    dat.Ref `dat:"PassiveTreeExpansionJewelSizesKey" ref:"PassiveTreeExpansionJewelSizes"`
}

type clusterNotableRow struct {
	Passive dat.Ref `dat:"PassiveSkillsKey" ref:"PassiveSkills"`
	Stat    dat.Ref `dat:"StatsKey" ref:"Stats"`
}

type graphIDRow struct {
	GraphID uint16 `dat:"PassiveSkillGraphId"`
}

type sizeRow struct {
	Name string `dat:"Name"`
}

// clusterSizes maps PassiveTreeExpansionJewelSizes names to cluster sizes.
var clusterSizes = map[string]dataset.ClusterSize{
	"Small":  dataset.SmallCluster,
	"Medium": dataset.MediumCluster,
	"Large":  dataset.LargeCluster,
}

// jewels reads the jewel radii and the cluster jewel tables. Rows naming passives name them
// by graph id, the tree's node id.
func jewels(r ot.Reader, s *dat.Schema, nStats int) ([]dataset.JewelRadius, dataset.Clusters, error) {
	var c dataset.Clusters
	radii, err := readRows[jewelRadiusRow](r, s, "PassiveJewelRadii")
	if err != nil {
		return nil, c, err
	}
	var out []dataset.JewelRadius
	for _, x := range radii {
		out = append(out, dataset.JewelRadius{ID: x.ID, Radius: x.Radius, RingInner: x.RingInner, RingOuter: x.RingOuter})
	}
	ct, err := readClusterTables(r, s)
	if err != nil {
		return nil, c, err
	}
	c, err = ct.build(nStats)
	return out, c, err
}

// clusterTables are the tables cluster jewel data is read from.
type clusterTables struct {
	slots    []jewelSlotRow
	jewels   []clusterJewelRow
	skills   []clusterSkillRow
	notables []clusterNotableRow
	sizes    []sizeRow
	passives []graphIDRow
	bases    []string
	tags     []string
}

func readClusterTables(r ot.Reader, s *dat.Schema) (clusterTables, error) {
	var t clusterTables
	var err error
	if t.slots, err = readRows[jewelSlotRow](r, s, "PassiveJewelSlots"); err != nil {
		return t, err
	}
	if t.jewels, err = readRows[clusterJewelRow](r, s, "PassiveTreeExpansionJewels"); err != nil {
		return t, err
	}
	if t.skills, err = readRows[clusterSkillRow](r, s, "PassiveTreeExpansionSkills"); err != nil {
		return t, err
	}
	if t.notables, err = readRows[clusterNotableRow](r, s, "PassiveTreeExpansionSpecialSkills"); err != nil {
		return t, err
	}
	if t.sizes, err = readRows[sizeRow](r, s, "PassiveTreeExpansionJewelSizes"); err != nil {
		return t, err
	}
	if t.passives, err = readRows[graphIDRow](r, s, "PassiveSkills"); err != nil {
		return t, err
	}
	if t.bases, err = readIDs(r, s, "BaseItemTypes"); err != nil {
		return t, err
	}
	t.tags, err = readIDs(r, s, "Tags")
	return t, err
}

func (t clusterTables) build(nStats int) (dataset.Clusters, error) {
	var c dataset.Clusters
	var err error
	if c.Sockets, err = t.sockets(); err != nil {
		return c, err
	}
	for i, row := range t.jewels {
		j := dataset.ClusterJewel{
			MinNodes: row.MinNodes, MaxNodes: row.MaxNodes, TotalIndices: row.TotalIndices,
			SmallIndices: row.SmallIndices, NotableIndices: row.NotableIndices, SocketIndices: row.SocketIndices,
		}
		if j.Size, err = t.size(row.Size); err == nil {
			j.Base, err = lookup(row.Base, t.bases, "BaseItemTypes")
		}
		if err != nil {
			return c, fmt.Errorf("PassiveTreeExpansionJewels row %d: %w", i, err)
		}
		c.Jewels = append(c.Jewels, j)
	}
	for i, row := range t.skills {
		s, err := t.skill(row)
		if err != nil {
			return c, fmt.Errorf("PassiveTreeExpansionSkills row %d: %w", i, err)
		}
		c.Skills = append(c.Skills, s)
	}
	for i, row := range t.notables {
		n := dataset.ClusterNotable{}
		var stat int
		if n.Passive, err = t.node(row.Passive); err == nil {
			stat, err = rowOf(row.Stat, nStats, "Stats")
		}
		if err != nil {
			return c, fmt.Errorf("PassiveTreeExpansionSpecialSkills row %d: %w", i, err)
		}
		n.Stat = uint32(stat)
		c.Notables = append(c.Notables, n)
	}
	return c, nil
}

func (t clusterTables) skill(row clusterSkillRow) (dataset.ClusterSkill, error) {
	var s dataset.ClusterSkill
	var err error
	if s.Size, err = t.size(row.Size); err != nil {
		return s, err
	}
	if s.Passive, err = t.node(row.Passive); err != nil {
		return s, err
	}
	if _, ok := row.Mastery.Row(); ok {
		if s.Mastery, err = t.node(row.Mastery); err != nil {
			return s, err
		}
	}
	s.Tag, err = lookup(row.Tag, t.tags, "Tags")
	return s, err
}

// sockets returns the jewel slots that take a cluster jewel, sorted by node.
func (t clusterTables) sockets() ([]dataset.ClusterSocket, error) {
	var out []dataset.ClusterSocket
	for i, row := range t.slots {
		if _, ok := row.Size.Row(); !ok {
			continue
		}
		s, err := t.socket(row)
		if err != nil {
			return nil, fmt.Errorf("PassiveJewelSlots row %d: %w", i, err)
		}
		out = append(out, s)
	}
	slices.SortFunc(out, func(a, b dataset.ClusterSocket) int { return cmp.Compare(a.Node, b.Node) })
	for i := 1; i < len(out); i++ {
		if out[i-1].Node == out[i].Node {
			return nil, fmt.Errorf("PassiveJewelSlots: node %d repeated", out[i].Node)
		}
	}
	return out, nil
}

func (t clusterTables) socket(row jewelSlotRow) (dataset.ClusterSocket, error) {
	s := dataset.ClusterSocket{Index: row.Index, StartIndices: row.StartIndices}
	var err error
	if s.Node, err = t.node(row.Slot); err != nil {
		return s, err
	}
	if s.Size, err = t.size(row.Size); err != nil {
		return s, err
	}
	if s.Proxy, err = t.node(row.Proxy); err != nil {
		return s, err
	}
	if _, ok := row.Parent.Row(); ok {
		parent, err := lookup(row.Parent, t.slots, "PassiveJewelSlots")
		if err != nil {
			return s, err
		}
		if s.Parent, err = t.node(parent.Slot); err != nil {
			return s, err
		}
	}
	return s, nil
}

// node returns the graph id of the PassiveSkills row ref references.
func (t clusterTables) node(ref dat.Ref) (uint32, error) {
	p, err := lookup(ref, t.passives, "PassiveSkills")
	return uint32(p.GraphID), err
}

func (t clusterTables) size(ref dat.Ref) (dataset.ClusterSize, error) {
	row, err := lookup(ref, t.sizes, "PassiveTreeExpansionJewelSizes")
	if err != nil {
		return 0, err
	}
	s, ok := clusterSizes[row.Name]
	if !ok {
		return 0, fmt.Errorf("unknown cluster jewel size %q", row.Name)
	}
	return s, nil
}

// checkClusterSockets reports where the sockets of the game tables and the expansionJewel of
// the tree export's nodes disagree. Both say the same of every socket.
func checkClusterSockets(sockets []dataset.ClusterSocket, e *treeexport.Export) error {
	table := map[uint32]dataset.ClusterSocket{}
	for _, s := range sockets {
		table[s.Node] = s
	}
	seen := 0
	for _, id := range slices.Sorted(maps.Keys(e.Nodes)) {
		x := e.Nodes[id].ExpansionJewel
		if x == nil {
			continue
		}
		s, ok := table[e.Nodes[id].Skill]
		if !ok {
			return fmt.Errorf("node %s: cluster socket in the tree export, not in PassiveJewelSlots", id)
		}
		seen++
		parent := "0"
		if s.Parent != 0 {
			parent = strconv.FormatUint(uint64(s.Parent), 10)
		}
		got := treeexport.ExpansionJewel{Size: int(s.Size), Index: int(s.Index), Proxy: strconv.FormatUint(uint64(s.Proxy), 10), Parent: parent}
		want := *x
		want.Parent = cmp.Or(want.Parent, "0")
		if got != want {
			return fmt.Errorf("node %s: PassiveJewelSlots says %+v, the tree export %+v", id, got, want)
		}
	}
	if seen != len(sockets) {
		return fmt.Errorf("%d cluster sockets in PassiveJewelSlots, %d in the tree export", len(sockets), seen)
	}
	return nil
}
