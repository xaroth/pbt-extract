package extract_test

import (
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat/dattest"
	"github.com/xaroth/pbt-extract/extract"
	"github.com/xaroth/pbt-extract/treeexport"
)

// jewelTablesJSON are the jewel tables of the test schema, with the columns the build reads.
const jewelTablesJSON = `
	{"name": "PassiveJewelRadii", "columns": [
		{"name": "ID", "type": "string"}, {"name": "RingOuterRadius", "type": "i32"},
		{"name": "RingInnerRadius", "type": "i32"}, {"name": "Radius", "type": "i32"}]},
	{"name": "PassiveJewelSlots", "columns": [
		{"name": "Slot", "type": "foreignrow"}, {"name": "ClusterJewelSize", "type": "foreignrow"},
		{"name": "ClusterIndex", "type": "i32"}, {"name": "ReplacesSlot", "type": "row"},
		{"name": "ProxySlot", "type": "foreignrow"}, {"name": "StartIndices", "type": "i32", "array": true}]},
	{"name": "PassiveTreeExpansionJewelSizes", "columns": [{"name": "Name", "type": "string"}]},
	{"name": "PassiveTreeExpansionJewels", "columns": [
		{"name": "BaseItemTypesKey", "type": "foreignrow"}, {"name": "PassiveTreeExpansionJewelSizesKey", "type": "foreignrow"},
		{"name": "MinNodes", "type": "i32"}, {"name": "MaxNodes", "type": "i32"},
		{"name": "SmallIndices", "type": "i32", "array": true}, {"name": "NotableIndices", "type": "i32", "array": true},
		{"name": "SocketIndices", "type": "i32", "array": true}, {"name": "TotalIndices", "type": "i32"}]},
	{"name": "PassiveTreeExpansionSkills", "columns": [
		{"name": "PassiveSkillsKey", "type": "foreignrow"}, {"name": "Mastery_PassiveSkillsKey", "type": "foreignrow"},
		{"name": "TagsKey", "type": "foreignrow"}, {"name": "PassiveTreeExpansionJewelSizesKey", "type": "foreignrow"}]},
	{"name": "PassiveTreeExpansionSpecialSkills", "columns": [
		{"name": "PassiveSkillsKey", "type": "foreignrow"}, {"name": "StatsKey", "type": "foreignrow"}]},`

// jewelGame adds jewel tables without rows to g.
func jewelGame(t *testing.T, g fstest.MapFS) {
	t.Helper()
	for _, name := range []string{"PassiveJewelRadii", "PassiveJewelSlots", "PassiveTreeExpansionJewels",
		"PassiveTreeExpansionSkills", "PassiveTreeExpansionSpecialSkills"} {
		g["Data/"+name+".datc64"] = &fstest.MapFile{Data: encode(t, name)}
	}
	g["Data/PassiveTreeExpansionJewelSizes.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveTreeExpansionJewelSizes",
		[]any{"Small"}, []any{"Medium"}, []any{"Large"})}
}

// clusterGame is the test game with a large cluster socket (PassiveSkills row 3, node 4,
// proxy node 5) holding a medium one (row 5, node 6, proxy node 7), beside the tree's plain
// socket, node 2. slots are the PassiveJewelSlots rows.
func clusterGame(t *testing.T, slots ...[]any) fstest.MapFS {
	t.Helper()
	g := game(t)
	empty := func(id uint16) []any { return passive(id, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}) }
	g["Data/PassiveSkills.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkills",
		empty(1), empty(2), empty(3), empty(4), empty(5), empty(6), empty(7), empty(8), empty(9))}
	null := dattest.Null
	g["Data/PassiveJewelRadii.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveJewelRadii",
		[]any{"Small", int32(1100), int32(800), int32(800)}, []any{"Large", int32(1700), int32(1400), int32(1500)})}
	g["Data/PassiveJewelSlots.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveJewelSlots", slots...)}
	g["Data/PassiveTreeExpansionJewels.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveTreeExpansionJewels",
		[]any{0, 2, int32(8), int32(12), []int32{0, 4}, []int32{6}, []int32{4, 8}, int32(12)})}
	g["Data/PassiveTreeExpansionSkills.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveTreeExpansionSkills",
		[]any{7, 8, 2, 2}, []any{7, null, 6, 0})}
	g["Data/PassiveTreeExpansionSpecialSkills.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveTreeExpansionSpecialSkills",
		[]any{8, 4})}
	return g
}

