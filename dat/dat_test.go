package dat_test

import (
	"bytes"
	"encoding/binary"
	"math"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/google/go-cmp/cmp"
	"github.com/xaroth/pbt-extract/dat"
)

const schemaJSON = `{"version": 7, "tables": [{"name": "T", "columns": [
	{"name": "Id", "type": "string", "array": false},
	{"name": "Flag", "type": "bool", "array": false},
	{"name": "Small", "type": "i16", "array": false},
	{"name": "USmall", "type": "u16", "array": false},
	{"name": "Int", "type": "i32", "array": false},
	{"name": "UInt", "type": "u32", "array": false},
	{"name": "Float", "type": "f32", "array": false},
	{"name": "Self", "type": "row", "array": false},
	{"name": "Other", "type": "foreignrow", "array": false, "references": {"table": "U"}},
	{"name": "Kind", "type": "enumrow", "array": false},
	{"name": "Range", "type": "i32", "array": false, "interval": true},
	{"name": "Ints", "type": "i32", "array": true},
	{"name": "Names", "type": "string", "array": true},
	{"name": null, "type": "array", "array": true}
]}]}`

// rowSize of table T: 8+1+2+2+4+4+4+8+16+4+8+16+16+16.
const rowSize = 109

type tableBuilder struct {
	fixed, vars []byte
}

func newTableBuilder() *tableBuilder {
	return &tableBuilder{vars: bytes.Repeat([]byte{0xbb}, 8)}
}

func (b *tableBuilder) str(s string) uint64 {
	off := uint64(len(b.vars))
	for _, u := range utf16.Encode([]rune(s)) {
		b.vars = binary.LittleEndian.AppendUint16(b.vars, u)
	}
	b.vars = append(b.vars, 0, 0, 0, 0)
	return off
}

func (b *tableBuilder) row(id, name1, name2 string, self, other uint64, ints ...int32) {
	le := binary.LittleEndian
	idOff := b.str(id)
	n1, n2 := b.str(name1), b.str(name2)
	intsOff := uint64(len(b.vars))
	for _, v := range ints {
		b.vars = le.AppendUint32(b.vars, uint32(v))
	}
	namesOff := uint64(len(b.vars))
	b.vars = le.AppendUint64(b.vars, n1)
	b.vars = le.AppendUint64(b.vars, n2)

	r := le.AppendUint64(nil, idOff)
	r = append(r, 1)
	r = le.AppendUint16(r, uint16(0xfffe)) // -2 as i16
	r = le.AppendUint16(r, 65000)
	r = le.AppendUint32(r, uint32(0xfffffff9)) // -7 as i32
	r = le.AppendUint32(r, 4000000000)
	r = le.AppendUint32(r, math.Float32bits(1.5))
	r = le.AppendUint64(r, self)
	r = le.AppendUint64(r, other)
	r = le.AppendUint64(r, 0)
	r = le.AppendUint32(r, 3)
	r = le.AppendUint32(r, 10)
	r = le.AppendUint32(r, 20)
	r = le.AppendUint64(r, uint64(len(ints)))
	r = le.AppendUint64(r, intsOff)
	r = le.AppendUint64(r, 2)
	r = le.AppendUint64(r, namesOff)
	r = le.AppendUint64(r, 5)
	r = le.AppendUint64(r, 0)
	b.fixed = append(b.fixed, r...)
}

func (b *tableBuilder) bytes(rows int) []byte {
	out := binary.LittleEndian.AppendUint32(nil, uint32(rows))
	out = append(out, b.fixed...)
	return append(out, b.vars...)
}

func tableDef(t *testing.T) dat.TableDef {
	t.Helper()
	s, err := dat.ParseSchema([]byte(schemaJSON))
	if err != nil {
		t.Fatal(err)
	}
	def, ok := s.Table("T")
	if !ok {
		t.Fatal("table T missing")
	}
	return def
}

