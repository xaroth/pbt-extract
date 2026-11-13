package uiimages_test

import (
	"encoding/binary"
	"image"
	"testing"
	"unicode/utf16"

	"github.com/google/go-cmp/cmp"

	"github.com/xaroth/pbt-extract/uiimages"
)

func utf16le(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return b
}

func TestParse(t *testing.T) {
	t.Parallel()
	got, err := uiimages.Parse(utf16le("\ufeff\"Art/A\" \"Art/1.dds\" 0 10 181 151\r\n\"Art/B C\" \"Art/2.dds\" 5 6 7 8\r\n\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]uiimages.Image{
		"Art/A":   {Texture: "Art/1.dds", Rect: image.Rect(0, 10, 181, 151)},
		"Art/B C": {Texture: "Art/2.dds", Rect: image.Rect(5, 6, 7, 8)},
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Parse mismatch (-want +got):\n%s", diff)
	}
}

func TestParseRejects(t *testing.T) {
	t.Parallel()
	for name, text := range map[string]string{
		"missing coordinate": "\"Art/A\" \"Art/1.dds\" 0 10 181\n",
		"listed twice":       "\"Art/A\" \"Art/1.dds\" 0 0 1 1\n\"Art/A\" \"Art/1.dds\" 0 0 1 1\n",
	} {
		if _, err := uiimages.Parse(utf16le(text)); err == nil {
			t.Errorf("%s: got nil error", name)
		}
	}
}
