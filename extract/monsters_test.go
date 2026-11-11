package extract_test

import (
	"maps"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat/dattest"
	"github.com/xaroth/pbt-extract/extract"
)

// monsterTablesJSON are the monster tables of the test schema, with the columns the build
// reads. Mods, Tags, ItemClasses and GrantedEffects are those of the item and skill tables.
const monsterTablesJSON = `
	{"name": "MonsterResistances", "columns": [
		{"name": "Id", "type": "string"},
		{"name": "FireNormal", "type": "i32"}, {"name": "ColdNormal", "type": "i32"},
		{"name": "LightningNormal", "type": "i32"}, {"name": "ChaosNormal", "type": "i32"},
		{"name": "FireCruel", "type": "i32"}, {"name": "ColdCruel", "type": "i32"},
		{"name": "LightningCruel", "type": "i32"}, {"name": "ChaosCruel", "type": "i32"},
		{"name": "FireMerciless", "type": "i32"}, {"name": "ColdMerciless", "type": "i32"},
		{"name": "LightningMerciless", "type": "i32"}, {"name": "ChaosMerciless", "type": "i32"}]},
	{"name": "MonsterTypes", "columns": [
		{"name": "Id", "type": "string"}, {"name": "Accuracy", "type": "i32"}, {"name": "IsPlayerMinion", "type": "bool"},
		{"name": "Armour", "type": "i32"}, {"name": "Evasion", "type": "i32"}, {"name": "EnergyShield", "type": "i32"},
		{"name": "DamageSpread", "type": "i32"}, {"name": "Resistances", "type": "foreignrow"},
		{"name": "AltLife1", "type": "bool"}, {"name": "AltLife2", "type": "bool"},
		{"name": "BaseDamageIgnoresAttackSpeed", "type": "bool"}]},
	{"name": "MonsterVarieties", "columns": [
		{"name": "Id", "type": "string"}, {"name": "MonsterTypesKey", "type": "foreignrow"},
		{"name": "ModsKeys", "type": "foreignrow", "array": true},
		{"name": "TagsKeys", "type": "foreignrow", "array": true},
		{"name": "GrantedEffectsKeys", "type": "foreignrow", "array": true},
		{"name": "ModsKeys2", "type": "foreignrow", "array": true},
		{"name": "Name", "type": "string"}, {"name": "DamageMultiplier", "type": "i32"},
		{"name": "LifeMultiplier", "type": "i32"}, {"name": "AttackSpeed", "type": "i32"},
		{"name": "MainHand_ItemClassesKey", "type": "foreignrow"}, {"name": "OffHand_ItemClassesKey", "type": "foreignrow"},
		{"name": "Special_ModsKeys", "type": "foreignrow", "array": true},
		{"name": "Part1_ModsKeys", "type": "foreignrow", "array": true},
		{"name": "Part2_ModsKeys", "type": "foreignrow", "array": true},
		{"name": "Endgame_ModsKeys", "type": "foreignrow", "array": true},
		{"name": "AilmentThreshold", "type": "i32"}]},
	{"name": "DefaultMonsterStats", "columns": [
		{"name": "DisplayLevel", "type": "string"}, {"name": "Damage", "type": "f32"},
		{"name": "Evasion", "type": "i32"}, {"name": "Accuracy", "type": "i32"}, {"name": "MonsterLife", "type": "i32"},
		{"name": "MinionLife", "type": "i32"}, {"name": "Armour", "type": "i32"}, {"name": "Difficulty", "type": "i32"},
		{"name": "MinionDamage", "type": "f32"}, {"name": "AltLife1", "type": "i32"}, {"name": "AltDamage1", "type": "f32"},
		{"name": "AltDamage2", "type": "f32"}, {"name": "AltLife2", "type": "i32"}, {"name": "EvasiveEvasion", "type": "i32"},
		{"name": "AilmentThreshold", "type": "i32"}, {"name": "MonsterPhysConversionMulti", "type": "i32"}]},
	{"name": "DisplayMinionMonsterType", "columns": [{"name": "Id", "type": "i32"}, {"name": "MonsterVarietiesKey", "type": "foreignrow"}]},
	{"name": "SkillTotemVariations", "columns": [
		{"name": "SkillTotemsKey", "type": "enumrow"}, {"name": "TotemSkinId", "type": "i32"},
		{"name": "MonsterVarietiesKey", "type": "foreignrow"}]},
	{"name": "SpectreOverrides", "columns": [{"name": "Monster", "type": "foreignrow"}, {"name": "Spectre", "type": "foreignrow"}]},
	{"name": "MinionType", "columns": [
		{"name": "Id", "type": "string"}, {"name": "LimitStat", "type": "foreignrow"}, {"name": "ActiveCountStat", "type": "foreignrow"}]},`

