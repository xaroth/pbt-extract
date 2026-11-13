// Package uiimages parses the game's UI atlas lists, such as Art/UIImages1.txt: per named image,
// the texture it is on and its rectangle there.
package uiimages

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

// Image places a named image on a texture.
type Image struct {
	Texture string
	Rect    image.Rectangle
}

var line = regexp.MustCompile(`^"([^"]+)" "([^"]+)" (\d+) (\d+) (\d+) (\d+)$`)

// Parse parses a UTF-16LE atlas list: one image per line, its quoted name, its quoted texture
// path and the rectangle x1 y1 x2 y2, x2 and y2 exclusive. Some lists hold empty rectangles.
func Parse(b []byte) (map[string]Image, error) {
	if len(b)%2 != 0 {
		return nil, errors.New("odd length for UTF-16")
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[2*i:])
	}
	text := strings.TrimPrefix(string(utf16.Decode(u)), "\ufeff")
	out := map[string]Image{}
	for i, l := range strings.Split(text, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		m := line.FindStringSubmatch(l)
		if m == nil {
			return nil, fmt.Errorf("line %d: %q is not an image", i+1, l)
		}
		var c [4]int
		for j := range c {
			v, err := strconv.Atoi(m[3+j])
			if err != nil {
				return nil, fmt.Errorf("line %d: %w", i+1, err)
			}
			c[j] = v
		}
		if _, dup := out[m[1]]; dup {
			return nil, fmt.Errorf("line %d: image %s listed twice", i+1, m[1])
		}
		out[m[1]] = Image{Texture: m[2], Rect: image.Rect(c[0], c[1], c[2], c[3])}
	}
	return out, nil
}
