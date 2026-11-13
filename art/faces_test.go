package art_test

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"testing/fstest"
	"unicode/utf16"

	"github.com/google/go-cmp/cmp"

	"github.com/xaroth/pbt-extract/art"
	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/dat/dattest"
	"github.com/xaroth/pbt-extract/dds/ddstest"
)

var (
	red  = color.NRGBA{R: 254, A: 254}
	blue = color.NRGBA{B: 254, A: 254}
)

func utf16le(s string) []byte {
	var b []byte
	for _, u := range utf16.Encode([]rune(s)) {
		b = binary.LittleEndian.AppendUint16(b, u)
	}
	return b
}

func charactersSchema(t *testing.T) *dat.Schema {
	t.Helper()
	s, err := dat.ParseSchema([]byte(`{"version": 7, "tables": [{"name": "Characters", "columns": [{"name": "Id", "type": "string"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// faceGame holds two characters whose faces are the two halves of an 8x4 texture: red, then blue.
func faceGame(t *testing.T) fstest.MapFS {
	t.Helper()
	def, _ := charactersSchema(t).Table("Characters")
	const tex = "Art/Textures/Interface/2D/2DArt/UIImages/Common/4K/3.dds"
	return fstest.MapFS{
		"Data/Characters.datc64": {Data: dattest.Encode(t, def,
			[]any{"Metadata/Characters/Str/Str"}, []any{"Metadata/Characters/DexInt/DexInt"})},
		"Art/UIImages1.txt": {Data: utf16le(`"Art/2DArt/UIImages/Common/4K/IconDexInt" "` + tex + `" 0 0 4 4` + "\r\n" +
			`"Art/2DArt/UIImages/Common/4K/IconStr" "` + tex + `" 4 0 8 4` + "\r\n")},
		tex: {Data: ddstest.BC7(8, 4, ddstest.Solid(red), ddstest.Solid(blue))},
	}
}

// recorder records the files read from fs.
type recorder struct {
	fs   fstest.MapFS
	read []string
}

func (r *recorder) ReadFile(name string) ([]byte, error) {
	r.read = append(r.read, name)
	return r.fs.ReadFile(name)
}

func TestClassFaces(t *testing.T) {
	t.Parallel()
	r := &recorder{fs: faceGame(t)}
	faces, err := art.ClassFaces(r, charactersSchema(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range r.read {
		if !art.Reads(p) {
			t.Errorf("ClassFaces read %s, which Reads does not declare", p)
		}
	}

	a, err := art.Build(nil, fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.AddClassFaces(faces); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"Metadata/Characters/Str/Str":       "class-str.png",
		"Metadata/Characters/DexInt/DexInt": "class-dexint.png",
	}
	if diff := cmp.Diff(want, a.Manifest.ClassFaces); diff != "" {
		t.Errorf("ClassFaces mismatch (-want +got):\n%s", diff)
	}
	if len(a.Manifest.Files) != 2 || a.Manifest.Files[0].Name != "class-dexint.png" {
		t.Errorf("Files = %v, want class-dexint.png and class-str.png", a.Manifest.Files)
	}
	for file, c := range map[string]color.NRGBA{"class-str.png": blue, "class-dexint.png": red} {
		img, err := png.Decode(bytes.NewReader(a.Files[file]))
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		if got := img.Bounds(); got != image.Rect(0, 0, 4, 4) {
			t.Errorf("%s: bounds %v, want 4x4 at the origin", file, got)
		}
		if got := color.NRGBAModel.Convert(img.At(3, 3)); got != c {
			t.Errorf("%s: (3,3) = %v, want %v", file, got, c)
		}
	}
}

func TestClassFacesErrors(t *testing.T) {
	t.Parallel()
	for name, testCase := range map[string]struct {
		list    string
		wantErr string
	}{
		"face missing": {
			list:    `"Art/2DArt/UIImages/Common/4K/IconDexInt" "Art/Textures/Interface/2D/2DArt/UIImages/Common/4K/3.dds" 0 0 4 4`,
			wantErr: "character Metadata/Characters/Str/Str: no Art/2DArt/UIImages/Common/4K/IconStr",
		},
		"face outside the texture": {
			list: `"Art/2DArt/UIImages/Common/4K/IconDexInt" "Art/Textures/Interface/2D/2DArt/UIImages/Common/4K/3.dds" 0 0 4 4` + "\n" +
				`"Art/2DArt/UIImages/Common/4K/IconStr" "Art/Textures/Interface/2D/2DArt/UIImages/Common/4K/3.dds" 4 0 12 4`,
			wantErr: "is not inside",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := faceGame(t)
			g["Art/UIImages1.txt"] = &fstest.MapFile{Data: utf16le(testCase.list)}
			_, err := art.ClassFaces(g, charactersSchema(t))
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("got error %v, want one containing %q", err, testCase.wantErr)
			}
		})
	}
}
