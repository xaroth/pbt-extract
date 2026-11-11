package extract

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/ot"
)

type idRow struct {
	ID string `dat:"Id"`
}

// readRows decodes Data/<name>.datc64 into rows of T with dat.Rows.
func readRows[T any](r ot.Reader, s *dat.Schema, name string) ([]T, error) {
	b, err := r.ReadFile("Data/" + name + ".datc64")
	if err != nil {
		return nil, err
	}
	return dat.Rows[T](s, name, b)
}

// readIDs reads the Id column of a table.
func readIDs(r ot.Reader, s *dat.Schema, name string) ([]string, error) {
	rows, err := readRows[idRow](r, s, name)
	if err != nil {
		return nil, err
	}
	return ids(rows, func(r idRow) string { return r.ID }), nil
}

func ids[T any](rows []T, id func(T) string) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = id(r)
	}
	return out
}

// rowOf returns the row ref references in a table of n rows; a null reference is an error.
func rowOf(ref dat.Ref, n int, table string) (int, error) {
	row, ok := ref.Row()
	switch {
	case !ok:
		return 0, fmt.Errorf("null %s reference", table)
	case row >= n:
		return 0, fmt.Errorf("%s row %d out of range", table, row)
	}
	return row, nil
}

// lookup returns the row of rows that ref references.
func lookup[T any](ref dat.Ref, rows []T, table string) (T, error) {
	row, err := rowOf(ref, len(rows), table)
	if err != nil {
		var zero T
		return zero, err
	}
	return rows[row], nil
}

// lookupAll returns the rows of rows that refs reference.
func lookupAll[T any](refs []dat.Ref, rows []T, table string) ([]T, error) {
	var out []T
	for _, ref := range refs {
		row, err := lookup(ref, rows, table)
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, nil
}

// statHandles returns the stat handles of refs, Stats rows in a table of nStats rows.
func statHandles(refs []dat.Ref, nStats int) ([]uint32, error) {
	var out []uint32
	for _, ref := range refs {
		row, err := rowOf(ref, nStats, "Stats")
		if err != nil {
			return nil, err
		}
		out = append(out, uint32(row))
	}
	return out, nil
}

// templateStats returns the Stats block of the object template at path, merged over its
// extends chain, as stat handles sorted by stat. handles maps stat ids to handles.
func templateStats(r ot.Reader, path string, handles map[string]uint32) ([]dataset.StatValue, error) {
	stats, err := ot.Stats(r, path)
	if err != nil {
		return nil, err
	}
	var out []dataset.StatValue
	for id, v := range stats {
		h, ok := handles[id]
		if !ok {
			return nil, fmt.Errorf("%s.ot: stat %s not in Stats", path, id)
		}
		if int(int32(v)) != v {
			return nil, fmt.Errorf("%s.ot: stat %s value %d overflows int32", path, id, v)
		}
		out = append(out, dataset.StatValue{Stat: h, Value: int32(v)})
	}
	slices.SortFunc(out, func(a, b dataset.StatValue) int { return cmp.Compare(a.Stat, b.Stat) })
	return out, nil
}

// optionalStat returns the stat handle ref references, Stats rows in a table of nStats rows, or
// nil for a null reference.
func optionalStat(ref dat.Ref, nStats int) (*uint32, error) {
	if _, ok := ref.Row(); !ok {
		return nil, nil
	}
	hs, err := statHandles([]dat.Ref{ref}, nStats)
	if err != nil {
		return nil, err
	}
	return &hs[0], nil
}
