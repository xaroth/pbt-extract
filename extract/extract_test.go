package extract_test

import (
	"encoding/binary"
	"maps"
	"testing"
	"testing/fstest"
	"unicode/utf16"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/dat/dattest"
	"github.com/xaroth/pbt-extract/extract"
	"github.com/xaroth/pbt-extract/treeexport"
)

const schemaJSON = `{"version": 7, "tables": [` + skillTablesJSON + monsterTablesJSON + overrideTablesJSON + `
	{"name": "Stats", "columns": [{"name": "Id", "type": "string"}]},
	{"name": "Characters", "columns": [
		{"name": "Id", "type": "string"}, {"name": "Name", "type": "string"}, {"name": "IntegerId", "type": "i32"},
		{"name": "BaseStrength", "type": "i32"}, {"name": "BaseDexterity", "type": "i32"},
		{"name": "BaseIntelligence", "type": "i32"}, {"name": "BaseMaxLife", "type": "i32"},
		{"name": "BaseMaxMana", "type": "i32"}, {"name": "MinDamage", "type": "i32"}, {"name": "MaxDamage", "type": "i32"},
		{"name": "WeaponSpeed", "type": "i32"}]},
	{"name": "PassiveSkills", "columns": [
		{"name": "PassiveSkillGraphId", "type": "u16"},
		{"name": "Stats", "type": "foreignrow", "array": true},
		{"name": "Stat1Value", "type": "i32"}, {"name": "Stat2Value", "type": "i32"},
		{"name": "Stat3Value", "type": "i32"}, {"name": "Stat4Value", "type": "i32"},
		{"name": "Stat5Value", "type": "i32"},
		{"name": "PassiveSkillBuffs", "type": "foreignrow", "array": true},
		{"name": "SkillPointsGranted", "type": "i32"},
		{"name": "HasHardmodeAlternate", "type": "bool"},
		{"name": "StatsHardmode", "type": "foreignrow", "array": true},
		{"name": "Stat1ValueHardmode", "type": "i32"}, {"name": "Stat2ValueHardmode", "type": "i32"},
		{"name": "Stat3ValueHardmode", "type": "i32"}, {"name": "Stat4ValueHardmode", "type": "i32"},
		{"name": "Stat5ValueHardmode", "type": "i32"},
		{"name": "BuffsHardmode", "type": "foreignrow", "array": true},
		{"name": "SkillPointsGrantedHardmode", "type": "i32"},
		{"name": "MasteryGroup", "type": "foreignrow"},
		{"name": "GrantedEffectsPerLevel", "type": "foreignrow"},
		{"name": "GrantedEffectsPerLevelHardmode", "type": "foreignrow"},
		{"name": "Name", "type": "string"}, {"name": "Icon_DDSFile", "type": "string"}]},
	{"name": "PassiveSkillMasteryGroups", "columns": [{"name": "MasteryCountStat", "type": "foreignrow"}]},
	{"name": "PassiveSkillMasteryEffects", "columns": [
		{"name": "HASH16", "type": "u16"},
		{"name": "Stats", "type": "foreignrow", "array": true},
		{"name": "Stat1Value", "type": "i32"}, {"name": "Stat2Value", "type": "i32"},
		{"name": "Stat3Value", "type": "i32"}]},
	{"name": "BuffTemplates", "columns": [
		{"name": "BuffDefinitionsKey", "type": "foreignrow"},
		{"name": "Buff_StatValues", "type": "i32", "array": true}]},
	{"name": "QuestStaticRewards", "columns": [
		{"name": "QuestFlag", "type": "foreignrow"}, {"name": "PassivePoints", "type": "i32"},
		{"name": "StatsKeys", "type": "foreignrow", "array": true}, {"name": "StatValues", "type": "i32", "array": true},
		{"name": "QuestKey", "type": "foreignrow"}, {"name": "AscendancyPoints", "type": "i32"},
		{"name": "ClientStringsKey", "type": "foreignrow"}, {"name": "RefundPoints", "type": "i32"},
		{"name": "StatValuesHardmode", "type": "i32", "array": true}, {"name": "ClientStringHardmode", "type": "foreignrow"},
		{"name": "LeagueReward", "type": "bool"}]},
	{"name": "QuestFlags", "columns": [{"name": "Id", "type": "string"}]},
	{"name": "Quest", "columns": [{"name": "Id", "type": "string"}, {"name": "Act", "type": "i32"}]},` + itemTables + `,` + buffTables + `
]}`

