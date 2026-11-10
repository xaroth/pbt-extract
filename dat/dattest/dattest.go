// Package dattest encodes .datc64 tables for tests.
package dattest

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"testing"
	"unicode/utf16"

	"github.com/xaroth/pbt-extract/dat"
)

// Null is an empty row or foreignrow value.
const Null = -1

// Encode encodes rows as a .datc64 file of def. Each row has one value per column, of the Go
// type Table.Value decodes that column to, except: row and foreignrow take an int row index or
// Null; intervals take a two-element array; lists take a slice of their element type; a list
// of unknown element type takes its element count as an int. A value of another type fails tb.
func Encode(tb testing.TB, def dat.TableDef, rows ...[]any) []byte {
	tb.Helper()
	e := encoder{tb: tb, vars: bytes.Repeat([]byte{0xbb}, 8)}
	fixed := binary.LittleEndian.AppendUint32(nil, uint32(len(rows)))
	for r, row := range rows {
		if len(row) != len(def.Columns) {
			tb.Fatalf("table %s row %d: %d values for %d columns", def.Name, r, len(row), len(def.Columns))
		}
		for c, v := range row {
			fixed = e.cell(fixed, def.Columns[c], v)
		}
	}
	return append(fixed, e.vars...)
}

type encoder struct {
	tb   testing.TB
	vars []byte
}

func (e *encoder) cell(b []byte, c dat.Column, v any) []byte {
	e.tb.Helper()
	le := binary.LittleEndian
	switch {
	case c.Array && c.Type == "array":
		return le.AppendUint64(le.AppendUint64(b, uint64(e.want(c, v, 0).(int))), 0)
	case c.Array:
		rv := reflect.ValueOf(v)
		if v != nil && rv.Kind() != reflect.Slice {
			e.tb.Fatalf("column %s: got %T, want a slice", name(c), v)
		}
		var elems []byte
		n := 0
		if v != nil {
			n = rv.Len()
			for i := range n {
				elems = e.scalar(elems, c, rv.Index(i).Interface())
			}
		}
		b = le.AppendUint64(b, uint64(n))
		b = le.AppendUint64(b, uint64(len(e.vars)))
		e.vars = append(e.vars, elems...)
		return b
	case c.Interval:
		rv := reflect.ValueOf(v)
		if rv.Kind() != reflect.Array || rv.Len() != 2 {
			e.tb.Fatalf("column %s: got %T, want a two-element array", name(c), v)
		}
		return e.scalar(e.scalar(b, c, rv.Index(0).Interface()), c, rv.Index(1).Interface())
	}
	return e.scalar(b, c, v)
}

func (e *encoder) scalar(b []byte, c dat.Column, v any) []byte {
	e.tb.Helper()
	le := binary.LittleEndian
	switch c.Type {
	case "bool":
		if e.want(c, v, false).(bool) {
			return append(b, 1)
		}
		return append(b, 0)
	case "i16":
		return le.AppendUint16(b, uint16(e.want(c, v, int16(0)).(int16)))
	case "u16":
		return le.AppendUint16(b, e.want(c, v, uint16(0)).(uint16))
	case "i32":
		return le.AppendUint32(b, uint32(e.want(c, v, int32(0)).(int32)))
	case "u32", "enumrow":
		return le.AppendUint32(b, e.want(c, v, uint32(0)).(uint32))
	case "f32":
		return le.AppendUint32(b, math.Float32bits(e.want(c, v, float32(0)).(float32)))
	case "string":
		b = le.AppendUint64(b, uint64(len(e.vars)))
		for _, u := range utf16.Encode([]rune(e.want(c, v, "").(string))) {
			e.vars = le.AppendUint16(e.vars, u)
		}
		e.vars = append(e.vars, 0, 0, 0, 0)
		return b
	case "row", "foreignrow":
		size := 8
		if c.Type == "foreignrow" {
			size = 16
		}
		r := e.want(c, v, 0).(int)
		if r == Null {
			return append(b, bytes.Repeat([]byte{0xfe}, size)...)
		}
		b = le.AppendUint64(b, uint64(r))
		return append(b, make([]byte, size-8)...)
	}
	e.tb.Fatalf("column %s: unknown type %q", name(c), c.Type)
	return nil
}

// want returns v if it has the type of like, and fails otherwise.
func (e *encoder) want(c dat.Column, v, like any) any {
	e.tb.Helper()
	if reflect.TypeOf(v) != reflect.TypeOf(like) {
		e.tb.Fatalf("column %s: got %T, want %T", name(c), v, like)
	}
	return v
}

func name(c dat.Column) string {
	if c.Name == nil {
		return "(unnamed)"
	}
	return *c.Name
}
