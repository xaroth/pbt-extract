package extract

import (
	"fmt"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
)

type grantLevelRow struct {
	GrantedEffect dat.Ref `dat:"GrantedEffect" ref:"GrantedEffects"`
	Level         int32   `dat:"Level"`
}

// grantLevels returns the skill each GrantedEffectsPerLevel row stands for, by row: mods and
// passives grant a skill by referencing such a row.
func grantLevels(r ot.Reader, s *dat.Schema) ([]dataset.GrantedSkill, error) {
	effects, err := readIDs(r, s, "GrantedEffects")
	if err != nil {
		return nil, err
	}
	rows, err := readRows[grantLevelRow](r, s, "GrantedEffectsPerLevel")
	if err != nil {
		return nil, err
	}
	out := make([]dataset.GrantedSkill, len(rows))
	for i, row := range rows {
		id, err := lookup(row.GrantedEffect, effects, "GrantedEffects")
		if err != nil {
			return nil, fmt.Errorf("table GrantedEffectsPerLevel row %d: %w", i, err)
		}
		out[i] = dataset.GrantedSkill{Effect: id, Level: row.Level}
	}
	return out, nil
}
