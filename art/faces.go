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
	"github.com/xaroth/pbt-extract/uiimages"
)

// UIImagesList is the UI atlas list that places the class faces on their textures.
const UIImagesList = "Art/UIImages1.txt"

// facePrefix plus the last part of a character id, such as Str for Metadata/Characters/Str/Str,
// names the character's face in the atlas list.
const facePrefix = "Art/2DArt/UIImages/Common/4K/Icon"

var readsPattern = regexp.MustCompile(`(?i)^(data/characters\.datc64|art/uiimages1\.txt|art/textures/interface/2d/2dart/uiimages/common/4k/[^/]+\.dds)$`)

// Reads reports whether ClassFaces may read the game file at path.
func Reads(path string) bool { return readsPattern.MatchString(path) }

// ClassFace is the face plate of a base class.
type ClassFace struct {
	// Character is the class's character id, such as Metadata/Characters/Str/Str.
	Character string
	Image     *image.NRGBA
}

type characterRow struct {
	ID string `dat:"Id"`
}

// ClassFaces reads the face of every character of the Characters table from the UI art. A
// character without a face is an error.
func ClassFaces(r ot.Reader, s *dat.Schema) ([]ClassFace, error) {
	b, err := r.ReadFile("Data/Characters.datc64")
	if err != nil {
		return nil, err
	}
	chars, err := dat.Rows[characterRow](s, "Characters", b)
	if err != nil {
		return nil, err
	}
	if b, err = r.ReadFile(UIImagesList); err != nil {
		return nil, err
	}
	list, err := uiimages.Parse(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", UIImagesList, err)
	}
	textures := map[string]*image.NRGBA{}
	faces := make([]ClassFace, 0, len(chars))
	for _, c := range chars {
		name := facePrefix + path.Base(c.ID)
		im, ok := list[name]
		if !ok {
			return nil, fmt.Errorf("character %s: no %s in %s", c.ID, name, UIImagesList)
		}
		tex, ok := textures[im.Texture]
		if !ok {
			if tex, err = readTexture(r, im.Texture); err != nil {
				return nil, fmt.Errorf("character %s: %w", c.ID, err)
			}
			textures[im.Texture] = tex
		}
		if im.Rect.Empty() || !im.Rect.In(tex.Bounds()) {
			return nil, fmt.Errorf("character %s: %s at %v is not inside %s (%v)", c.ID, name, im.Rect, im.Texture, tex.Bounds().Size())
		}
		face := image.NewNRGBA(image.Rectangle{Max: im.Rect.Size()})
		draw.Draw(face, face.Bounds(), tex, im.Rect.Min, draw.Src)
		faces = append(faces, ClassFace{Character: c.ID, Image: face})
	}
	return faces, nil
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

// faceFile is the file name of a character's face in the art directory.
func faceFile(character string) string {
	return "class-" + strings.ToLower(path.Base(character)) + ".png"
}