// Rows of Mods: modItem is an item mod and modMonster a monster mod of the items fixture; the
// monster fixture adds the monster mods of monsterMods after it, one per mod list of a variety.
const (
	modItem    = 0
	modMonster = 1
)

// monsterMods are the monster mods the monster fixture adds to the items fixture's Mods.
var monsterMods = []string{"MonsterMods2", "MonsterSpecial", "MonsterPart1", "MonsterPart2", "MonsterEndgame"}

// monsterMod returns the Mods row of monsterMods[i].
func monsterMod(i int) int { return len(itemMods()) + i }

// monsterVariety is a MonsterVarieties row with the given mods and no other lists.
func monsterVariety(id, name string, typ int, mods ...int) []any {
	return []any{id, typ, mods, []int(nil), []int(nil), []int(nil), name, int32(100), int32(100), int32(1000),
		dattest.Null, dattest.Null, []int(nil), []int(nil), []int(nil), []int(nil), int32(100)}
}

// monsterLevel is a DefaultMonsterStats row with life and damage only.
func monsterLevel(level string, difficulty, life int32, damage float32) []any {
	return []any{level, damage, int32(0), int32(0), life, int32(0), int32(0), difficulty,
		float32(0), int32(0), float32(0), float32(0), int32(0), int32(0), int32(0), int32(0)}
}

// monsterRows are the rows of the monster tables of the test game. Stats are those of game.
func monsterRows() map[string][][]any {
	return map[string][][]any{
		"MonsterResistances": {
			{"Strong", int32(1), int32(2), int32(3), int32(4), int32(5), int32(6), int32(7), int32(8),
				int32(9), int32(10), int32(11), int32(12)},
		},
		"MonsterTypes": {
			{"Zombie", int32(10), true, int32(20), int32(30), int32(40), int32(50), 0, true, false, true},
			{"Totem", int32(0), true, int32(0), int32(0), int32(0), int32(0), dattest.Null, false, true, false},
		},
		"MonsterVarieties": {
			// every column set
			{"Metadata/Monsters/Zombie", 0, []int{modMonster}, []int{tagLife}, []int{0}, []int{monsterMod(0)},
				"Zombie", int32(165), int32(375), int32(1200), 0, 2,
				[]int{monsterMod(1)}, []int{monsterMod(2)}, []int{monsterMod(3)}, []int{monsterMod(4)}, int32(150)},
			monsterVariety("Metadata/Monsters/TotemA", "Totem", 1),
			monsterVariety("Metadata/Monsters/TotemB", "", 1),
			monsterVariety("Metadata/Monsters/ZombieSpectre", "Zombie", 0),
		},
		// out of level order
		"DefaultMonsterStats": {
			monsterLevel("2", 1, 20, 2.5),
			{"1", float32(1.5), int32(3), int32(4), int32(10), int32(5), int32(6), int32(1), float32(1.25), int32(7),
				float32(1.75), float32(2), int32(8), int32(9), int32(11), int32(250)},
		},
		"DisplayMinionMonsterType": {{int32(1), 0}},
		// out of key and skin order
		"SkillTotemVariations": {{uint32(5), int32(1), 2}, {uint32(1), int32(0), 1}, {uint32(5), int32(0), 1}},
		"SpectreOverrides":     {{0, 3}},
		"MinionType":           {{"ZombieMinion", 3, 4}, {"OtherMinion", dattest.Null, dattest.Null}},
	}
}

// monsterFiles encodes rows as table files, with Monster.ot and Mods.
func monsterFiles(t *testing.T, rows map[string][][]any) fstest.MapFS {
	t.Helper()
	mods := itemMods()
	for _, id := range monsterMods {
		mods = append(mods, modRow{id: id, domain: 3, generation: 3, typ: 0})
	}
	g := fstest.MapFS{
		"Metadata/Monsters/Monster.ot": {Data: utf16le("extends \"nothing\"\nStats\n{\n\tbase_speed = 10\n\tlevel = 1\n}\n")},
		"Data/Mods.datc64":             {Data: encodeMods(t, mods...)},
	}
	for table, rs := range rows {
		g["Data/"+table+".datc64"] = &fstest.MapFile{Data: encode(t, table, rs...)}
	}
	return g
}