// clusterSlots are the PassiveJewelSlots rows of clusterGame: the plain socket, the medium
// socket on the large one, and the large one.
func clusterSlots() [][]any {
	null := dattest.Null
	return [][]any{
		{1, null, int32(0), null, null, []int32(nil)},
		{5, 1, int32(1), 2, 6, []int32{6}},
		{3, 2, int32(0), null, 4, []int32{0, 4}},
	}
}

// clusterExport is the test export with clusterGame's sockets.
func clusterExport() *treeexport.Export {
	e := export()
	e.Nodes["4"] = treeexport.Node{Skill: 4, IsJewelSocket: true, In: []string{"1"},
		ExpansionJewel: &treeexport.ExpansionJewel{Size: 2, Index: 0, Proxy: "5"}}
	e.Nodes["6"] = treeexport.Node{Skill: 6, IsJewelSocket: true,
		ExpansionJewel: &treeexport.ExpansionJewel{Size: 1, Index: 1, Proxy: "7", Parent: "4"}}
	e.JewelSlots = []uint32{2, 4, 6}
	return e
}

func TestJewels(t *testing.T) {
	t.Parallel()
	got, err := extract.Dataset(input(t, clusterGame(t, clusterSlots()...), clusterExport()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	wantRadii := []dataset.JewelRadius{
		{ID: "Small", Radius: 800, RingInner: 800, RingOuter: 1100},
		{ID: "Large", Radius: 1500, RingInner: 1400, RingOuter: 1700},
	}
	base := itemBases()[0][0].(string)
	wantClusters := dataset.Clusters{
		// sorted by node; the parent is the parent slot's node
		Sockets: []dataset.ClusterSocket{
			{Node: 4, Size: dataset.LargeCluster, Proxy: 5, StartIndices: []int32{0, 4}},
			{Node: 6, Size: dataset.MediumCluster, Index: 1, Proxy: 7, Parent: 4, StartIndices: []int32{6}},
		},
		Jewels: []dataset.ClusterJewel{{
			Size: dataset.LargeCluster, Base: base, MinNodes: 8, MaxNodes: 12,
			SmallIndices: []int32{0, 4}, NotableIndices: []int32{6}, SocketIndices: []int32{4, 8}, TotalIndices: 12,
		}},
		Skills: []dataset.ClusterSkill{
			{Size: dataset.LargeCluster, Passive: 8, Mastery: 9, Tag: itemTags[2]},
			{Size: dataset.SmallCluster, Passive: 8, Tag: itemTags[6]},
		},
		Notables: []dataset.ClusterNotable{{Passive: 9, Stat: 4}},
	}
	for _, tree := range got.Trees {
		if diff := cmp.Diff(wantRadii, tree.JewelRadii); diff != "" {
			t.Errorf("tree %s radii mismatch (-want +got):\n%s", tree.Variant, diff)
		}
		if diff := cmp.Diff(wantClusters, tree.Clusters, cmpopts.EquateEmpty()); diff != "" {
			t.Errorf("tree %s clusters mismatch (-want +got):\n%s", tree.Variant, diff)
		}
	}
}

func TestJewelsErrors(t *testing.T) {
	t.Parallel()
	null := dattest.Null
	testCases := []struct {
		name   string
		slots  [][]any
		sizes  []string
		export func(*treeexport.Export)
	}{
		{name: "export has another proxy", export: func(e *treeexport.Export) { e.Nodes["6"].ExpansionJewel.Proxy = "5" }},
		{name: "export has another parent", export: func(e *treeexport.Export) { e.Nodes["6"].ExpansionJewel.Parent = "" }},
		{name: "export has another size", export: func(e *treeexport.Export) { e.Nodes["4"].ExpansionJewel.Size = 1 }},
		{name: "socket not in the export", export: func(e *treeexport.Export) {
			n := e.Nodes["6"]
			n.ExpansionJewel = nil
			e.Nodes["6"] = n
		}},
		{name: "socket not in the tables", slots: clusterSlots()[:1]},
		{name: "unknown size", sizes: []string{"Small", "Medium", "Huge"}},
		{name: "socket without proxy", slots: [][]any{{3, 2, int32(0), null, null, []int32(nil)}}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			slots := testCase.slots
			if slots == nil {
				slots = clusterSlots()
			}
			g := clusterGame(t, slots...)
			if testCase.sizes != nil {
				var rows [][]any
				for _, s := range testCase.sizes {
					rows = append(rows, []any{s})
				}
				g["Data/PassiveTreeExpansionJewelSizes.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveTreeExpansionJewelSizes", rows...)}
			}
			e := clusterExport()
			if testCase.export != nil {
				testCase.export(e)
			}
			if _, err := extract.Dataset(input(t, g, e)); err == nil {
				t.Errorf("Dataset: got nil error, want an error")
			}
		})
	}
}
