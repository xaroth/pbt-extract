package statdesc_test

import (
	"encoding/binary"
	"errors"
	"io/fs"
	"math"
	"testing"
	"testing/fstest"
	"unicode/utf16"

	"github.com/google/go-cmp/cmp"

	"github.com/xaroth/pbt-engine/statdesc"

	parser "github.com/xaroth/pbt-extract/statdesc"
)

func encode(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune("\ufeff" + s)) {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return b
}

const sample = "include \"Metadata/StatDescriptions/base.txt\"\r\n\t\t\r\nhas_identifiers\t\t\r\n\r\n" +
	"no_description hidden_stat\r\n" +
	"description\r\n" +
	"\t1 life_+%\r\n" +
	"\t2\r\n" +
	"\t\t1|# \"{0}% increased maximum Life\" reminderstring ReminderTextLife\r\n" +
	"\t\t#|-1 \"{0}% reduced maximum Life\" negate 1\r\n" +
	"\tlang \"Portuguese\"\r\n" +
	"\t2\r\n" +
	"\t\t1|1|# \"typo in another language\"\r\n" +
	"\t\t#|-1 \"vida\" negate 1\r\n" +
	"\r\n" +
	"description named_one\r\n" +
	"\t2 min_fire max_fire\r\n" +
	"\t1\r\n" +
	"\t\t# !0 \"Adds {0} to {1} Fire Damage\" canonical_line divide_by_ten_0dp 2\r\n" +
	"\r\n" +
	"handed_description weapon_fire_damage_range\r\n" +
	"\t2\r\n" +
	"main_hand_min_fire main_hand_max_fire\r\n" +
	"\t\toff_hand_min_fire off_hand_max_fire\r\n" +
	"\t1\r\n" +
	"\t\t# # \"{0} to {1} Fire Damage\"\r\n" +
	"\tlang \"French\"\r\n" +
	"\t1\r\n" +
	"\t\t# # \"{0} à {1}\"\r\n" +
	"\r\n" +
	"description\r\n" +
	"\t1 buff_duration_ms\r\n" +
	"\t2\r\n" +
	"\t\t# gem_quality \"Buff lasts {0:+d} seconds\" milliseconds_to_seconds 1\r\n" +
	"\t\t# \"Buff lasts {0} seconds\" milliseconds_to_seconds 1\r\n"

