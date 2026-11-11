package extract

import (
	"fmt"

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
