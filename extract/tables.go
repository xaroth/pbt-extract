package extract

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
)

// everyLeague are the tables read with readRows, readIDs or readTable. Each is in the game files
// of every league datasets are built for, 3.27 to 3.30, and of 3.26: a client without one fails
// the build. A table that only some leagues have is read with readOptionalRows instead.
var everyLeague = map[string]bool{
	"ActiveSkills": true, "ActiveSkillType": true, "AlternatePassiveAdditions": true,
	"AlternatePassiveSkills": true, "AlternateTreeVersions": true, "ArmourTypes": true,
	"BaseItemTypes": true, "BuffDefinitions": true, "BuffTemplates": true, "Characters": true,
	"ComponentCharges": true, "CostTypes": true, "CraftingBenchOptions": true,
	"CraftingItemClassCategories": true, "DefaultMonsterStats": true, "DisplayMinionMonsterType": true,
	"EffectivenessCostConstants": true, "Essences": true, "Flasks": true, "GemEffects": true,
	"GemTags": true, "GrantedEffectQualityStats": true, "GrantedEffects": true,
	"GrantedEffectsPerLevel": true, "GrantedEffectStatSets": true, "GrantedEffectStatSetsPerLevel": true,
	"ItemClasses": true, "MinionType": true, "ModFamily": true, "Mods": true, "ModType": true,
	"MonsterResistances": true, "MonsterTypes": true, "MonsterVarieties": true,
	"PassiveJewelRadii": true, "PassiveJewelSlots": true, "PassiveSkillMasteryEffects": true,
	"PassiveSkillMasteryGroups": true, "PassiveSkillOverrides": true, "PassiveSkillOverrideTypes": true,
	"PassiveSkills": true, "PassiveSkillTattoos": true, "PassiveSkillTattooTargetSets": true,
	"PassiveTreeExpansionJewels": true, "PassiveTreeExpansionJewelSizes": true,
	"PassiveTreeExpansionSkills": true, "PassiveTreeExpansionSpecialSkills": true, "Quest": true,
	"QuestFlags": true, "QuestStaticRewards": true, "ShieldTypes": true, "SkillGems": true,
	"SkillTotemVariations": true, "SpectreOverrides": true, "Stats": true, "Tags": true,
	"Tinctures": true, "WeaponTypes": true,
}

func tablePath(name string) string { return "Data/" + name + ".datc64" }

// readTableFile returns the file of a table in everyLeague.
func readTableFile(r ot.Reader, name string) ([]byte, error) {
	if !everyLeague[name] {
		return nil, fmt.Errorf("table %s is not one every league has; read it as optional", name)
	}
	return r.ReadFile(tablePath(name))
}

// readOptionalRows is readRows for a table that only some leagues have. A client without the
// file reads as no rows, with a warning.
func readOptionalRows[T any](r ot.Reader, s *dat.Schema, name string, warn func(string)) ([]T, error) {
	if everyLeague[name] {
		return nil, fmt.Errorf("table %s is listed as in every league", name)
	}
	b, err := r.ReadFile(tablePath(name))
	if errors.Is(err, fs.ErrNotExist) {
		warn(fmt.Sprintf("table %s: not in the game files; read as empty", name))
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return dat.Rows[T](s, name, b)
}