func TestParse(t *testing.T) {
	t.Parallel()
	got, err := parser.Parse("child.txt", encode(sample))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	anyValue := statdesc.Limit{Min: math.MinInt32, Max: math.MaxInt32}
	fireRange := []statdesc.Variant{{Limits: []statdesc.Limit{anyValue, anyValue}, Text: "{0} to {1} Fire Damage"}}
	want := statdesc.Scope{
		Name:   "child.txt",
		Parent: "Metadata/StatDescriptions/base.txt",
		Descriptions: []statdesc.Description{
			{Stats: []string{"hidden_stat"}},
			{
				Stats: []string{"life_+%"},
				Variants: []statdesc.Variant{
					{
						Limits:   []statdesc.Limit{{Min: 1, Max: math.MaxInt32}},
						Text:     "{0}% increased maximum Life",
						Handlers: []statdesc.Handler{{Name: "reminderstring"}},
					},
					{
						Limits:   []statdesc.Limit{{Min: math.MinInt32, Max: -1}},
						Text:     "{0}% reduced maximum Life",
						Handlers: []statdesc.Handler{{Name: "negate"}},
					},
				},
			},
			{
				Stats: []string{"min_fire", "max_fire"},
				Variants: []statdesc.Variant{{
					Limits:   []statdesc.Limit{anyValue, {Min: 0, Max: 0, Not: true}},
					Text:     "Adds {0} to {1} Fire Damage",
					Handlers: []statdesc.Handler{{Name: "canonical_line"}, {Name: "divide_by_ten_0dp", Stat: 1}},
				}},
			},
			// a handed description gives one description per hand
			{Stats: []string{"main_hand_min_fire", "main_hand_max_fire"}, Variants: fireRange},
			{Stats: []string{"off_hand_min_fire", "off_hand_max_fire"}, Variants: fireRange},
			// the gem_quality variant is left out
			{
				Stats: []string{"buff_duration_ms"},
				Variants: []statdesc.Variant{{
					Limits:   []statdesc.Limit{anyValue},
					Text:     "Buff lasts {0} seconds",
					Handlers: []statdesc.Handler{{Name: "milliseconds_to_seconds"}},
				}},
			},
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
		{name: "second include", src: encode("include \"a\"\ninclude \"b\"\n")},
		{name: "unquoted include", src: encode("include a\n")},
		{name: "unknown directive", src: encode("footed_description\n")},
		{name: "handed without stat count", src: encode("handed_description x\n")},
		{name: "handed stat count line", src: encode("handed_description x\n\t2 a b\n")},
		{name: "handed hand stat count", src: encode("handed_description x\n\t2\na b\n\tc\n\t1\n\t\t# # \"x\"\n")},
		{name: "handed bad variants", src: encode("handed_description x\n\t1\na\n\tb\n\t1\n\t\t# # \"x\"\n")},
		{name: "gem_quality before limits", src: encode("description\n\t1 a\n\t1\n\t\tgem_quality # \"x\"\n")},
		{name: "no_description without stat", src: encode("no_description\n")},
		{name: "stat count mismatch", src: encode("description\n\t2 a\n\t1\n\t\t# # \"x\"\n")},
		{name: "bad variant count", src: encode("description\n\t1 a\n\tx\n")},
		{name: "end of file in block", src: encode("description\n\t1 a\n\t2\n\t\t# \"x\"\n")},
		{name: "end of file before stats", src: encode("description\n")},
		{name: "limit count", src: encode("description\n\t1 a\n\t1\n\t\t# # \"x\"\n")},
		{name: "bad limit", src: encode("description\n\t1 a\n\t1\n\t\tx \"x\"\n")},
		{name: "bad not limit", src: encode("description\n\t1 a\n\t1\n\t\t!x \"x\"\n")},
		{name: "missing text", src: encode("description\n\t1 a\n\t1\n\t\t#\n")},
		{name: "handler stat out of range", src: encode("description\n\t1 a\n\t1\n\t\t# \"x\" negate 2\n")},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if _, err := parser.Parse("f.txt", testCase.src); err == nil {
				t.Errorf("Parse: got nil error, want an error")
			}
		})
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()
	fsys := fstest.MapFS{
		"child.txt":  {Data: encode("include \"base.txt\"\n")},
		"other.txt":  {Data: encode("include \"child.txt\"\n")},
		"base.txt":   {Data: encode("no_description x\n")},
		"loop.txt":   {Data: encode("include \"loop.txt\"\n")},
		"orphan.txt": {Data: encode("include \"gone.txt\"\n")},
		"broken.txt": {Data: encode("description\n")},
	}
	for _, c := range []struct {
		paths []string
		want  []string
	}{
		{[]string{"child.txt"}, []string{"child.txt", "base.txt"}},
		// a file reached twice is loaded once
		{[]string{"child.txt", "other.txt", "base.txt"}, []string{"child.txt", "base.txt", "other.txt"}},
		{[]string{"other.txt", "child.txt"}, []string{"other.txt", "child.txt", "base.txt"}},
	} {
		got, err := parser.Load(fsys, c.paths...)
		if err != nil {
			t.Fatalf("Load(%v): %v", c.paths, err)
		}
		var names []string
		for _, s := range got {
			names = append(names, s.Name)
		}
		if diff := cmp.Diff(c.want, names); diff != "" {
			t.Errorf("Load(%v) scopes mismatch (-want +got):\n%s", c.paths, diff)
		}
	}

	testCases := []struct {
		name     string
		path     string
		notExist bool
	}{
		{name: "cycle", path: "loop.txt"},
		{name: "missing include", path: "orphan.txt", notExist: true},
		{name: "parse error", path: "broken.txt"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := parser.Load(fsys, testCase.path)
			if err == nil {
				t.Fatalf("Load(%s): got nil error, want an error", testCase.path)
			}
			if got := errors.Is(err, fs.ErrNotExist); got != testCase.notExist {
				t.Errorf("errors.Is(err, fs.ErrNotExist): got %v, want %v (err %v)", got, testCase.notExist, err)
			}
		})
	}
}
