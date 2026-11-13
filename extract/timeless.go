package extract

import (
	"fmt"
	"strings"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
)

type timelessVersionRow struct {
	ID                       string `dat:"Id"`
	SmallAttributeReplaced   bool   `dat:"SmallAttributeReplaced"`
	SmallNormalReplaced      bool   `dat:"SmallNormalReplaced"`
	SmallAttributeMin        int32  `dat:"SmallAttributeAdditionsMin"`
	SmallAttributeMax        int32  `dat:"SmallAttributeAdditionsMax"`
	SmallNormalMin           int32  `dat:"SmallNormalAdditionsMin"`
	SmallNormalMax           int32  `dat:"SmallNormalAdditionsMax"`
	NotableMin               int32  `dat:"NotableAdditionsMin"`
	NotableMax               int32  `dat:"NotableAdditionsMax"`
	NotableReplacementChance int32  `dat:"NotableReplacementChance"`
}

type timelessReplacementRow struct {
	ID        string    `dat:"Id"`
	Version   dat.Ref   `dat:"AlternateTreeVersionsKey" ref:"AlternateTreeVersions"`
	Kinds     []uint32  `dat:"PassiveType"`
	Stats     []dat.Ref `dat:"StatsKeys" ref:"Stats"`
	Stat1Min  int32     `dat:"Stat1Min"`
	Stat1Max  int32     `dat:"Stat1Max"`
	Stat2Min  int32     `dat:"Stat2Min"`
	Stat2Max  int32     `dat:"Stat2Max"`
	Weight    int32     `dat:"SpawnWeight"`
	RandomMin int32     `dat:"RandomMin"`
	RandomMax int32     `dat:"RandomMax"`
	Conqueror int32     `dat:"ConquerorIndex"`
	ConqVer   int32     `dat:"ConquerorVersion"`
}

type timelessAdditionRow struct {
	ID       string    `dat:"Id"`
	Version  dat.Ref   `dat:"AlternateTreeVersionsKey" ref:"AlternateTreeVersions"`
	Weight   int32     `dat:"SpawnWeight"`
	Stats    []dat.Ref `dat:"StatsKeys" ref:"Stats"`
	Stat1Min int32     `dat:"Stat1Min"`
	Stat1Max int32     `dat:"Stat1Max"`
	Stat2Min int32     `dat:"Stat2Min"`
	Stat2Max int32     `dat:"Stat2Max"`
	Kinds    []int32   `dat:"PassiveType"`
}

// abyssPrefix starts the ids of the Abyss versions, which no known jewel uses.
const abyssPrefix = "Abyss"

