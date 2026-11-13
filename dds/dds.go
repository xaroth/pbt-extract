// Package dds decodes DirectDraw Surface textures in the block formats the game's UI art uses.
package dds

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
)

const (
	headerSize     = 4 + 124
	dx10HeaderSize = 20
	fourCCOffset   = 84
	// DXGI formats; the sRGB variant holds the same bits.
	formatBC7     = 98
	formatBC7SRGB = 99
	bc7BlockSize  = 16
)

// Decode decodes the top mip level of a DDS file. It supports BC7 in a DX10 header.
func Decode(b []byte) (*image.NRGBA, error) {
	if len(b) < headerSize || string(b[:4]) != "DDS " {
		return nil, errors.New("not a DDS file")
	}
	height := int(binary.LittleEndian.Uint32(b[12:]))
	width := int(binary.LittleEndian.Uint32(b[16:]))
	if fourCC := string(b[fourCCOffset : fourCCOffset+4]); fourCC != "DX10" {
		return nil, fmt.Errorf("DDS pixel format %q not supported", fourCC)
	}
	if len(b) < headerSize+dx10HeaderSize {
		return nil, errors.New("DDS DX10 header cut short")
	}
	if f := binary.LittleEndian.Uint32(b[headerSize:]); f != formatBC7 && f != formatBC7SRGB {
		return nil, fmt.Errorf("DXGI format %d not supported", f)
	}
	data := b[headerSize+dx10HeaderSize:]
	bw, bh := (width+3)/4, (height+3)/4
	if len(data) < bw*bh*bc7BlockSize {
		return nil, fmt.Errorf("DDS %dx%d: %d bytes of BC7 data, want %d", width, height, len(data), bw*bh*bc7BlockSize)
	}
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for by := range bh {
		for bx := range bw {
			px := decodeBC7(data[(by*bw+bx)*bc7BlockSize:])
			for i, p := range px {
				x, y := bx*4+i%4, by*4+i/4
				if x < width && y < height {
					copy(img.Pix[img.PixOffset(x, y):], p[:])
				}
			}
		}
	}
	return img, nil
}
