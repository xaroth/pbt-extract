package art

import (
	"fmt"
	"image"
	"image/draw"
	"path"
	"regexp"
	"strings"

	"github.com/xaroth/pbt-extract/dat"
	"github.com/xaroth/pbt-extract/dds"
	"github.com/xaroth/pbt-extract/ot"
	"github.com/xaroth/pbt-extract/treeexport"
	"github.com/xaroth/pbt-extract/uiimages"
)

// UIImagesList is the UI atlas list that places the faces on their textures.
const UIImagesList = "Art/UIImages1.txt"

// facePrefix plus a class's base, the last part of its character id such as Str for
// Metadata/Characters/Str/Str, names the class's face in the atlas list. An ascendancy's face
// adds _ and the ascendancy id, such as IconDex_Raider for the Warden.
const facePrefix = "Art/2DArt/UIImages/Common/4K/Icon"

var readsPattern = regexp.MustCompile(`(?i)^(data/characters\.datc64|art/uiimages1\.txt|art/textures/interface/2d/2dart/uiimages/common/4k/[^/]+\.dds)$`)

// Reads reports whether ReadFaces may read the game file at path.
func Reads(path string) bool { return readsPattern.MatchString(path) }

// Face is a face plate cut from the UI art.
type Face struct {
	// ID is a character id, such as Metadata/Characters/Str/Str, or an ascendancy id, such as
	// Raider.
	ID    string
	Image *image.NRGBA
}

// Faces holds the faces of the base classes and of their ascendancies.
type Faces struct {
	Classes      []Face
	Ascendancies []Face
}

type characterRow struct {
	ID        string `dat:"Id"`
	IntegerID int32  `dat:"IntegerId"`
}

// ReadFaces reads the face of every character of the Characters table and of every ascendancy
// of classes, the tree export's classes, whose index is a character's IntegerId. A character
// without a face is an error; an ascendancy without one is passed to warn and left out.
func ReadFaces(r ot.Reader, s *dat.Schema, classes []treeexport.Class, warn func(string)) (Faces, error) {
	b, err := r.ReadFile("Data/Characters.datc64")
	if err != nil {
		return Faces{}, err
	}
	chars, err := dat.Rows[characterRow](s, "Characters", b)
	if err != nil {
		return Faces{}, err
	}
	at, err := openAtlas(r)
	if err != nil {
		return Faces{}, err
	}
	var f Faces
	bases := make(map[int32]string, len(chars))
	for _, c := range chars {
		base := path.Base(c.ID)
		img, err := at.cut(facePrefix + base)
		if err != nil {
			return Faces{}, fmt.Errorf("character %s: %w", c.ID, err)
		}
		f.Classes = append(f.Classes, Face{ID: c.ID, Image: img})
		bases[c.IntegerID] = base
	}
	for i, c := range classes {
		base, ok := bases[int32(i)]
		if !ok {
			return Faces{}, fmt.Errorf("class %d (%s) has no Characters row with that IntegerId", i, c.Name)
		}
		for _, a := range c.Ascendancies {
			name := facePrefix + base + "_" + a.ID
			if _, listed := at.list[name]; !listed {
				warn(fmt.Sprintf("ascendancy %s: no face: no %s in %s", a.ID, name, UIImagesList))
				continue
			}
			img, err := at.cut(name)
			if err != nil {
				return Faces{}, fmt.Errorf("ascendancy %s: %w", a.ID, err)
			}
			f.Ascendancies = append(f.Ascendancies, Face{ID: a.ID, Image: img})
		}
	}
	return f, nil
}

// atlas cuts named images from the UI art, decoding each texture once.
type atlas struct {
	r        ot.Reader
	list     map[string]uiimages.Image
	textures map[string]*image.NRGBA
}

func openAtlas(r ot.Reader) (*atlas, error) {
	b, err := r.ReadFile(UIImagesList)
	if err != nil {
		return nil, err
	}
	list, err := uiimages.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", UIImagesList, err)
	}
	return &atlas{r: r, list: list, textures: map[string]*image.NRGBA{}}, nil
}

func (at *atlas) cut(name string) (*image.NRGBA, error) {
	im, ok := at.list[name]
	if !ok {
		return nil, fmt.Errorf("no %s in %s", name, UIImagesList)
	}
	tex, ok := at.textures[im.Texture]
	if !ok {
		var err error
		if tex, err = readTexture(at.r, im.Texture); err != nil {
			return nil, err
		}
		at.textures[im.Texture] = tex
	}
	if im.Rect.Empty() || !im.Rect.In(tex.Bounds()) {
		return nil, fmt.Errorf("%s at %v is not inside %s (%v)", name, im.Rect, im.Texture, tex.Bounds().Size())
	}
	img := image.NewNRGBA(image.Rectangle{Max: im.Rect.Size()})
	draw.Draw(img, img.Bounds(), tex, im.Rect.Min, draw.Src)
	return img, nil
}

func readTexture(r ot.Reader, name string) (*image.NRGBA, error) {
	b, err := r.ReadFile(name)
	if err != nil {
		return nil, err
	}
	img, err := dds.Decode(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return img, nil
}

// classFaceFile is the file name of a character's face in the art directory.
func classFaceFile(character string) string {
	return "class-" + strings.ToLower(path.Base(character)) + ".webp"
}

// ascendancyFaceFile is the file name of an ascendancy's face in the art directory.
func ascendancyFaceFile(id string) string {
	return "ascendancy-" + strings.ToLower(id) + ".webp"
}
