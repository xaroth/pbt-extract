package dat_test

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/dat/dattest"
)

const rowsSchemaJSON = `{"version": 7, "tables": [
	{"name": "T", "columns": [
		{"name": "Id", "type": "string"},
		{"name": "Flag", "type": "bool"},
		{"name": "Small", "type": "i16"},
		{"name": "USmall", "type": "u16"},
		{"name": "Int", "type": "i32"},
		{"name": "UInt", "type": "u32"},
		{"name": "Float", "type": "f32"},
		{"name": "Self", "type": "row"},
		{"name": "Other", "type": "foreignrow", "references": {"table": "U"}},
		{"name": "Loose", "type": "foreignrow"},
		{"name": "Kind", "type": "enumrow"},
		{"name": "Range", "type": "i32", "interval": true},
		{"name": "Ints", "type": "i32", "array": true},
		{"name": "Others", "type": "foreignrow", "array": true, "references": {"table": "U"}},
		{"name": "Unknown", "type": "array", "array": true},
		{"name": null, "type": "i32"}
	]},
	{"name": "U", "columns": [{"name": "Id", "type": "string"}]}
]}`

type tRow struct {
	ID     string    `dat:"Id"`
	Flag   bool      `dat:"Flag"`
	Small  int16     `dat:"Small"`
	USmall uint16    `dat:"USmall"`
	Int    int32     `dat:"Int"`
	UInt   uint32    `dat:"UInt"`
	Float  float32   `dat:"Float"`
	Self   dat.Ref   `dat:"Self" ref:"T"`
	Other  dat.Ref   `dat:"Other" ref:"U"`
	Loose  dat.Ref   `dat:"Loose" ref:"U"`
	Kind   uint32    `dat:"Kind"`
	Range  [2]int32  `dat:"Range"`
	Ints   []int32   `dat:"Ints"`
	Others []dat.Ref `dat:"Others" ref:"U"`
}

func rowsSchema(t *testing.T) *dat.Schema {
	t.Helper()
	s, err := dat.ParseSchema([]byte(rowsSchemaJSON))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func encodeT(t *testing.T, s *dat.Schema, rows ...[]any) []byte {
	t.Helper()
	def, _ := s.Table("T")
	return dattest.Encode(t, def, rows...)
}

func TestRows(t *testing.T) {
	t.Parallel()
	s := rowsSchema(t)
	b := encodeT(t, s,
		[]any{"first", true, int16(-2), uint16(65000), int32(-7), uint32(4000000000), float32(1.5),
			1, 0, 3, uint32(2), [2]int32{10, 20}, []int32{1, -2}, []int{0, dattest.Null}, 4, int32(9)},
		[]any{"second", false, int16(0), uint16(0), int32(0), uint32(0), float32(0),
			dattest.Null, dattest.Null, dattest.Null, uint32(0), [2]int32{}, []int32(nil), []int(nil), 0, int32(0)},
	)
	got, err := dat.Rows[tRow](s, "T", b)
	if err != nil {
		t.Fatal(err)
	}
	want := []tRow{
		{ID: "first", Flag: true, Small: -2, USmall: 65000, Int: -7, UInt: 4000000000, Float: 1.5,
			Self: dat.RefTo(1), Other: dat.RefTo(0), Loose: dat.RefTo(3), Kind: 2, Range: [2]int32{10, 20},
			Ints: []int32{1, -2}, Others: []dat.Ref{dat.RefTo(0), {}}},
		{ID: "second", Ints: []int32{}, Others: []dat.Ref{}},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("(-want +got)\n%s", diff)
	}
}

func TestRowsMissingColumns(t *testing.T) {
	t.Parallel()
	s := rowsSchema(t)
	def, _ := s.Table("T")
	b := dattest.Encode(t, dat.TableDef{Name: "T", Columns: def.Columns[:2]}, []any{"first", true})
	type idRow struct {
		ID string `dat:"Id"`
	}
	got, err := dat.Rows[idRow](s, "T", b)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff([]idRow{{ID: "first"}}, got); diff != "" {
		t.Errorf("(-want +got)\n%s", diff)
	}
	_, err = dat.Rows[struct {
		Int int32 `dat:"Int"`
	}](s, "T", b)
	if want := "column Int not in the file, whose rows hold the first 2 columns"; err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("got error %v, want one containing %q", err, want)
	}
	type optionalRow struct {
		ID  string `dat:"Id"`
		Int int32  `dat:"Int,optional"`
	}
	opt, err := dat.Rows[optionalRow](s, "T", b)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff([]optionalRow{{ID: "first"}}, opt); diff != "" {
		t.Errorf("optional column (-want +got)\n%s", diff)
	}
	_, err = dat.Rows[struct {
		Int int32 `dat:"Int,later"`
	}](s, "T", b)
	if want := `unknown dat tag option "later"`; err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("got error %v, want one containing %q", err, want)
	}
}

