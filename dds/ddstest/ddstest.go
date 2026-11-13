// Package ddstest writes BC7 DDS files for tests.
package ddstest

import (
	"encoding/binary"
	"image/color"
)

// BC7 returns a DDS file of width by height holding blocks, in row order, as its only mip level.
func BC7(width, height int, blocks ...[]byte) []byte {
	b := make([]byte, 148, 148+16*len(blocks))
	copy(b, "DDS ")
	le := binary.LittleEndian
	le.PutUint32(b[4:], 124)
	le.PutUint32(b[8:], 0x1007) // caps, height, width, pixel format
	le.PutUint32(b[12:], uint32(height))
	le.PutUint32(b[16:], uint32(width))
	le.PutUint32(b[28:], 1)
	le.PutUint32(b[76:], 32)
	le.PutUint32(b[80:], 4) // fourCC
	copy(b[84:], "DX10")
	le.PutUint32(b[128:], 98) // BC7_UNORM
	le.PutUint32(b[132:], 3)  // 2D texture
	le.PutUint32(b[140:], 1)
	for _, blk := range blocks {
		b = append(b, blk...)
	}
	return b
}

// Solid returns a BC7 block whose 16 pixels are all c: mode 6 with both endpoints c and all
// indices 0. The channels of c must share their lowest bit, as mode 6 keeps one per endpoint.
func Solid(c color.NRGBA) []byte {
	if low := c.R & 1; c.G&1 != low || c.B&1 != low || c.A&1 != low {
		panic("ddstest.Solid: channels of c differ in their lowest bit")
	}
	var lo, hi uint64
	pos := 0
	put := func(v uint64, n int) {
		if pos < 64 {
			lo |= v << pos
			if pos+n > 64 {
				hi |= v >> (64 - pos)
			}
		} else {
			hi |= v << (pos - 64)
		}
		pos += n
	}
	put(1<<6, 7)
	for _, ch := range []uint8{c.R, c.G, c.B, c.A} {
		put(uint64(ch>>1), 7)
		put(uint64(ch>>1), 7)
	}
	put(uint64(c.R&1), 1) // endpoint p-bits, shared by every channel
	put(uint64(c.R&1), 1)
	b := make([]byte, 16)
	binary.LittleEndian.PutUint64(b, lo)
	binary.LittleEndian.PutUint64(b[8:], hi)
	return b
}
