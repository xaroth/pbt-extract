package dat_test

import (
	"encoding/binary"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/dat/dattest"
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

// row is a row of table T with the given strings, references and i32 list; the other cells
// are fixed.
func row(id, name1, name2 string, self, other int, ints ...int32) []any {
	return []any{id, true, int16(-2), uint16(65000), int32(-7), uint32(4000000000), float32(1.5),
		self, other, uint32(3), [2]int32{10, 20}, ints, []string{name1, name2}, 5}
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
	b := dattest.Encode(t, def, row("first", "a", "Ünïcödé", 1, dattest.Null, 1, -2, 3), row("second", "", "x", dattest.Null, 7))
	tbl, err := dat.Read(def, b)
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
	def := tableDef(t)
	tbl, err := dat.Read(def, dattest.Encode(t, def))
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
	good := dattest.Encode(t, def, row("id", "a", "b", 0, 0))
	wider := slices.Insert(slices.Clone(good), 4+rowSize, 0, 0, 0, 0)

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
		b := dattest.Encode(t, def, row("id", "a", "b", 0, 0))
		cell := b[4+colOffset(t, def, col):]
		if def.Columns[col].Array {
			le.PutUint64(cell, count)
			le.PutUint64(cell[8:], off)
		} else {
			le.PutUint64(cell, off)
		}
		return b
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
		{"row reference past the table", corrupt(7, 1, 0), 0, 7},
		{"foreignrow reference above 2^31-1", corrupt(8, 1<<31, 0), 0, 8},
		{"foreignrow reference with the top bit set", corrupt(8, 1<<63, 0), 0, 8},
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

func TestColumn(t *testing.T) {
	t.Parallel()
	def := tableDef(t)
	testCases := []struct {
		name   string
		want   int
		wantOK bool
	}{
		{name: "Id", want: 0, wantOK: true},
		{name: "Names", want: 12, wantOK: true},
		{name: "Missing", want: 0, wantOK: false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, ok := def.Column(testCase.name)
			if got != testCase.want || ok != testCase.wantOK {
				t.Errorf("Column(%q): got %d, %v, want %d, %v", testCase.name, got, ok, testCase.want, testCase.wantOK)
			}
		})
	}
}
