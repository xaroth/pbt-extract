package dat

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"unicode/utf16"
)

var marker = bytes.Repeat([]byte{0xbb}, 8)

// nullRef marks an empty row or foreignrow value.
const nullRef = 0xfefefefefefefefe

// Table is one decoded .datc64 file.
type Table struct {
	def      TableDef
	rowCount int
	rowSize  int
	fixed    []byte
	varData  []byte // starts at the marker; variable offsets are relative to it
	offsets  []int  // byte offset of each column within a row, and the row size last
	columns  int    // columns of def the file holds
}

// Read parses a .datc64 file against def. A file whose rows hold only the first columns of def
// is read with those columns: an older game version that lacks columns a later version
// appended. It fails when the file's row size is not the size of def or of a prefix of its
// columns, which is how a schema from the wrong game version shows up.
func Read(def TableDef, b []byte) (*Table, error) {
	if len(b) < 4 {
		return nil, errors.New("file shorter than row count")
	}
	t := &Table{def: def, rowCount: int(binary.LittleEndian.Uint32(b))}
	off := 0
	t.offsets = append(t.offsets, off)
	for _, c := range def.Columns {
		n, _ := c.size()
		off += n
		t.offsets = append(t.offsets, off)
	}
	for t.columns = len(def.Columns); t.columns >= min(1, len(def.Columns)); t.columns-- {
		t.rowSize = t.offsets[t.columns]
		end := 4 + t.rowCount*t.rowSize
		if end+len(marker) <= len(b) && bytes.Equal(b[end:end+len(marker)], marker) {
			t.fixed, t.varData = b[4:end], b[end:]
			return t, nil
		}
	}
	if actual, ok := actualRowSize(b, t.rowCount); ok {
		return nil, fmt.Errorf("table %s: schema row size %d, file row size %d", def.Name, def.RowSize(), actual)
	}
	return nil, fmt.Errorf("table %s: no variable-data marker after %d rows of %d bytes", def.Name, t.rowCount, def.RowSize())
}

// actualRowSize finds the row size implied by the first marker that divides evenly, for
// error messages only.
func actualRowSize(b []byte, rows int) (int, bool) {
	if rows == 0 {
		return 0, false
	}
	for i := 4; ; i++ {
		j := bytes.Index(b[i:], marker)
		if j < 0 {
			return 0, false
		}
		i += j
		if (i-4)%rows == 0 {
			return (i - 4) / rows, true
		}
	}
}

// Len returns the number of rows.
func (t *Table) Len() int { return t.rowCount }

// Def returns the table definition the file was read with.
func (t *Table) Def() TableDef { return t.def }

// Columns returns how many of Def's columns the file holds; the others are missing from it.
func (t *Table) Columns() int { return t.columns }

// Value decodes one cell. Scalars decode to bool, int16, uint16, int32, uint32, float32 or
// string; row and foreignrow to an int row index, or nil when empty. A row reference past the
// table's end, or any reference above 2^31-1, is an error; whether a foreignrow index is inside
// the referenced table is the caller's check, as only it reads that table; enumrow to uint32;
// intervals to a two-element array of the scalar type; lists to a slice of those values.
// A list of unknown element type decodes to its element count as an int.
func (t *Table) Value(row, col int) (any, error) {
	if row < 0 || row >= t.rowCount {
		return nil, fmt.Errorf("row %d out of range [0, %d)", row, t.rowCount)
	}
	if col < 0 || col >= len(t.def.Columns) {
		return nil, fmt.Errorf("column %d out of range [0, %d)", col, len(t.def.Columns))
	}
	if col >= t.columns {
		return nil, fmt.Errorf("column %d not in the file, whose rows hold %d columns", col, t.columns)
	}
	c := t.def.Columns[col]
	cell := t.fixed[row*t.rowSize+t.offsets[col]:]
	if !c.Array {
		if c.Interval {
			n := scalarSizes[c.Type]
			a, err := t.scalar(c.Type, cell)
			if err != nil {
				return nil, err
			}
			b, err := t.scalar(c.Type, cell[n:])
			if err != nil {
				return nil, err
			}
			return [2]any{a, b}, nil
		}
		return t.scalar(c.Type, cell)
	}
	count := binary.LittleEndian.Uint64(cell)
	off := binary.LittleEndian.Uint64(cell[8:])
	if c.Type == "array" {
		return int(count), nil
	}
	n := uint64(scalarSizes[c.Type])
	if count > uint64(len(t.varData)) || off > uint64(len(t.varData)) || count*n > uint64(len(t.varData))-off {
		return nil, fmt.Errorf("row %d column %d: list of %d at %d outside variable data of %d bytes", row, col, count, off, len(t.varData))
	}
	out := make([]any, count)
	for i := range out {
		v, err := t.scalar(c.Type, t.varData[off+uint64(i)*n:])
		if err != nil {
			return nil, fmt.Errorf("row %d column %d element %d: %w", row, col, i, err)
		}
		out[i] = v
	}
	return out, nil
}

func (t *Table) scalar(typ string, b []byte) (any, error) {
	le := binary.LittleEndian
	switch typ {
	case "bool":
		return b[0] != 0, nil
	case "i16":
		return int16(le.Uint16(b)), nil
	case "u16":
		return le.Uint16(b), nil
	case "i32":
		return int32(le.Uint32(b)), nil
	case "u32":
		return le.Uint32(b), nil
	case "f32":
		return math.Float32frombits(le.Uint32(b)), nil
	case "enumrow":
		return le.Uint32(b), nil
	case "row", "foreignrow":
		v := le.Uint64(b)
		switch {
		case v == nullRef:
			return nil, nil
		case typ == "row" && v >= uint64(t.rowCount):
			return nil, fmt.Errorf("row reference %d outside table of %d rows", v, t.rowCount)
		case v > math.MaxInt32:
			// No table has 2^31 rows; a larger value is corrupt data, not a row.
			return nil, fmt.Errorf("row reference %#x out of range", v)
		}
		return int(v), nil
	case "string":
		return t.str(le.Uint64(b))
	}
	return nil, fmt.Errorf("unknown column type %q", typ)
}

// str reads a UTF-16LE string terminated by four zero bytes at a 2-byte boundary.
func (t *Table) str(off uint64) (string, error) {
	if off < uint64(len(marker)) || off > uint64(len(t.varData)) {
		return "", fmt.Errorf("string offset %d outside variable data of %d bytes", off, len(t.varData))
	}
	b := t.varData[off:]
	var units []uint16
	for i := 0; ; i += 2 {
		if i+4 > len(b) {
			return "", fmt.Errorf("string at %d not terminated", off)
		}
		if binary.LittleEndian.Uint32(b[i:]) == 0 {
			return string(utf16.Decode(units)), nil
		}
		units = append(units, binary.LittleEndian.Uint16(b[i:]))
	}
}
