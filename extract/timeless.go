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
	Name      string    `dat:"Name"`
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
// up to the first Abyss version, and the English names of the kept replacements in table order.
// A jewel's mod names its version by game row, so a version after an Abyss one cannot keep its
// row as its index; it is left out with its rows and a warning. A replacement without a name is
// a warning.
func timelessVersions(r ot.Reader, s *dat.Schema, nStats int, warn func(string)) ([]dataset.TimelessVersion, []dataset.Name, error) {
	versions, err := readRows[timelessVersionRow](r, s, "AlternateTreeVersions")
	if err != nil {
		return nil, nil, err
	}
	var out []dataset.TimelessVersion
	var dropped []int // rows of versions after an Abyss one
	for i, v := range versions {
		if strings.HasPrefix(v.ID, abyssPrefix) {
			continue
		}
		if len(out) != i {
			dropped = append(dropped, i)
			continue
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
		return nil, nil, err
	}
	var names []dataset.Name
	for i, row := range replacements {
		v, keep, err := keptVersion(row.Version, len(versions), len(out))
		if err == nil && keep {
			var rep dataset.TimelessReplacement
			if rep, err = replacement(row, nStats); err == nil {
				out[v].Replacements = append(out[v].Replacements, rep)
				if row.Name == "" {
					warn(fmt.Sprintf("timeless passive %s has no name", row.ID))
				} else {
					names = append(names, dataset.Name{ID: row.ID, Name: row.Name})
				}
			}
		}
		if err != nil {
			return nil, nil, fmt.Errorf("AlternatePassiveSkills row %d (%s): %w", i, row.ID, err)
		}
	}
	additions, err := readRows[timelessAdditionRow](r, s, "AlternatePassiveAdditions")
	if err != nil {
		return nil, nil, err
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
			return nil, nil, fmt.Errorf("AlternatePassiveAdditions row %d (%s): %w", i, row.ID, err)
		}
	}
	for _, v := range dropped {
		warn(fmt.Sprintf("timeless version %s (row %d) follows an Abyss version; left out with its %d replacements and %d additions",
			versions[v].ID, v, countVersion(replacements, v, func(r timelessReplacementRow) dat.Ref { return r.Version }),
			countVersion(additions, v, func(r timelessAdditionRow) dat.Ref { return r.Version })))
	}
	return out, names, nil
}

// countVersion counts the rows whose version is row v.
func countVersion[T any](rows []T, v int, version func(T) dat.Ref) int {
	n := 0
	for _, r := range rows {
		if row, ok := version(r).Row(); ok && row == v {
			n++
		}
	}
	return n
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
