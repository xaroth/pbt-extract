package build

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
)

// QuestStaticRewards columns dat-schema leaves unnamed, by our reading of the 3.28.0.16 and
// 3.29.3.3 rows: passive points sum to 24 over the campaign quests (23 plus 1 for a2q7, killing
// all bandits, as PoB's Build.lua counts); ascendancy points are 2 on each LabyrinthComplete
// flag (8 in all); refund points are 2 on ten campaign quests (20 in all); the flag column
// marks league mechanic rows (Delve, Heist, Descent), which are not campaign rewards.
const (
	colPassivePoints    = "_1"
	colAscendancyPoints = "_5"
	colRefundPoints     = "_7"
	colLeagueReward     = "_10"
)

// quests reads the campaign and labyrinth rows of QuestStaticRewards, in table order, leaving
// out league mechanic rows. nStats is the number of rows in Stats.
func quests(r ot.Reader, schema *dat.Schema, nStats int) ([]dataset.Quest, error) {
	rewards, err := readTable(r, schema, "QuestStaticRewards", "QuestFlag", "QuestKey", "StatsKeys", "StatValues",
		colPassivePoints, colAscendancyPoints, colRefundPoints, colLeagueReward)
	if err != nil {
		return nil, err
	}
	flags, err := readTable(r, schema, "QuestFlags", "Id")
	if err != nil {
		return nil, err
	}
	qs, err := readTable(r, schema, "Quest", "Id", "Act")
	if err != nil {
		return nil, err
	}
	var out []dataset.Quest
	for row := range rewards.t.Len() {
		q, keep, err := questRow(rewards, flags, qs, row, nStats)
		if err != nil {
			return nil, fmt.Errorf("table QuestStaticRewards row %d: %w", row, err)
		}
		if keep {
			out = append(out, q)
		}
	}
	return out, nil
}

func questRow(rewards, flags, qs table, row, nStats int) (dataset.Quest, bool, error) {
	var q dataset.Quest
	league, err := rewards.boolean(row, colLeagueReward)
	if err != nil || league {
		return q, false, err
	}
	flag, err := rewards.ref(row, "QuestFlag")
	if err != nil {
		return q, false, err
	}
	if q.Flag, err = flags.str(flag, "Id"); err != nil {
		return q, false, err
	}
	if quest, err := rewards.ref(row, "QuestKey"); err != nil {
		return q, false, err
	} else if quest >= 0 {
		if q.Quest, err = qs.str(quest, "Id"); err != nil {
			return q, false, err
		}
		if q.Act, err = qs.i32(quest, "Act"); err != nil {
			return q, false, err
		}
	}
	for col, dst := range map[string]*int32{
		colPassivePoints: &q.PassivePoints, colAscendancyPoints: &q.AscendancyPoints, colRefundPoints: &q.RefundPoints,
	} {
		if *dst, err = rewards.i32(row, col); err != nil {
			return q, false, err
		}
	}
	if q.Stats, err = rewardStats(rewards, row, nStats); err != nil {
		return q, false, err
	}
	return q, true, nil
}

// rewardStats pairs StatsKeys with StatValues, sorted by stat.
func rewardStats(t table, row, nStats int) ([]dataset.StatValue, error) {
	keys, err := t.ints(row, "StatsKeys")
	if err != nil {
		return nil, err
	}
	values, err := t.i32s(row, "StatValues")
	if err != nil {
		return nil, err
	}
	if len(keys) != len(values) {
		return nil, fmt.Errorf("%d stats, %d values", len(keys), len(values))
	}
	var out []dataset.StatValue
	for i, k := range keys {
		if k < 0 || k >= nStats {
			return nil, fmt.Errorf("stat row %d out of range", k)
		}
		out = append(out, dataset.StatValue{Stat: uint32(k), Value: values[i]})
	}
	slices.SortFunc(out, func(a, b dataset.StatValue) int { return cmp.Compare(a.Stat, b.Stat) })
	return out, nil
}