// encode encodes rows as Data/<table>.datc64 of the test schema.
func encode(t *testing.T, table string, rows ...[]any) []byte {
	t.Helper()
	return encodeAs(t, schemaJSON, table, rows...)
}

// encodeAs encodes rows as Data/<table>.datc64 of the schema in the JSON s.
func encodeAs(t *testing.T, s, table string, rows ...[]any) []byte {
	t.Helper()
	sc, err := dat.ParseSchema([]byte(s))
	if err != nil {
		t.Fatal(err)
	}
	def, ok := sc.Table(table)
	if !ok {
		t.Fatalf("table %s not in schema", table)
	}
	return dattest.Encode(t, def, rows...)
}

func utf16le(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return b
}

func schema(t *testing.T) *dat.Schema {
	t.Helper()
	s, err := dat.ParseSchema([]byte(schemaJSON))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// questReward is one QuestStaticRewards row: flag, passive points, stats and values, quest,
// ascendancy points, refund points and the league mechanic flag.
func questReward(flag int, points int32, stats []int, vals []int32, quest int, asc, refund int32, league bool) []any {
	return []any{flag, points, stats, vals, quest, asc, dattest.Null, refund, vals, dattest.Null, league}
}

// passive is one PassiveSkills row: graph id, stats and values, buffs, points, then the
// hardmode flag, stats and values. Hardmode buffs and points, mastery group, granted skills,
// name and icon are empty.
func passive(id uint16, stats []int, vals [5]int32, buffs []int, points int32, hard bool, hStats []int, hVals [5]int32) []any {
	row := []any{id, stats}
	for _, v := range vals {
		row = append(row, v)
	}
	row = append(row, buffs, points, hard, hStats)
	for _, v := range hVals {
		row = append(row, v)
	}
	return append(row, []int(nil), int32(0), dattest.Null, dattest.Null, dattest.Null, "", "")
}

// granting sets the GrantedEffectsPerLevel rows a passive row grants, outside and in Ruthless.
func granting(row []any, grant, hardmode int) []any {
	row[len(row)-4], row[len(row)-3] = grant, hardmode
	return row
}

// masteryPassive is a PassiveSkills row of a mastery node in mastery group 0.
func masteryPassive(id uint16) []any {
	row := passive(id, nil, [5]int32{}, nil, 0, false, nil, [5]int32{})
	row[len(row)-5] = 0
	return row
}

const sdDir = "Metadata/StatDescriptions/"

func game(t *testing.T) fstest.MapFS {
	t.Helper()
	g := fstest.MapFS{
		"Data/Stats.datc64": {Data: encode(t, "Stats",
			[]any{"level"}, []any{"life_per_level"}, []any{"base_speed"}, []any{"base_strength"}, []any{"aura_damage_+%"},
		)},
		"Data/Characters.datc64": {Data: encode(t, "Characters",
			[]any{"Metadata/Characters/Str/Str", "Marauder", int32(0), int32(32), int32(14), int32(14), int32(38), int32(34), int32(2), int32(8), int32(833)},
		)},
		"Data/PassiveSkills.datc64": {Data: encode(t, "PassiveSkills",
			passive(1, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
			granting(passive(2, []int{3, 1}, [5]int32{10, 5}, []int{0, 1}, 1, true, []int{3}, [5]int32{20}), 1, 0),
			// grants a Ruthless skill of its own without HasHardmodeAlternate
			granting(masteryPassive(3), dattest.Null, 2),
		)},
		"Data/PassiveSkillMasteryGroups.datc64": {Data: encode(t, "PassiveSkillMasteryGroups", []any{2})},
		"Data/PassiveSkillMasteryEffects.datc64": {Data: encode(t, "PassiveSkillMasteryEffects",
			[]any{uint16(7), []int{1}, int32(3), int32(0), int32(0)},
		)},
		"Data/BuffTemplates.datc64": {Data: encode(t, "BuffTemplates",
			[]any{0, []int32{-5, 9}},
			[]any{1, []int32{1}},
		)},
		"Data/QuestStaticRewards.datc64": {Data: encode(t, "QuestStaticRewards",
			questReward(0, 1, nil, nil, 0, 0, 0, false),
			questReward(1, 0, []int{4, 3}, []int32{-30, -20}, 1, 0, 2, false),
			questReward(2, 0, nil, nil, dattest.Null, 2, 0, false),
			questReward(3, 3, nil, nil, dattest.Null, 0, 3, true),
		)},
		"Data/QuestFlags.datc64": {Data: encode(t, "QuestFlags",
			[]any{"a2q7-QuestRewardTaken"}, []any{"a5q6-CompletedBoss"}, []any{"LabyrinthComplete1"}, []any{"EndlessDelvePassiveRewards1"},
		)},
		"Data/Quest.datc64":                                {Data: encode(t, "Quest", []any{"a2q7", int32(2)}, []any{"a5q6", int32(5)})},
		"Metadata/Characters/Character.ot":                 {Data: utf16le("extends \"nothing\"\nStats\n{\n\tlife_per_level = 12\n\tlevel = 1\n}\n")},
		"Metadata/Characters/Str/Str.ot":                   {Data: utf16le("extends \"Metadata/Characters/Character\"\nStats\n{\n}\n")},
		sdDir + "passive_skill_aura_stat_descriptions.txt": {Data: utf16le("include \"" + sdDir + "passive_skill_stat_descriptions.txt\"\n")},
		sdDir + "passive_skill_stat_descriptions.txt":      {Data: utf16le("include \"" + sdDir + "stat_descriptions.txt\"\n")},
		sdDir + "stat_descriptions.txt":                    {Data: utf16le("description\n\t1 base_strength\n\t1\n\t\t# \"{0:+d} to Strength\"\n")},
	}
	for _, f := range extract.SkillDescriptions() {
		g[f] = &fstest.MapFile{Data: utf16le("include \"" + sdDir + "stat_descriptions.txt\"\n")}
	}
	itemGame(t, g)
	buffGame(t, g)
	overrideGame(t, g)
	maps.Copy(g, skillFiles(t, skillRows()))
	maps.Copy(g, monsterFiles(t, monsterRows()))
	return g
}

func export() *treeexport.Export {
	start := 0
	e := &treeexport.Export{
		Classes: []treeexport.Class{{
			Name: "Marauder", BaseStrength: 32, BaseDexterity: 14, BaseIntelligence: 14,
			Ascendancies: []treeexport.Ascendancy{{ID: "Juggernaut", Name: "Juggernaut"}},
		}},
		AlternateAscendancies: []treeexport.Ascendancy{{ID: "Bloodline", Name: "Bloodline"}},
		Nodes: map[string]treeexport.Node{
			"1": {Skill: 1, Name: "MARAUDER", ClassStartIndex: &start, Out: []string{"2"}, Group: 2, Icon: "Art/start.png"},
			"2": {
				Skill: 2, Name: "Might", IsNotable: true, AscendancyName: "Juggernaut", In: []string{"1"}, Out: []string{"1"},
				Group: 10, Orbit: 2, OrbitIndex: 2, Icon: "Art/might.png",
			},
			"3": {
				Skill: 3, Name: "Mastery", IsMastery: true, MasteryEffects: []treeexport.MasteryEffect{{Effect: 7}},
				Icon: "Art/mastery.png", InactiveIcon: "Art/inactive.png", ActiveIcon: "Art/active.png", ActiveEffectImage: "Art/effect.png",
			},
		},
		JewelSlots: []uint32{2},
		Groups: map[string]treeexport.Group{
			"2":  {X: -2824.29, Y: 1642.43, Orbits: []uint8{}},
			"10": {X: 100, Y: -50, Orbits: []uint8{0, 2}, Background: &treeexport.Background{Image: "PSGroupBackground3", IsHalfImage: true}, IsProxy: true},
		},
		ExtraImages: map[string]treeexport.ExtraImage{"0": {X: -4560.69, Y: 292.425, Image: "Art/2DArt/BaseClassIllustrations/Str.png"}},
		MinX:        -14159, MinY: -10689, MaxX: 12430, MaxY: 10900,
	}
	e.Constants.SkillsPerOrbit = []uint32{1, 6, 16, 16, 40, 72}
	e.Constants.OrbitRadii = []float64{0, 82, 162, 335, 493, 662}
	e.Points.Total, e.Points.Ascendancy = 123, 8
	return e
}

func input(t *testing.T, g fstest.MapFS, e *treeexport.Export) extract.Input {
	t.Helper()
	return extract.Input{
		Game:   g,
		Schema: schema(t),
		Trees: []extract.Variant{
			{Name: "standard", Export: e},
			{Name: "ruthless", Export: e, Hardmode: true},
		},
		Constants: []dataset.Constant{{ID: "life_per_strength", Value: 1, Per: 2}},
	}
}

func TestDataset(t *testing.T) {
	t.Parallel()
	got, err := extract.Dataset(input(t, game(t), export()))
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	baseTree := func(variant string, n2 dataset.Node) dataset.Tree {
		return dataset.Tree{
			Variant: variant, TotalPoints: 123, AscendancyPoints: 8,
			Classes: []dataset.Class{{
				Character: "Metadata/Characters/Str/Str", BaseStrength: 32, BaseDexterity: 14, BaseIntelligence: 14, StartNode: 1,
				Ascendancies: []dataset.Ascendancy{{ID: "Juggernaut"}},
			}},
			AlternateAscendancies: []dataset.Ascendancy{{ID: "Bloodline"}},
			Nodes: []dataset.Node{
				{ID: 1, Kind: dataset.ClassStart, Neighbours: []uint32{2}},
				n2,
				// the mastery's group counts it in stat 2
				{ID: 3, Kind: dataset.Mastery, Stats: []dataset.StatValue{{Stat: 2, Value: 1}},
					MasteryEffects: []dataset.MasteryEffect{{ID: 7, Stats: []dataset.StatValue{{Stat: 1, Value: 3}}}}},
			},
			JewelSlots: []uint32{2},
		}
	}
	might := dataset.Node{ID: 2, Kind: dataset.Notable, Ascendancy: "Juggernaut", Neighbours: []uint32{1}}
	standard, ruthless := might, might
	standard.Stats = []dataset.StatValue{{Stat: 1, Value: 5}, {Stat: 3, Value: 10}}
	standard.Buffs = [][]dataset.StatValue{{{Stat: 4, Value: -5}}}
	standard.SkillPoints = 1
	standard.GrantedSkill = &dataset.GrantedSkill{Effect: "Fireball", Level: 1}
	ruthless.Stats = []dataset.StatValue{{Stat: 3, Value: 20}}
	ruthless.GrantedSkill = &dataset.GrantedSkill{Effect: "Fireball", Level: 2}
	ruthlessTree := baseTree("ruthless", ruthless)
	ruthlessTree.Nodes[2].GrantedSkill = &dataset.GrantedSkill{Effect: "SupportEcho", Level: 1}

	want := &dataset.Dataset{
		Stats: []dataset.Stat{{ID: "level"}, {ID: "life_per_level"}, {ID: "base_speed"}, {ID: "base_strength"}, {ID: "aura_damage_+%"}},
		Characters: []dataset.Character{{
			ID:           "Metadata/Characters/Str/Str",
			BaseStrength: 32, BaseDexterity: 14, BaseIntelligence: 14, BaseMaxLife: 38, BaseMaxMana: 34,
			Stats:         []dataset.StatValue{{Stat: 0, Value: 1}, {Stat: 1, Value: 12}},
			UnarmedDamage: dataset.ValueRange{Min: 2, Max: 8}, UnarmedAttackTime: 833,
		}},
		Constants: []dataset.Constant{{ID: "life_per_strength", Value: 1, Per: 2}},
		// the league mechanic row is left out; reward stats are sorted
		Quests: []dataset.Quest{
			{Quest: "a2q7", Flag: "a2q7-QuestRewardTaken", Act: 2, PassivePoints: 1},
			{Quest: "a5q6", Flag: "a5q6-CompletedBoss", Act: 5, RefundPoints: 2, Stats: []dataset.StatValue{{Stat: 3, Value: -20}, {Stat: 4, Value: -30}}},
			{Flag: "LabyrinthComplete1", AscendancyPoints: 2},
		},
		Trees: []dataset.Tree{baseTree("standard", standard), ruthlessTree},
		Texts: []dataset.Text{{
			Language:       "en",
			NodeNames:      []dataset.NodeName{{ID: 1, Name: "MARAUDER"}, {ID: 2, Name: "Might"}, {ID: 3, Name: "Mastery"}},
			CharacterNames: []dataset.Name{{ID: "Metadata/Characters/Str/Str", Name: "Marauder"}},
			AscendancyNames: []dataset.AscendancyName{
				{Variant: "standard", ID: "Juggernaut", Name: "Juggernaut"}, {Variant: "standard", ID: "Bloodline", Name: "Bloodline"},
				{Variant: "ruthless", ID: "Juggernaut", Name: "Juggernaut"}, {Variant: "ruthless", ID: "Bloodline", Name: "Bloodline"},
			},
		}},
	}
	// TestLayouts, TestItems, TestSkills, TestBuffs and TestMonsters check their groups.
	ignore := cmp.Options{
		cmpopts.IgnoreFields(dataset.Text{}, "Descriptions", "ItemClassNames", "BaseNames", "ModNames", "GemNames", "SkillNames", "BuffNames", "MonsterNames"),
		cmpopts.IgnoreFields(dataset.Dataset{}, "Items", "Mods", "Skills", "Buffs", "Monsters", "Layouts"),
	}
	if diff := cmp.Diff(want, got, ignore, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Dataset mismatch (-want +got):\n%s", diff)
	}
	var scopes []string
	for _, s := range got.Texts[0].Descriptions {
		scopes = append(scopes, s.Name)
	}
	// stat_descriptions.txt, which every skill file includes, is loaded once
	wantScopes := append([]string{extract.DescriptionsRoot, sdDir + "passive_skill_stat_descriptions.txt", sdDir + "stat_descriptions.txt"},
		extract.SkillDescriptions()...)
	if diff := cmp.Diff(wantScopes, scopes); diff != "" {
		t.Errorf("description scopes mismatch (-want +got):\n%s", diff)
	}
}

func TestDatasetProxyWithoutRow(t *testing.T) {
	t.Parallel()
	e := export()
	e.Nodes["9"] = treeexport.Node{Skill: 9, Name: "Growth", IsNotable: true, IsProxy: true, In: []string{"2"}, Group: 10}
	in := input(t, game(t), e)
	in.Trees = in.Trees[:1]
	var warnings []string
	in.Warn = func(msg string) { warnings = append(warnings, msg) }
	got, err := extract.Dataset(in)
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	want := []string{"tree standard node 9: proxy node has no PassiveSkills row; carried without stats"}
	if diff := cmp.Diff(want, warnings); diff != "" {
		t.Errorf("warnings mismatch (-want +got):\n%s", diff)
	}
	nodes := got.Trees[0].Nodes
	wantNode := dataset.Node{ID: 9, Kind: dataset.Proxy, Neighbours: []uint32{2}}
	if diff := cmp.Diff(wantNode, nodes[len(nodes)-1], cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("proxy node mismatch (-want +got):\n%s", diff)
	}
}

func TestDatasetErrors(t *testing.T) {
	t.Parallel()
	const statsI32 = `{"version": 7, "tables": [{"name": "Stats", "columns": [{"name": "Id", "type": "i32"}]}]}`
	testCases := []struct {
		name   string
		game   func(*testing.T, fstest.MapFS)
		export func(*treeexport.Export)
		input  func(*extract.Input)
		schema string
	}{
		{name: "unknown stat", game: func(_ *testing.T, g fstest.MapFS) {
			g["Metadata/Characters/Str/Str.ot"] = &fstest.MapFile{Data: utf16le("extends \"nothing\"\nStats\n{\n\tnew_stat = 1\n}\n")}
		}},
		{name: "stat overflows int32", game: func(_ *testing.T, g fstest.MapFS) {
			g["Metadata/Characters/Str/Str.ot"] = &fstest.MapFile{Data: utf16le("extends \"nothing\"\nStats\n{\n\tlevel = 4294967296\n}\n")}
		}},
		{name: "missing ot", game: func(_ *testing.T, g fstest.MapFS) { delete(g, "Metadata/Characters/Str/Str.ot") }},
		{name: "missing table file", game: func(_ *testing.T, g fstest.MapFS) { delete(g, "Data/Characters.datc64") }},
		{name: "missing descriptions", game: func(_ *testing.T, g fstest.MapFS) { delete(g, sdDir+"stat_descriptions.txt") }},
		{name: "missing skill descriptions", game: func(_ *testing.T, g fstest.MapFS) { delete(g, extract.SkillDescriptions()[0]) }},
		{name: "row size mismatch", game: func(t *testing.T, g fstest.MapFS) {
			wider := `{"version": 7, "tables": [{"name": "Stats", "columns": [{"name": "Id", "type": "string"}, {"name": "X", "type": "i32"}]}]}`
			g["Data/Stats.datc64"] = &fstest.MapFile{Data: encodeAs(t, wider, "Stats", []any{"level", int32(1)})}
		}},
		{name: "table not in schema", schema: `{"version": 7, "tables": []}`},
		{name: "column not in schema", schema: `{"version": 7, "tables": [{"name": "Stats", "columns": [{"name": "Other", "type": "string"}]}]}`},
		{name: "column of wrong type", schema: statsI32, game: func(t *testing.T, g fstest.MapFS) {
			g["Data/Stats.datc64"] = &fstest.MapFile{Data: encodeAs(t, statsI32, "Stats", []any{int32(1)})}
		}},
		{name: "missing tree table", game: func(_ *testing.T, g fstest.MapFS) { delete(g, "Data/BuffDefinitions.datc64") }},
		{name: "mastery count stat out of range", game: func(t *testing.T, g fstest.MapFS) {
			g["Data/PassiveSkillMasteryGroups.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkillMasteryGroups", []any{99})}
		}},
		{name: "missing mastery groups", game: func(_ *testing.T, g fstest.MapFS) { delete(g, "Data/PassiveSkillMasteryGroups.datc64") }},
		{name: "missing quest table", game: func(_ *testing.T, g fstest.MapFS) { delete(g, "Data/QuestStaticRewards.datc64") }},
		{name: "quest stats without values", game: func(t *testing.T, g fstest.MapFS) {
			g["Data/QuestStaticRewards.datc64"] = &fstest.MapFile{Data: encode(t, "QuestStaticRewards", questReward(1, 0, []int{4}, nil, 1, 0, 0, false))}
		}},
		{name: "quest stat out of range", game: func(t *testing.T, g fstest.MapFS) {
			g["Data/QuestStaticRewards.datc64"] = &fstest.MapFile{Data: encode(t, "QuestStaticRewards", questReward(1, 0, []int{9}, []int32{1}, 1, 0, 0, false))}
		}},
		{name: "repeated graph id", game: func(t *testing.T, g fstest.MapFS) {
			g["Data/PassiveSkills.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkills",
				passive(1, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
				passive(1, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
			)}
		}},
		{name: "repeated node stat", game: func(t *testing.T, g fstest.MapFS) {
			g["Data/PassiveSkills.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkills",
				passive(1, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
				passive(2, []int{3, 3}, [5]int32{1, 1}, nil, 0, false, nil, [5]int32{}),
				passive(3, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
			)}
		}},
		{name: "more stats than value columns", game: func(t *testing.T, g fstest.MapFS) {
			g["Data/PassiveSkills.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkills",
				passive(1, []int{0, 1, 2, 3, 4, 0}, [5]int32{}, nil, 0, false, nil, [5]int32{}),
				passive(2, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
				passive(3, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
			)}
		}},
		{name: "node stat out of range", game: func(t *testing.T, g fstest.MapFS) {
			g["Data/PassiveSkills.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkills",
				passive(1, []int{9}, [5]int32{1}, nil, 0, false, nil, [5]int32{}),
				passive(2, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
				passive(3, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
			)}
		}},
		{name: "granted skill out of range", game: func(t *testing.T, g fstest.MapFS) {
			g["Data/PassiveSkills.datc64"] = &fstest.MapFile{Data: encode(t, "PassiveSkills",
				granting(passive(1, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}), 9, dattest.Null),
				passive(2, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
				passive(3, nil, [5]int32{}, nil, 0, false, nil, [5]int32{}),
			)}
		}},
		{name: "buff with fewer values than stats", game: func(t *testing.T, g fstest.MapFS) {
			g["Data/BuffDefinitions.datc64"] = &fstest.MapFile{Data: encodeBuffDefs(t, buffDef{id: "a", stats: []int{4, 4, 4}}, buffDef{id: "b"})}
		}},
		{name: "buff without definition", game: func(t *testing.T, g fstest.MapFS) {
			g["Data/BuffTemplates.datc64"] = &fstest.MapFile{Data: encode(t, "BuffTemplates", []any{-1, []int32{1}}, []any{1, []int32{1}})}
		}},
		{name: "buff stat out of range", game: func(t *testing.T, g fstest.MapFS) {
			g["Data/BuffDefinitions.datc64"] = &fstest.MapFile{Data: encodeBuffDefs(t, buffDef{id: "a", stats: []int{9}}, buffDef{id: "b"})}
		}},
		{name: "non-proxy node not in PassiveSkills", export: func(e *treeexport.Export) { e.Nodes["9"] = treeexport.Node{Skill: 9} }},
		{name: "unknown mastery effect", export: func(e *treeexport.Export) {
			n := e.Nodes["3"]
			n.MasteryEffects = []treeexport.MasteryEffect{{Effect: 8}}
			e.Nodes["3"] = n
		}},
		{name: "edge to unknown node", export: func(e *treeexport.Export) {
			n := e.Nodes["2"]
			n.Out = []string{"9"}
			e.Nodes["2"] = n
		}},
		{name: "edge not a number", export: func(e *treeexport.Export) {
			n := e.Nodes["2"]
			n.Out = []string{"x"}
			e.Nodes["2"] = n
		}},
		{name: "class without start node", export: func(e *treeexport.Export) {
			e.Classes = append(e.Classes, treeexport.Class{Name: "Ranger"})
		}},
		{name: "node renamed in another variant", input: func(in *extract.Input) {
			other := *in.Trees[0].Export
			other.Nodes = maps.Clone(other.Nodes)
			n := other.Nodes["2"]
			n.Name = "Other"
			other.Nodes["2"] = n
			in.Trees[1].Export = &other
		}},
		{name: "class start node twice", export: func(e *treeexport.Export) {
			n := e.Nodes["3"]
			n.ClassStartIndex = e.Nodes["1"].ClassStartIndex
			e.Nodes["3"] = n
		}},
		{name: "class without Characters row", game: func(t *testing.T, g fstest.MapFS) {
			g["Data/Characters.datc64"] = &fstest.MapFile{Data: encode(t, "Characters",
				[]any{"Metadata/Characters/Str/Str", "Marauder", int32(5), int32(32), int32(14), int32(14), int32(38), int32(34), int32(2), int32(8), int32(833)},
			)}
		}},
		{name: "repeated IntegerId", game: func(t *testing.T, g fstest.MapFS) {
			g["Data/Characters.datc64"] = &fstest.MapFile{Data: encode(t, "Characters",
				[]any{"Metadata/Characters/Str/Str", "Marauder", int32(0), int32(32), int32(14), int32(14), int32(38), int32(34), int32(2), int32(8), int32(833)},
				[]any{"Metadata/Characters/Str/Str", "Marauder", int32(0), int32(32), int32(14), int32(14), int32(38), int32(34), int32(2), int32(8), int32(833)},
			)}
		}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g, e := game(t), export()
			if testCase.game != nil {
				testCase.game(t, g)
			}
			if testCase.export != nil {
				testCase.export(e)
			}
			in := input(t, g, e)
			if testCase.input != nil {
				testCase.input(&in)
			}
			if testCase.schema != "" {
				var err error
				if in.Schema, err = dat.ParseSchema([]byte(testCase.schema)); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := extract.Dataset(in); err == nil {
				t.Errorf("Dataset: got nil error, want an error")
			}
		})
	}
}
