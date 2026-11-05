package ot_test

import (
	"errors"
	"io/fs"
	"testing"
	"testing/fstest"
	"unicode/utf16"

	"github.com/google/go-cmp/cmp"

	"github.com/xaroth/pbt-extract/ot"
)

func encode(s string) []byte {
	u := utf16.Encode([]rune("\ufeff" + s))
	b := make([]byte, 2*len(u))
	for i, c := range u {
		b[2*i], b[2*i+1] = byte(c), byte(c>>8)
	}
	return b
}

const sample = "version 3\r\nabstract\r\nextends \"nothing\"\r\n\r\n" +
	"Stats\r\n{\r\n\tlevel = 1\r\n\tpvp_damage_+%_final = -15 // trailing comment\r\n}\r\n" +
	"Buffs { }\r\n" +
	"Actor\r\n{\t\r\n\tbasic_action = \"Move\"\r\n\tbasic_action = \"Dance\"\r\n}\r\n" +
	"// Targetable {}\r\n" +
	"client\r\n{\r\n\tRender\r\n\t{\r\n\t\tscale = 5.0\r\n\t\tfade = true\r\n\t}\r\n}\r\n" +
	"StateMachine\r\n{\r\n\ton_open =\r\n\t{\r\n\t\tPlay( 'open' ); // it's open\r\n" +
	"\t\tIf( x, (){ Add( \"a}b\" ); } );\r\n\t}\r\n\tlabel =\r\n\t\"two\r\nlines\"\r\n}\r\n"

func TestParse(t *testing.T) {
	t.Parallel()
	got, err := ot.Parse(encode(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	want := &ot.File{
		Extends: "nothing",
		Blocks: []ot.Block{
			{Name: "Stats", Fields: []ot.Field{{Key: "level", Value: "1"}, {Key: "pvp_damage_+%_final", Value: "-15"}}},
			{Name: "Buffs"},
			{Name: "Actor", Fields: []ot.Field{{Key: "basic_action", Value: "Move"}, {Key: "basic_action", Value: "Dance"}}},
			{Name: "client", Blocks: []ot.Block{
				{Name: "Render", Fields: []ot.Field{{Key: "scale", Value: "5.0"}, {Key: "fade", Value: "true"}}},
			}},
			{Name: "StateMachine", Fields: []ot.Field{
				{Key: "on_open", Value: "Play( 'open' ); // it's open\r\n\t\tIf( x, (){ Add( \"a}b\" ); } );"},
				{Key: "label", Value: "two\r\nlines"},
			}},
		},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Parse mismatch (-want +got):\n%s", diff)
	}
}

func TestParseErrors(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name string
		src  []byte
	}{
		{name: "odd length", src: []byte{0xff, 0xfe, 0x41}},
		{name: "unclosed block", src: encode("Stats\n{\n\tlevel = 1\n")},
		{name: "block without brace", src: encode("Stats\nlevel = 1\n")},
		{name: "missing value", src: encode("Stats\n{\n\tlevel =\n}\n")},
		{name: "unterminated string", src: encode("extends \"nothing\n")},
		{name: "unquoted extends", src: encode("extends nothing\n")},
		{name: "unterminated script", src: encode("A\n{\n\ton = { Play(\n}\n")},
		{name: "stray brace", src: encode("}\n")},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := ot.Parse(testCase.src); err == nil {
				t.Errorf("Parse: got nil error, want an error")
			}
		})
	}
}

func TestStats(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"Metadata/Base.ot":       {Data: encode("extends \"nothing\"\nStats\n{\n\ta = 1\n\tb = 2\n}\n")},
		"Metadata/Mid.ot":        {Data: encode("extends \"Metadata/Base\"\nLife { }\n")},
		"Metadata/Leaf.ot":       {Data: encode("extends \"Metadata/Mid\"\nStats\n{\n\tb = -3\n\tc = 4\n}\n")},
		"Metadata/Text.ot":       {Data: encode("extends \"nothing\"\nStats\n{\n\ta = \"x\"\n}\n")},
		"Metadata/NoExtends.ot":  {Data: encode("Stats\n{\n}\n")},
		"Metadata/Loop.ot":       {Data: encode("extends \"Metadata/Loop\"\n")},
		"Metadata/BadParent.ot":  {Data: encode("extends \"Metadata/Missing\"\n")},
		"Metadata/BadSyntax.ot":  {Data: encode("Stats\n{\n")},
		"Metadata/EmptyStats.ot": {Data: encode("extends \"nothing\"\n")},
	}
	got, err := ot.Stats(fsys, "Metadata/Leaf")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if diff := cmp.Diff(map[string]int{"a": 1, "b": -3, "c": 4}, got); diff != "" {
		t.Errorf("Stats mismatch (-want +got):\n%s", diff)
	}

	got, err = ot.Stats(fsys, "Metadata/EmptyStats")
	if err != nil || len(got) != 0 {
		t.Errorf("Stats(EmptyStats): got %v, %v, want empty map and nil error", got, err)
	}

	testCases := []struct {
		name     string
		path     string
		notExist bool
	}{
		{name: "missing file", path: "Metadata/Nope", notExist: true},
		{name: "missing parent", path: "Metadata/BadParent", notExist: true},
		{name: "non-integer stat", path: "Metadata/Text"},
		{name: "no extends", path: "Metadata/NoExtends"},
		{name: "extends cycle", path: "Metadata/Loop"},
		{name: "syntax error", path: "Metadata/BadSyntax"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := ot.Stats(fsys, testCase.path)
			if err == nil {
				t.Fatalf("Stats(%s): got nil error, want an error", testCase.path)
			}
			if got := errors.Is(err, fs.ErrNotExist); got != testCase.notExist {
				t.Errorf("errors.Is(err, fs.ErrNotExist): got %v, want %v (err %v)", got, testCase.notExist, err)
			}
		})
	}
}
