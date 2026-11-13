package art_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	"github.com/xaroth/pbt-extract/treeexport"
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
	s, err := dat.ParseSchema([]byte(`{"version": 7, "tables": [{"name": "Characters", "columns": [{"name": "Id", "type": "string"}, {"name": "IntegerId", "type": "i32"}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

const faceTexture = "Art/Textures/Interface/2D/2DArt/UIImages/Common/4K/3.dds"

// faceGame holds two characters whose faces are the two halves of an 8x4 texture: red, then blue.
func faceGame(t *testing.T) fstest.MapFS {
	t.Helper()
	def, _ := charactersSchema(t).Table("Characters")
	return fstest.MapFS{
		"Data/Characters.datc64": {Data: dattest.Encode(t, def,
			[]any{"Metadata/Characters/Str/Str", int32(0)}, []any{"Metadata/Characters/DexInt/DexInt", int32(1)})},
		"Art/UIImages1.txt": {Data: utf16le(`"Art/2DArt/UIImages/Common/4K/IconDexInt" "` + faceTexture + `" 0 0 4 4` + "\r\n" +
			`"Art/2DArt/UIImages/Common/4K/IconStr" "` + faceTexture + `" 4 0 8 4` + "\r\n")},
		faceTexture: {Data: ddstest.BC7(8, 4, ddstest.Solid(red), ddstest.Solid(blue))},
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

func noWarn(t *testing.T) func(string) {
	return func(msg string) { t.Errorf("unexpected warning: %s", msg) }
}

func TestClassFaces(t *testing.T) {
	t.Parallel()
	r := &recorder{fs: faceGame(t)}
	faces, err := art.ReadFaces(r, charactersSchema(t), nil, noWarn(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range r.read {
		if !art.Reads(p) {
			t.Errorf("ReadFaces read %s, which Reads does not declare", p)
		}
	}

	a, err := art.Build(nil, fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.AddFaces(faces); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"Metadata/Characters/Str/Str":       "class-str.png",
		"Metadata/Characters/DexInt/DexInt": "class-dexint.png",
	}
	if diff := cmp.Diff(want, a.Manifest.ClassFaces); diff != "" {
		t.Errorf("ClassFaces mismatch (-want +got):\n%s", diff)
	}
	if a.Manifest.AscendancyFaces != nil {
		t.Errorf("AscendancyFaces = %v, want none", a.Manifest.AscendancyFaces)
	}
	if len(a.Manifest.Files) != 2 || a.Manifest.Files[0].Name != "class-dexint.png" {
		t.Errorf("Files = %v, want class-dexint.png and class-str.png", a.Manifest.Files)
	}
	for file, c := range map[string]color.NRGBA{"class-str.png": blue, "class-dexint.png": red} {
		checkFace(t, a, file, image.Rect(0, 0, 4, 4), c)
	}
}

func checkFace(t *testing.T, a *art.Art, file string, bounds image.Rectangle, c color.NRGBA) {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(a.Files[file]))
	if err != nil {
		t.Errorf("%s: %v", file, err)
		return
	}
	if got := img.Bounds(); got != bounds {
		t.Errorf("%s: bounds %v, want %v", file, got, bounds)
	}
	if got := color.NRGBAModel.Convert(img.At(bounds.Max.X-1, bounds.Max.Y-1)); got != c {
		t.Errorf("%s: corner = %v, want %v", file, got, c)
	}
}

func TestClassFacesErrors(t *testing.T) {
	t.Parallel()
	for name, testCase := range map[string]struct {
		list    string
		wantErr string
	}{
		"face missing": {
			list:    `"Art/2DArt/UIImages/Common/4K/IconDexInt" "` + faceTexture + `" 0 0 4 4`,
			wantErr: "character Metadata/Characters/Str/Str: no Art/2DArt/UIImages/Common/4K/IconStr",
		},
		"face outside the texture": {
			list: `"Art/2DArt/UIImages/Common/4K/IconDexInt" "` + faceTexture + `" 0 0 4 4` + "\n" +
				`"Art/2DArt/UIImages/Common/4K/IconStr" "` + faceTexture + `" 4 0 12 4`,
			wantErr: "is not inside",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g := faceGame(t)
			g["Art/UIImages1.txt"] = &fstest.MapFile{Data: utf16le(testCase.list)}
			_, err := art.ReadFaces(g, charactersSchema(t), nil, noWarn(t))
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("got error %v, want one containing %q", err, testCase.wantErr)
			}
		})
	}
}

// classes329 are the classes of the 3.29 tree export, by index, with the IntegerId's character
// base.
var classes329 = []struct {
	base  string
	class treeexport.Class
}{
	{"StrDexInt", treeexport.Class{Name: "Scion", Ascendancies: ascendancies("Ascendant", "Reliquarian", "Luminary")}},
	{"Str", treeexport.Class{Name: "Marauder", Ascendancies: ascendancies("Juggernaut", "Berserker", "Chieftain")}},
	{"Dex", treeexport.Class{Name: "Ranger", Ascendancies: ascendancies("Raider", "Deadeye", "Pathfinder")}},
	{"Int", treeexport.Class{Name: "Witch", Ascendancies: ascendancies("Occultist", "Elementalist", "Necromancer")}},
	{"StrDex", treeexport.Class{Name: "Duelist", Ascendancies: ascendancies("Slayer", "Gladiator", "Champion")}},
	{"StrInt", treeexport.Class{Name: "Templar", Ascendancies: ascendancies("Inquisitor", "Hierophant", "Guardian")}},
	{"DexInt", treeexport.Class{Name: "Shadow", Ascendancies: ascendancies("Assassin", "Trickster", "Saboteur")}},
}

func ascendancies(ids ...string) []treeexport.Ascendancy {
	out := make([]treeexport.Ascendancy, len(ids))
	for i, id := range ids {
		out[i] = treeexport.Ascendancy{ID: id, Name: id}
	}
	return out
}

// ascendancyGame holds the 3.29 characters and the faces of the 3.29 atlas list: the class
// faces and then the 21 ascendancy faces, each one 4x4 block of its own color on one texture.
// It returns the color of each face by atlas name.
func ascendancyGame(t *testing.T) (fstest.MapFS, []treeexport.Class, map[string]color.NRGBA) {
	t.Helper()
	def, _ := charactersSchema(t).Table("Characters")
	var (
		rows    [][]any
		classes []treeexport.Class
		names   []string
	)
	for i, c := range classes329 {
		rows = append(rows, []any{"Metadata/Characters/" + c.base + "/" + c.base, int32(i)})
		classes = append(classes, c.class)
		names = append(names, "Icon"+c.base)
		for _, a := range c.class.Ascendancies {
			names = append(names, "Icon"+c.base+"_"+a.ID)
		}
	}
	var (
		list   strings.Builder
		blocks [][]byte
		colors = map[string]color.NRGBA{}
	)
	for i, n := range names {
		c := color.NRGBA{R: uint8(8 * i), G: 254 - uint8(8*i), A: 254}
		colors[n] = c
		blocks = append(blocks, ddstest.Solid(c))
		fmt.Fprintf(&list, "\"Art/2DArt/UIImages/Common/4K/%s\" \"%s\" %d 0 %d 4\r\n", n, faceTexture, 4*i, 4*i+4)
	}
	return fstest.MapFS{
		"Data/Characters.datc64": {Data: dattest.Encode(t, def, rows...)},
		"Art/UIImages1.txt":      {Data: utf16le(list.String())},
		faceTexture:              {Data: ddstest.BC7(4*len(names), 4, blocks...)},
	}, classes, colors
}

func TestAscendancyFaces(t *testing.T) {
	t.Parallel()
	g, classes, colors := ascendancyGame(t)
	faces, err := art.ReadFaces(g, charactersSchema(t), classes, noWarn(t))
	if err != nil {
		t.Fatal(err)
	}
	a, err := art.Build(nil, fstest.MapFS{})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.AddFaces(faces); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"Ascendant": "ascendancy-ascendant.png", "Reliquarian": "ascendancy-reliquarian.png", "Luminary": "ascendancy-luminary.png",
		"Juggernaut": "ascendancy-juggernaut.png", "Berserker": "ascendancy-berserker.png", "Chieftain": "ascendancy-chieftain.png",
		"Raider": "ascendancy-raider.png", "Deadeye": "ascendancy-deadeye.png", "Pathfinder": "ascendancy-pathfinder.png",
		"Occultist": "ascendancy-occultist.png", "Elementalist": "ascendancy-elementalist.png", "Necromancer": "ascendancy-necromancer.png",
		"Slayer": "ascendancy-slayer.png", "Gladiator": "ascendancy-gladiator.png", "Champion": "ascendancy-champion.png",
		"Inquisitor": "ascendancy-inquisitor.png", "Hierophant": "ascendancy-hierophant.png", "Guardian": "ascendancy-guardian.png",
		"Assassin": "ascendancy-assassin.png", "Trickster": "ascendancy-trickster.png", "Saboteur": "ascendancy-saboteur.png",
	}
	if diff := cmp.Diff(want, a.Manifest.AscendancyFaces); diff != "" {
		t.Errorf("AscendancyFaces mismatch (-want +got):\n%s", diff)
	}
	if len(a.Manifest.ClassFaces) != 7 {
		t.Errorf("ClassFaces = %v, want 7", a.Manifest.ClassFaces)
	}
	// Each file holds the face its atlas name places, such as IconDex_Raider for Raider.
	for _, c := range classes329 {
		for _, asc := range c.class.Ascendancies {
			checkFace(t, a, want[asc.ID], image.Rect(0, 0, 4, 4), colors["Icon"+c.base+"_"+asc.ID])
		}
	}
}

func TestAscendancyFaceMissing(t *testing.T) {
	t.Parallel()
	g, classes, _ := ascendancyGame(t)
	b, _ := g.ReadFile("Art/UIImages1.txt")
	list := strings.ReplaceAll(string(b), string(utf16le("IconDex_Raider")), string(utf16le("IconDex_Other")))
	g["Art/UIImages1.txt"] = &fstest.MapFile{Data: []byte(list)}
	var warnings []string
	faces, err := art.ReadFaces(g, charactersSchema(t), classes, func(msg string) { warnings = append(warnings, msg) })
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "ascendancy Raider: no face") {
		t.Errorf("warnings = %q, want one for Raider", warnings)
	}
	if len(faces.Ascendancies) != 20 {
		t.Errorf("%d ascendancy faces, want 20", len(faces.Ascendancies))
	}
}

func TestAscendancyClassWithoutCharacter(t *testing.T) {
	t.Parallel()
	g, classes, _ := ascendancyGame(t)
	classes = append(classes, treeexport.Class{Name: "Extra"})
	_, err := art.ReadFaces(g, charactersSchema(t), classes, noWarn(t))
	if err == nil || !strings.Contains(err.Error(), "class 7 (Extra) has no Characters row") {
		t.Errorf("got error %v, want one for class 7", err)
	}
}
