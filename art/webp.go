package art

import (
	"bytes"
	"image"
	"image/png"
	"path"
	"strings"

	"github.com/HugoSmits86/nativewebp"
)

// webpName is the name a PNG sheet takes once written as WebP. Other sheets keep theirs: JPEG
// sheets would grow as lossless WebP, and WebP sheets already are.
func webpName(name string) string {
	if path.Ext(name) != ".png" {
		return name
	}
	return strings.TrimSuffix(name, ".png") + ".webp"
}

// encodeWebP encodes img as lossless WebP, so it decodes to the same pixels.
func encodeWebP(img image.Image) ([]byte, error) {
	var b bytes.Buffer
	if err := nativewebp.Encode(&b, img, &nativewebp.Options{CompressionLevel: nativewebp.BestCompression}); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func pngToWebP(b []byte) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return encodeWebP(img)
}
