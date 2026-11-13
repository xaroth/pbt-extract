package dds

import (
	"encoding/binary"
	"math/bits"
)

// bc7Mode describes one of the eight BC7 block layouts. Field names follow the BC7 format
// specification.
type bc7Mode struct {
	subsets, partitionBits, rotationBits, indexSelBits int
	colorBits, alphaBits                               int
	endpointPBits, sharedPBits                         bool
	indexBits, indexBits2                              int
}

var bc7Modes = [8]bc7Mode{
	{subsets: 3, partitionBits: 4, colorBits: 4, endpointPBits: true, indexBits: 3},
	{subsets: 2, partitionBits: 6, colorBits: 6, sharedPBits: true, indexBits: 3},
	{subsets: 3, partitionBits: 6, colorBits: 5, indexBits: 2},
	{subsets: 2, partitionBits: 6, colorBits: 7, endpointPBits: true, indexBits: 2},
	{subsets: 1, rotationBits: 2, indexSelBits: 1, colorBits: 5, alphaBits: 6, indexBits: 2, indexBits2: 3},
	{subsets: 1, rotationBits: 2, colorBits: 7, alphaBits: 8, indexBits: 2, indexBits2: 2},
	{subsets: 1, colorBits: 7, alphaBits: 7, endpointPBits: true, indexBits: 4},
	{subsets: 2, partitionBits: 6, colorBits: 5, alphaBits: 5, endpointPBits: true, indexBits: 2},
}

// Subset of each pixel per partition: one bit per pixel for two subsets, two bits for three.
var (
	partitions2 = [64]uint16{
		0xcccc, 0x8888, 0xeeee, 0xecc8, 0xc880, 0xfeec, 0xfec8, 0xec80, 0xc800, 0xffec,
		0xfe80, 0xe800, 0xffe8, 0xff00, 0xfff0, 0xf000, 0xf710, 0x008e, 0x7100, 0x08ce,
		0x008c, 0x7310, 0x3100, 0x8cce, 0x088c, 0x3110, 0x6666, 0x366c, 0x17e8, 0x0ff0,
		0x718e, 0x399c, 0xaaaa, 0xf0f0, 0x5a5a, 0x33cc, 0x3c3c, 0x55aa, 0x9696, 0xa55a,
		0x73ce, 0x13c8, 0x324c, 0x3bdc, 0x6996, 0xc33c, 0x9966, 0x0660, 0x0272, 0x04e4,
		0x4e40, 0x2720, 0xc936, 0x936c, 0x39c6, 0x639c, 0x9336, 0x9cc6, 0x817e, 0xe718,
		0xccf0, 0x0fcc, 0x7744, 0xee22,
	}
	partitions3 = [64]uint32{
		0xaa685050, 0x6a5a5040, 0x5a5a4200, 0x5450a0a8, 0xa5a50000, 0xa0a05050, 0x5555a0a0,
		0x5a5a5050, 0xaa550000, 0xaa555500, 0xaaaa5500, 0x90909090, 0x94949494, 0xa4a4a4a4,
		0xa9a59450, 0x2a0a4250, 0xa5945040, 0x0a425054, 0xa5a5a500, 0x55a0a0a0, 0xa8a85454,
		0x6a6a4040, 0xa4a45000, 0x1a1a0500, 0x0050a4a4, 0xaaa59090, 0x14696914, 0x69691400,
		0xa08585a0, 0xaa821414, 0x50a4a450, 0x6a5a0200, 0xa9a58000, 0x5090a0a8, 0xa8a09050,
		0x24242424, 0x00aa5500, 0x24924924, 0x24499224, 0x50a50a50, 0x500aa550, 0xaaaa4444,
		0x66660000, 0xa5a0a5a0, 0x50a050a0, 0x69286928, 0x44aaaa44, 0x66666600, 0xaa444444,
		0x54a854a8, 0x95809580, 0x96969600, 0xa85454a8, 0x80959580, 0xaa141414, 0x96960000,
		0xaaaa1414, 0xa05050a0, 0xa0a5a5a0, 0x96000000, 0x40804080, 0xa9a8a9a8, 0xaaaaaa44,
		0x2a4a5254,
	}
)

// Anchor pixel of the second subset of two, and of the second and third subsets of three. The
// first subset's anchor is pixel 0.
var (
	anchors2 = [64]uint8{
		15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15,
		15, 2, 8, 2, 2, 8, 8, 15, 2, 8, 2, 2, 8, 8, 2, 2,
		15, 15, 6, 8, 2, 8, 15, 15, 2, 8, 2, 2, 2, 15, 15, 6,
		6, 2, 6, 8, 15, 15, 2, 2, 15, 15, 15, 15, 15, 2, 2, 15,
	}
	anchors3a = [64]uint8{
		3, 3, 15, 15, 8, 3, 15, 15, 8, 8, 6, 6, 6, 5, 3, 3,
		3, 3, 8, 15, 3, 3, 6, 10, 5, 8, 8, 6, 8, 5, 15, 15,
		8, 15, 3, 5, 6, 10, 8, 15, 15, 3, 15, 5, 15, 15, 15, 15,
		3, 15, 5, 5, 5, 8, 5, 10, 5, 10, 8, 13, 15, 12, 3, 3,
	}
	anchors3b = [64]uint8{
		15, 8, 8, 3, 15, 15, 3, 8, 15, 15, 15, 15, 15, 15, 15, 8,
		15, 8, 15, 3, 15, 8, 15, 8, 3, 15, 6, 10, 15, 15, 10, 8,
		15, 3, 15, 10, 10, 8, 9, 10, 6, 15, 8, 15, 3, 6, 6, 8,
		15, 3, 15, 15, 15, 15, 15, 15, 15, 15, 15, 15, 3, 15, 15, 8,
	}
)

