package dat

import (
	"fmt"
	"reflect"
)

// Ref is a row or foreignrow value: a row index into the referenced table. The zero Ref is
// null.
type Ref struct {
	row int
	ok  bool
}

// RefTo returns a reference to row, which must not be negative.
func RefTo(row int) Ref {
	if row < 0 {
		panic(fmt.Sprintf("dat.RefTo(%d): negative row", row))
	}
	return Ref{row: row, ok: true}
}

// Row returns the referenced row index, and false for a null reference.
func (r Ref) Row() (int, bool) { return r.row, r.ok }

// Equal reports whether r and o reference the same row; go-cmp uses it.
func (r Ref) Equal(o Ref) bool { return r == o }

var (
	refType   = reflect.TypeFor[Ref]()
	goScalars = map[string]reflect.Type{
		"bool":       reflect.TypeFor[bool](),
		"i16":        reflect.TypeFor[int16](),
		"u16":        reflect.TypeFor[uint16](),
		"i32":        reflect.TypeFor[int32](),
		"u32":        reflect.TypeFor[uint32](),
		"f32":        reflect.TypeFor[float32](),
		"string":     reflect.TypeFor[string](),
		"enumrow":    reflect.TypeFor[uint32](),
		"row":        refType,
		"foreignrow": refType,
	}
)

// Rows decodes the .datc64 file b of table name into one T per row; the slice index is the
// row index. T is a struct whose fields declare the columns read:
//
//	type passiveSkill struct {
//		GraphID uint16    `dat:"PassiveSkillGraphId"`
//		Stats   []dat.Ref `dat:"Stats" ref:"Stats"`
//		Range   [2]int32  `dat:"Range"`
//	}
//
// Every field names a column in its dat tag. A scalar column takes the Go type Table.Value
// decodes it to, with Ref for row and foreignrow; an interval takes a two-element array of
// it, a list a slice of it. A Ref field names the table it references in its ref tag, which
// must exist and match the schema's references (for row, the table itself); for a foreignrow
// column whose schema has no references, any table in the schema is accepted. A reference to
// a row past the end of the referenced table is not detected here: the step that joins the
// two tables checks it. Fields must be exported. Lists of unknown element type are not
// supported. Fields are checked against the schema before any row is decoded. A field whose
// column the file lacks (see Read) is an error.
func Rows[T any](s *Schema, name string, b []byte) ([]T, error) {
	def, ok := s.Table(name)
	if !ok {
		return nil, fmt.Errorf("table %s not in schema", name)
	}
	cols, err := fieldColumns(s, def, reflect.TypeFor[T]())
	if err != nil {
		return nil, fmt.Errorf("table %s: %w", name, err)
	}
	t, err := Read(def, b)
	if err != nil {
		return nil, err
	}
	for _, c := range cols {
		if c >= t.Columns() {
			return nil, fmt.Errorf("table %s: column %s not in the file, whose rows hold the first %d columns", name, *def.Columns[c].Name, t.Columns())
		}
	}
	out := make([]T, t.Len())
	for r := range out {
		rv := reflect.ValueOf(&out[r]).Elem()
		for f, c := range cols {
			v, err := t.Value(r, c)
			if err != nil {
				return nil, fmt.Errorf("table %s row %d: %w", name, r, err)
			}
			set(rv.Field(f), v)
		}
	}
	return out, nil
}

// fieldColumns returns the column index of each field of the struct type typ.
func fieldColumns(s *Schema, def TableDef, typ reflect.Type) ([]int, error) {
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("row type %s is not a struct", typ)
	}
	cols := make([]int, typ.NumField())
	for i := range typ.NumField() {
		f := typ.Field(i)
		if !f.IsExported() {
			return nil, fmt.Errorf("field %s is not exported", f.Name)
		}
		name, ok := f.Tag.Lookup("dat")
		if !ok {
			return nil, fmt.Errorf("field %s has no dat tag", f.Name)
		}
		c, ok := def.Column(name)
		if !ok {
			return nil, fmt.Errorf("no column %s", name)
		}
		if err := check(s, def, def.Columns[c], f); err != nil {
			return nil, fmt.Errorf("column %s: %w", name, err)
		}
		cols[i] = c
	}
	return cols, nil
}

func check(s *Schema, def TableDef, c Column, f reflect.StructField) error {
	scalar := goScalars[c.Type]
	if scalar == nil {
		return fmt.Errorf("type %s not supported", c.Type)
	}
	want := scalar
	switch {
	case c.Array:
		want = reflect.SliceOf(scalar)
	case c.Interval:
		want = reflect.ArrayOf(2, scalar)
	}
	if f.Type != want {
		return fmt.Errorf("field %s is %s, want %s", f.Name, f.Type, want)
	}
	ref, tagged := f.Tag.Lookup("ref")
	switch {
	case scalar != refType:
		if tagged {
			return fmt.Errorf("ref tag on a %s column", c.Type)
		}
		return nil
	case !tagged:
		return fmt.Errorf("field %s needs a ref tag", f.Name)
	case c.Type == "row" && ref != def.Name:
		return fmt.Errorf("row references %s, ref tag says %s", def.Name, ref)
	case c.References != nil && c.References.Table != ref:
		return fmt.Errorf("references %s, ref tag says %s", c.References.Table, ref)
	}
	if _, ok := s.Table(ref); !ok {
		return fmt.Errorf("referenced table %s not in schema", ref)
	}
	return nil
}

// set stores v, as Table.Value decodes it, in the field fv checked by fieldColumns.
func set(fv reflect.Value, v any) {
	switch x := v.(type) {
	case []any:
		s := reflect.MakeSlice(fv.Type(), len(x), len(x))
		for i, e := range x {
			set(s.Index(i), e)
		}
		fv.Set(s)
	case [2]any:
		set(fv.Index(0), x[0])
		set(fv.Index(1), x[1])
	case nil:
		fv.SetZero() // null reference
	case int:
		fv.Set(reflect.ValueOf(RefTo(x)))
	default:
		fv.Set(reflect.ValueOf(v))
	}
}