func TestRef(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		ref     dat.Ref
		wantRow int
		wantOK  bool
	}{
		{"zero is null", dat.Ref{}, 0, false},
		{"row 0", dat.RefTo(0), 0, true},
		{"row 7", dat.RefTo(7), 7, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			row, ok := testCase.ref.Row()
			if row != testCase.wantRow || ok != testCase.wantOK {
				t.Errorf("got %d, %v, want %d, %v", row, ok, testCase.wantRow, testCase.wantOK)
			}
		})
	}
}

func TestRowsErrors(t *testing.T) {
	t.Parallel()
	s := rowsSchema(t)
	good := encodeT(t, s, []any{"id", false, int16(0), uint16(0), int32(0), uint32(0), float32(0),
		dattest.Null, dattest.Null, dattest.Null, uint32(0), [2]int32{}, []int32(nil), []int(nil), 0, int32(0)})
	testCases := []struct {
		name    string
		rows    func() error
		wantMsg string
	}{
		{"unknown table", func() error { _, err := dat.Rows[tRow](s, "X", good); return err }, "table X not in schema"},
		{"not a struct", func() error { _, err := dat.Rows[int](s, "T", good); return err }, "not a struct"},
		{"unexported field", func() error {
			_, err := dat.Rows[struct {
				id string `dat:"Id"`
			}](s, "T", good)
			return err
		}, "field id is not exported"},
		{"field without tag", func() error {
			_, err := dat.Rows[struct{ ID string }](s, "T", good)
			return err
		}, "field ID has no dat tag"},
		{"unknown column", func() error {
			_, err := dat.Rows[struct {
				X string `dat:"X"`
			}](s, "T", good)
			return err
		}, "no column X"},
		{"wrong scalar type", func() error {
			_, err := dat.Rows[struct {
				Int int64 `dat:"Int"`
			}](s, "T", good)
			return err
		}, "column Int: field Int is int64, want int32"},
		{"scalar for a list", func() error {
			_, err := dat.Rows[struct {
				Ints int32 `dat:"Ints"`
			}](s, "T", good)
			return err
		}, "want []int32"},
		{"scalar for an interval", func() error {
			_, err := dat.Rows[struct {
				Range int32 `dat:"Range"`
			}](s, "T", good)
			return err
		}, "want [2]int32"},
		{"list of unknown type", func() error {
			_, err := dat.Rows[struct {
				Unknown []int `dat:"Unknown"`
			}](s, "T", good)
			return err
		}, "type array not supported"},
		{"ref without ref tag", func() error {
			_, err := dat.Rows[struct {
				Other dat.Ref `dat:"Other"`
			}](s, "T", good)
			return err
		}, "needs a ref tag"},
		{"ref tag on a scalar", func() error {
			_, err := dat.Rows[struct {
				Int int32 `dat:"Int" ref:"U"`
			}](s, "T", good)
			return err
		}, "ref tag on a i32 column"},
		{"ref to another table than the schema", func() error {
			_, err := dat.Rows[struct {
				Other dat.Ref `dat:"Other" ref:"T"`
			}](s, "T", good)
			return err
		}, "references U, ref tag says T"},
		{"row ref to another table", func() error {
			_, err := dat.Rows[struct {
				Self dat.Ref `dat:"Self" ref:"U"`
			}](s, "T", good)
			return err
		}, "row references T, ref tag says U"},
		{"ref to a table not in the schema", func() error {
			_, err := dat.Rows[struct {
				Loose dat.Ref `dat:"Loose" ref:"X"`
			}](s, "T", good)
			return err
		}, "referenced table X not in schema"},
		{"file does not fit the schema", func() error { _, err := dat.Rows[tRow](s, "T", good[:8]); return err }, "variable-data marker"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if err := testCase.rows(); err == nil || !strings.Contains(err.Error(), testCase.wantMsg) {
				t.Errorf("got error %v, want one containing %q", err, testCase.wantMsg)
			}
		})
	}
}