// timelessVersions reads the alternate tree versions with their replacement and addition rows,
// without the Abyss versions. A jewel's mod names its version by game row, so the versions kept
// must come before every Abyss version; otherwise the build fails.
func timelessVersions(r ot.Reader, s *dat.Schema, nStats int) ([]dataset.TimelessVersion, error) {
	versions, err := readRows[timelessVersionRow](r, s, "AlternateTreeVersions")
	if err != nil {
		return nil, err
	}
	var out []dataset.TimelessVersion
	for i, v := range versions {
		if strings.HasPrefix(v.ID, abyssPrefix) {
			continue
		}
		if len(out) != i {
			return nil, fmt.Errorf("AlternateTreeVersions: version %s at row %d follows an Abyss version; kept versions must come first so their index is their row", v.ID, i)
		}
		out = append(out, dataset.TimelessVersion{
			ID:                       v.ID,
			SmallAttributeReplaced:   v.SmallAttributeReplaced,
			SmallNormalReplaced:      v.SmallNormalReplaced,
			SmallAttributeAdditions:  dataset.ValueRange{Min: v.SmallAttributeMin, Max: v.SmallAttributeMax},
			SmallNormalAdditions:     dataset.ValueRange{Min: v.SmallNormalMin, Max: v.SmallNormalMax},
			NotableAdditions:         dataset.ValueRange{Min: v.NotableMin, Max: v.NotableMax},
			NotableReplacementChance: v.NotableReplacementChance,
		})
	}
	replacements, err := readRows[timelessReplacementRow](r, s, "AlternatePassiveSkills")
	if err != nil {
		return nil, err
	}
	for i, row := range replacements {
		v, keep, err := keptVersion(row.Version, len(versions), len(out))
		if err == nil && keep {
			var rep dataset.TimelessReplacement
			if rep, err = replacement(row, nStats); err == nil {
				out[v].Replacements = append(out[v].Replacements, rep)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("AlternatePassiveSkills row %d (%s): %w", i, row.ID, err)
		}
	}
	additions, err := readRows[timelessAdditionRow](r, s, "AlternatePassiveAdditions")
	if err != nil {
		return nil, err
	}
	for i, row := range additions {
		v, keep, err := keptVersion(row.Version, len(versions), len(out))
		if err == nil && keep {
			var add dataset.TimelessAddition
			if add, err = addition(row, nStats); err == nil {
				out[v].Additions = append(out[v].Additions, add)
			}
		}
		if err != nil {
			return nil, fmt.Errorf("AlternatePassiveAdditions row %d (%s): %w", i, row.ID, err)
		}
	}
	return out, nil
}

// keptVersion returns the version row ref references and whether it is one of the kept first
// versions.
func keptVersion(ref dat.Ref, versions, kept int) (int, bool, error) {
	v, err := rowOf(ref, versions, "AlternateTreeVersions")
	return v, v < kept, err
}

func replacement(row timelessReplacementRow, nStats int) (dataset.TimelessReplacement, error) {
	rep := dataset.TimelessReplacement{
		ID:        row.ID,
		Additions: dataset.ValueRange{Min: row.RandomMin, Max: row.RandomMax},
	}
	if row.Conqueror < 0 || row.ConqVer < 0 {
		return rep, fmt.Errorf("conqueror %d version %d negative", row.Conqueror, row.ConqVer)
	}
	rep.Conqueror = dataset.TimelessConqueror{Index: uint32(row.Conqueror), Version: uint32(row.ConqVer)}
	var err error
	if rep.Kinds, err = timelessKinds(row.Kinds); err != nil {
		return rep, err
	}
	if rep.Weight, err = weight(row.Weight); err != nil {
		return rep, err
	}
	// The columns of a third and fourth stat are not known; no row has one yet.
	rep.Stats, err = statRanges(row.Stats, nStats, [][2]int32{{row.Stat1Min, row.Stat1Max}, {row.Stat2Min, row.Stat2Max}})
	return rep, err
}

func addition(row timelessAdditionRow, nStats int) (dataset.TimelessAddition, error) {
	add := dataset.TimelessAddition{ID: row.ID}
	kinds := make([]uint32, len(row.Kinds))
	for i, k := range row.Kinds {
		if k < 0 {
			return add, fmt.Errorf("passive type %d", k)
		}
		kinds[i] = uint32(k)
	}
	var err error
	if add.Kinds, err = timelessKinds(kinds); err != nil {
		return add, err
	}
	if add.Weight, err = weight(row.Weight); err != nil {
		return add, err
	}
	add.Stats, err = statRanges(row.Stats, nStats, [][2]int32{{row.Stat1Min, row.Stat1Max}, {row.Stat2Min, row.Stat2Max}})
	return add, err
}

// timelessKinds converts PassiveType values. A row without one could never be drawn, and the
// engine refuses it.
func timelessKinds(types []uint32) ([]dataset.TimelessKind, error) {
	if len(types) == 0 {
		return nil, fmt.Errorf("no passive type")
	}
	out := make([]dataset.TimelessKind, len(types))
	for i, t := range types {
		if t < uint32(dataset.TimelessSmallAttribute) || t > uint32(dataset.TimelessJewelSocket) {
			return nil, fmt.Errorf("unknown passive type %d", t)
		}
		out[i] = dataset.TimelessKind(t)
	}
	return out, nil
}

func weight(w int32) (uint32, error) {
	if w < 0 {
		return 0, fmt.Errorf("spawn weight %d negative", w)
	}
	return uint32(w), nil
}

// statRanges pairs the stats refs reference with the value ranges of their columns, in row
// order.
func statRanges(refs []dat.Ref, nStats int, ranges [][2]int32) ([]dataset.StatRange, error) {
	if len(refs) > len(ranges) {
		return nil, fmt.Errorf("%d stats, value columns known for %d", len(refs), len(ranges))
	}
	handles, err := statHandles(refs, nStats)
	if err != nil {
		return nil, err
	}
	out := make([]dataset.StatRange, len(handles))
	for i, h := range handles {
		out[i] = dataset.StatRange{Stat: h, Min: ranges[i][0], Max: ranges[i][1]}
	}
	return out, nil
}