// Interpolation weights by index bits.
var bc7Weights = [5][]int{
	2: {0, 21, 43, 64},
	3: {0, 9, 18, 27, 37, 46, 55, 64},
	4: {0, 4, 9, 13, 17, 21, 26, 30, 34, 38, 43, 47, 51, 55, 60, 64},
}

// bitReader reads a 128-bit block from its least significant bit up.
type bitReader struct {
	lo, hi uint64
}

func (r *bitReader) read(n int) int {
	if n == 0 {
		return 0
	}
	v := r.lo & (1<<n - 1)
	r.lo = r.lo>>n | r.hi<<(64-n)
	r.hi >>= n
	return int(v)
}

// decodeBC7 decodes one 16-byte block into 16 RGBA pixels in row order. A block with no valid
// mode decodes to transparent black, as the specification says.
func decodeBC7(b []byte) (px [16][4]uint8) {
	r := bitReader{lo: binary.LittleEndian.Uint64(b), hi: binary.LittleEndian.Uint64(b[8:])}
	m := bits.TrailingZeros8(b[0])
	if m >= len(bc7Modes) {
		return px
	}
	r.read(m + 1)
	mode := bc7Modes[m]
	partition := r.read(mode.partitionBits)
	rotation := r.read(mode.rotationBits)
	indexSel := r.read(mode.indexSelBits)

	var ep [3][2][4]int // subset, endpoint, channel
	for c := range 3 {
		for s := range mode.subsets {
			for e := range 2 {
				ep[s][e][c] = r.read(mode.colorBits)
			}
		}
	}
	for s := range mode.subsets {
		for e := range 2 {
			ep[s][e][3] = r.read(mode.alphaBits)
		}
	}
	colorBits, alphaBits := mode.colorBits, mode.alphaBits
	if mode.endpointPBits || mode.sharedPBits {
		var p [3][2]int
		for s := range mode.subsets {
			if mode.sharedPBits {
				p[s][0] = r.read(1)
				p[s][1] = p[s][0]
				continue
			}
			p[s][0], p[s][1] = r.read(1), r.read(1)
		}
		for s := range mode.subsets {
			for e := range 2 {
				for c := range 4 {
					ep[s][e][c] = ep[s][e][c]<<1 | p[s][e]
				}
			}
		}
		colorBits++
		if alphaBits > 0 {
			alphaBits++
		}
	}
	for s := range mode.subsets {
		for e := range 2 {
			for c := range 3 {
				ep[s][e][c] = expand(ep[s][e][c], colorBits)
			}
			if alphaBits > 0 {
				ep[s][e][3] = expand(ep[s][e][3], alphaBits)
			} else {
				ep[s][e][3] = 255
			}
		}
	}

	var subset [16]int
	for i := range subset {
		switch mode.subsets {
		case 2:
			subset[i] = int(partitions2[partition]>>i) & 1
		case 3:
			subset[i] = int(partitions3[partition]>>(2*i)) & 3
		}
	}
	anchor := func(i int) bool {
		switch {
		case i == 0:
			return true
		case mode.subsets == 2:
			return i == int(anchors2[partition])
		case mode.subsets == 3:
			return i == int(anchors3a[partition]) || i == int(anchors3b[partition])
		}
		return false
	}
	var idx, idx2 [16]int
	for i := range idx {
		n := mode.indexBits
		if anchor(i) {
			n--
		}
		idx[i] = r.read(n)
	}
	if mode.indexBits2 > 0 {
		for i := range idx2 {
			n := mode.indexBits2
			if i == 0 {
				n--
			}
			idx2[i] = r.read(n)
		}
	}

	cw, aw := bc7Weights[mode.indexBits], bc7Weights[mode.indexBits]
	ci, ai := &idx, &idx
	if mode.indexBits2 > 0 {
		aw, ai = bc7Weights[mode.indexBits2], &idx2
		if indexSel == 1 {
			cw, aw = aw, cw
			ci, ai = ai, ci
		}
	}
	for i := range px {
		e := ep[subset[i]]
		for c := range 4 {
			w, j := cw, ci
			if c == 3 {
				w, j = aw, ai
			}
			px[i][c] = uint8(((64-w[j[i]])*e[0][c] + w[j[i]]*e[1][c] + 32) >> 6)
		}
		if rotation > 0 {
			px[i][3], px[i][rotation-1] = px[i][rotation-1], px[i][3]
		}
	}
	return px
}

// expand widens an n-bit value to 8 bits by repeating its high bits.
func expand(v, n int) int {
	v <<= 8 - n
	return v | v>>n
}