func TestMonsters(t *testing.T) {
	t.Parallel()
	in := input(t, game(t), export())
	in.EditionTotems = []dataset.EditionTotem{{GrantedEffect: "SupportEcho", Totem: 1, Source: "test"}}
	got, err := extract.Dataset(in)
	if err != nil {
		t.Fatalf("Dataset: %v", err)
	}
	limit, count := uint32(3), uint32(4)
	var plainMods []dataset.Mod
	for _, id := range monsterMods {
		plainMods = append(plainMods, dataset.Mod{ID: id, Domain: 3, Generation: 3, Type: "IncreasedLife"})
	}
	plain := func(id, typ string) dataset.MonsterVariety {
		return dataset.MonsterVariety{ID: id, Type: typ, LifeMultiplier: 100, DamageMultiplier: 100, AttackTime: 1000, AilmentThreshold: 100}
	}
	want := &dataset.Monsters{
		Types: []dataset.MonsterType{
			{ID: "Zombie", Accuracy: 10, PlayerMinion: true, Armour: 20, Evasion: 30, EnergyShield: 40, DamageSpread: 50,
				Resistances: "Strong", AltLife1: true, BaseDamageIgnoresAttackSpeed: true},
			{ID: "Totem", PlayerMinion: true, AltLife2: true},
		},
		Resistances: []dataset.MonsterResistance{{
			ID:        "Strong",
			Normal:    dataset.Resistances{Fire: 1, Cold: 2, Lightning: 3, Chaos: 4},
			Cruel:     dataset.Resistances{Fire: 5, Cold: 6, Lightning: 7, Chaos: 8},
			Merciless: dataset.Resistances{Fire: 9, Cold: 10, Lightning: 11, Chaos: 12},
		}},
		Varieties: []dataset.MonsterVariety{
			{
				ID: "Metadata/Monsters/Zombie", Type: "Zombie", LifeMultiplier: 375, DamageMultiplier: 165, AilmentThreshold: 150,
				AttackTime: 1200, MainHand: "Body Armour", OffHand: "Shield", Tags: []string{"life"}, GrantedEffects: []string{"Fireball"},
				// one mod per list, so a list read from the wrong column fails
				Mods: []string{"MonsterLife"}, Mods2: []string{"MonsterMods2"}, SpecialMods: []string{"MonsterSpecial"},
				Part1Mods: []string{"MonsterPart1"}, Part2Mods: []string{"MonsterPart2"}, EndgameMods: []string{"MonsterEndgame"},
			},
			plain("Metadata/Monsters/TotemA", "Totem"),
			plain("Metadata/Monsters/TotemB", "Totem"),
			plain("Metadata/Monsters/ZombieSpectre", "Zombie"),
		},
		// sorted by level
		Levels: []dataset.MonsterLevel{
			{Level: 1, Difficulty: 1, Life: 10, AllyLife: 5, AltLife1: 7, AltLife2: 8, Damage: 1.5, AllyDamage: 1.25,
				AltDamage1: 1.75, AltDamage2: 2, Armour: 6, Evasion: 3, EvasiveEvasion: 9, Accuracy: 4, AilmentThreshold: 11,
				PhysicalConversion: 250},
			{Level: 2, Difficulty: 1, Life: 20, Damage: 2.5},
		},
		// sorted by stat
		BaseStats: []dataset.StatValue{{Stat: 0, Value: 1}, {Stat: 2, Value: 10}},
		// only the monster domain
		Mods: append([]dataset.Mod{{ID: "MonsterLife", Domain: 3, Generation: 3, Type: "IncreasedLife",
			Stats: []dataset.ModStat{{Stat: 1, Min: 1, Max: 1, RuthlessMin: 1, RuthlessMax: 1}}}}, plainMods...),
		DisplayMinions: []dataset.DisplayMinion{{ID: 1, Variety: "Metadata/Monsters/Zombie"}},
		// sorted by key, skins by skin
		Totems: []dataset.Totem{
			{Key: 1, Skins: []dataset.TotemSkin{{Skin: 0, Variety: "Metadata/Monsters/TotemA"}}},
			{Key: 5, Skins: []dataset.TotemSkin{{Skin: 0, Variety: "Metadata/Monsters/TotemA"}, {Skin: 1, Variety: "Metadata/Monsters/TotemB"}}},
		},
		SpectreOverrides: []dataset.SpectreOverride{{Monster: "Metadata/Monsters/Zombie", Spectre: "Metadata/Monsters/ZombieSpectre"}},
		MinionTypes:      []dataset.MinionType{{ID: "ZombieMinion", LimitStat: &limit, ActiveCountStat: &count}, {ID: "OtherMinion"}},
		EditionTotems:    []dataset.EditionTotem{{GrantedEffect: "SupportEcho", Totem: 1, Source: "test"}},
	}
	if diff := cmp.Diff(want, got.Monsters, cmpopts.EquateEmpty()); diff != "" {
		t.Errorf("Monsters mismatch (-want +got):\n%s", diff)
	}
	// a variety without a name has no entry
	wantNames := []dataset.Name{
		{ID: "Metadata/Monsters/Zombie", Name: "Zombie"},
		{ID: "Metadata/Monsters/TotemA", Name: "Totem"},
		{ID: "Metadata/Monsters/ZombieSpectre", Name: "Zombie"},
	}
	if diff := cmp.Diff(wantNames, got.Texts[0].MonsterNames); diff != "" {
		t.Errorf("monster names mismatch (-want +got):\n%s", diff)
	}
	if _, err := dataset.Write(t.TempDir(), got); err != nil {
		t.Errorf("Write: %v", err)
	}
}

func TestMonstersErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		edit  func(rows map[string][][]any)
		game  func(fstest.MapFS)
		error string
	}{
		{name: "missing table", game: func(g fstest.MapFS) { delete(g, "Data/SpectreOverrides.datc64") }, error: "SpectreOverrides"},
		{name: "missing template", game: func(g fstest.MapFS) { delete(g, "Metadata/Monsters/Monster.ot") }, error: "Monster"},
		{name: "template stat unknown", game: func(g fstest.MapFS) {
			g["Metadata/Monsters/Monster.ot"] = &fstest.MapFile{Data: utf16le("extends \"nothing\"\nStats\n{\n\tnew_stat = 1\n}\n")}
		}, error: "stat new_stat not in Stats"},
		{name: "level not a number", edit: func(r map[string][][]any) { r["DefaultMonsterStats"][0][0] = "2a" }, error: `DisplayLevel "2a" is not a level`},
		{name: "resistances out of range", edit: func(r map[string][][]any) { r["MonsterTypes"][0][7] = 9 }, error: "MonsterTypes row 0: MonsterResistances row 9 out of range"},
		{name: "variety without type", edit: func(r map[string][][]any) { r["MonsterVarieties"][1][1] = dattest.Null }, error: "MonsterVarieties row 1: null MonsterTypes reference"},
		{name: "item mod on a variety", edit: func(r map[string][][]any) { r["MonsterVarieties"][0][5] = []int{monsterMod(0), modItem} },
			error: "MonsterVarieties row 0: ModsKeys2 item 1: not a monster mod"},
		{name: "variety mod out of range", edit: func(r map[string][][]any) { r["MonsterVarieties"][0][12] = []int{99} },
			error: "MonsterVarieties row 0: Special_ModsKeys: Mods row 99 out of range"},
		{name: "granted effect out of range", edit: func(r map[string][][]any) { r["MonsterVarieties"][0][4] = []int{99} },
			error: "MonsterVarieties row 0: GrantedEffects row 99 out of range"},
		{name: "display minion without variety", edit: func(r map[string][][]any) { r["DisplayMinionMonsterType"][0][1] = dattest.Null },
			error: "DisplayMinionMonsterType row 0: null MonsterVarieties reference"},
		{name: "totem key out of range", edit: func(r map[string][][]any) { r["SkillTotemVariations"][0][0] = uint32(1 << 31) },
			error: "SkillTotemVariations row 0: column SkillTotemsKey"},
		{name: "totem variety out of range", edit: func(r map[string][][]any) { r["SkillTotemVariations"][0][2] = 99 },
			error: "SkillTotemVariations row 0: MonsterVarieties row 99 out of range"},
		{name: "spectre override without spectre", edit: func(r map[string][][]any) { r["SpectreOverrides"][0][1] = dattest.Null },
			error: "SpectreOverrides row 0: null MonsterVarieties reference"},
		{name: "minion stat out of range", edit: func(r map[string][][]any) { r["MinionType"][0][2] = 99 },
			error: "MinionType row 0: Stats row 99 out of range"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			g := game(t)
			if testCase.edit != nil {
				rows := monsterRows()
				testCase.edit(rows)
				maps.Copy(g, monsterFiles(t, rows))
			}
			if testCase.game != nil {
				testCase.game(g)
			}
			_, err := extract.Dataset(input(t, g, export()))
			if err == nil || !strings.Contains(err.Error(), testCase.error) {
				t.Errorf("Dataset: got error %v, want one containing %q", err, testCase.error)
			}
		})
	}
}