func TestRead(t *testing.T) {
	t.Parallel()
	def := tableDef(t)
	if got := def.RowSize(); got != rowSize {
		t.Fatalf("got row size %d, want %d", got, rowSize)
	}
	b := newTableBuilder()
	b.row("first", "a", "Ünïcödé", 1, 0xfefefefefefefefe, 1, -2, 3)
	b.row("second", "", "x", 0xfefefefefefefefe, 7)
	tbl, err := dat.Read(def, b.bytes(2))
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Len() != 2 {
		t.Fatalf("got %d rows, want 2", tbl.Len())
	}
	testCases := []struct {
		name     string
		row, col int
		want     any
	}{
		{"string", 0, 0, "first"},
		{"bool", 0, 1, true},
		{"i16", 0, 2, int16(-2)},
		{"u16", 0, 3, uint16(65000)},
		{"i32", 0, 4, int32(-7)},
		{"u32", 0, 5, uint32(4000000000)},
		{"f32", 0, 6, float32(1.5)},
		{"row", 0, 7, 1},
		{"empty foreignrow", 0, 8, nil},
		{"empty row", 1, 7, nil},
		{"foreignrow", 1, 8, 7},
		{"enumrow", 0, 9, uint32(3)},
		{"interval", 0, 10, [2]any{int32(10), int32(20)}},
		{"i32 list", 0, 11, []any{int32(1), int32(-2), int32(3)}},
		{"empty list", 1, 11, []any{}},
		{"string list", 0, 12, []any{"a", "Ünïcödé"}},
		{"empty string in list", 1, 12, []any{"", "x"}},
		{"unknown element list", 0, 13, 5},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, err := tbl.Value(testCase.row, testCase.col)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(testCase.want, got); diff != "" {
				t.Errorf("(-want +got)\n%s", diff)
			}
		})
	}
}

func TestReadEmptyTable(t *testing.T) {
	t.Parallel()
	tbl, err := dat.Read(tableDef(t), newTableBuilder().bytes(0))
	if err != nil {
		t.Fatal(err)
	}
	if tbl.Len() != 0 {
		t.Errorf("got %d rows, want 0", tbl.Len())
	}
}

func TestReadErrors(t *testing.T) {
	t.Parallel()
	def := tableDef(t)
	b := newTableBuilder()
	b.row("id", "a", "b", 0, 0)
	good := b.bytes(1)
	wider := append(append(binary.LittleEndian.AppendUint32(nil, 1), b.fixed...), 0, 0, 0, 0)
	wider = append(wider, b.vars...)

	testCases := []struct {
		name    string
		b       []byte
		wantMsg string
	}{
		{"too short", good[:2], "shorter than row count"},
		{"schema row size differs from file", wider, "schema row size 109, file row size 113"},
		{"no marker", good[:4+rowSize], "no variable-data marker"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := dat.Read(def, testCase.b)
			if err == nil || !strings.Contains(err.Error(), testCase.wantMsg) {
				t.Errorf("got error %v, want one containing %q", err, testCase.wantMsg)
			}
		})
	}
}

func TestValueErrors(t *testing.T) {
	t.Parallel()
	def := tableDef(t)
	le := binary.LittleEndian
	corrupt := func(col int, off uint64, count uint64) []byte {
		b := newTableBuilder()
		b.row("id", "a", "b", 0, 0)
		cell := b.fixed[colOffset(t, def, col):]
		if def.Columns[col].Array {
			le.PutUint64(cell, count)
			le.PutUint64(cell[8:], off)
		} else {
			le.PutUint64(cell, off)
		}
		return b.bytes(1)
	}
	testCases := []struct {
		name     string
		b        []byte
		row, col int
	}{
		{"string offset inside marker", corrupt(0, 4, 0), 0, 0},
		{"string offset past end", corrupt(0, 1<<40, 0), 0, 0},
		{"list past end", corrupt(11, 8, 1<<20), 0, 11},
		{"row out of range", corrupt(0, 8, 0), 1, 0},
		{"column out of range", corrupt(0, 8, 0), 0, 14},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			tbl, err := dat.Read(def, testCase.b)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := tbl.Value(testCase.row, testCase.col); err == nil {
				t.Error("got nil error, want an error")
			}
		})
	}
}

func colOffset(t *testing.T, def dat.TableDef, col int) int {
	t.Helper()
	sub := dat.TableDef{Columns: def.Columns[:col]}
	return sub.RowSize()
}

func TestParseSchemaErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct{ name, json string }{
		{"not json", `{`},
		{"format version", `{"version": 6, "tables": []}`},
		{"unknown type", `{"version": 7, "tables": [{"name": "T", "columns": [{"type": "i64"}]}]}`},
		{"interval list", `{"version": 7, "tables": [{"name": "T", "columns": [{"type": "i32", "array": true, "interval": true}]}]}`},
		{"duplicate table", `{"version": 7, "tables": [{"name": "T", "columns": []}, {"name": "T", "columns": []}]}`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := dat.ParseSchema([]byte(testCase.json)); err == nil {
				t.Error("got nil error, want an error")
			}
		})
	}
}
