package extract

// Internal: no table the build reads today is optional, so Dataset cannot reach readOptionalRows.

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/dat/dattest"
)

const optionalSchema = `{"version": 7, "tables": [
	{"name": "Stats", "columns": [{"name": "Id", "type": "string"}]},
	{"name": "LeagueOnly", "columns": [{"name": "Id", "type": "string"}]}]}`

func TestReadOptionalRows(t *testing.T) {
	t.Parallel()
	s, err := dat.ParseSchema([]byte(optionalSchema))
	if err != nil {
		t.Fatal(err)
	}
	def, _ := s.Table("LeagueOnly")
	withTable := fstest.MapFS{"Data/LeagueOnly.datc64": {Data: dattest.Encode(t, def, []any{"A"}, []any{"B"})}}
	testCases := []struct {
		name         string
		game         fstest.MapFS
		table        string
		wantRows     []idRow
		wantWarnings []string
		wantErr      string
	}{
		{name: "present", game: withTable, table: "LeagueOnly", wantRows: []idRow{{ID: "A"}, {ID: "B"}}},
		{name: "absent", game: fstest.MapFS{}, table: "LeagueOnly",
			wantWarnings: []string{"table LeagueOnly: not in the game files; read as empty"}},
		{name: "table every league has", game: fstest.MapFS{}, table: "Stats", wantErr: "table Stats is listed as in every league"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var warnings []string
			rows, err := readOptionalRows[idRow](testCase.game, s, testCase.table, func(msg string) { warnings = append(warnings, msg) })
			if testCase.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
					t.Fatalf("got error %v, want one containing %q", err, testCase.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("readOptionalRows: %v", err)
			}
			if diff := cmp.Diff(testCase.wantRows, rows); diff != "" {
				t.Errorf("rows mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(testCase.wantWarnings, warnings); diff != "" {
				t.Errorf("warnings mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestReadRowsUnlistedTable(t *testing.T) {
	t.Parallel()
	s, err := dat.ParseSchema([]byte(optionalSchema))
	if err != nil {
		t.Fatal(err)
	}
	_, err = readRows[idRow](fstest.MapFS{}, s, "LeagueOnly")
	if want := "table LeagueOnly is not one every league has"; err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("got error %v, want one containing %q", err, want)
	}
}
