// Package bundle reads Path of Exile Bundles2 archives: the bundle container format and the
// _.index.bin file table that maps paths to bundles.
package bundle

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Decompressor decodes one Oodle-compressed chunk into exactly size bytes.
type Decompressor interface {
	Decompress(src []byte, size int) ([]byte, error)
}

const headerSize = 60

type header struct {
	size        int
	chunkSize   int
	chunkSizes  []int
	chunkOffset []int // into the bundle file
}

func parseHeader(b []byte) (header, error) {
	if len(b) < headerSize {
		return header{}, errors.New("bundle shorter than header")
	}
	le := binary.LittleEndian
	h := header{
		size:      int(le.Uint32(b[0:])),
		chunkSize: int(le.Uint32(b[40:])),
	}
	payload := int(le.Uint32(b[4:]))
	count := int(le.Uint32(b[36:]))
	if h.chunkSize <= 0 || count != (h.size+h.chunkSize-1)/h.chunkSize {
		return header{}, fmt.Errorf("chunk count %d does not fit size %d at %d per chunk", count, h.size, h.chunkSize)
	}
	pos := headerSize + 4*count
	if len(b) < pos {
		return header{}, errors.New("bundle shorter than chunk table")
	}
	h.chunkSizes = make([]int, count)
	h.chunkOffset = make([]int, count)
	for i := range count {
		n := int(le.Uint32(b[headerSize+4*i:]))
		h.chunkSizes[i], h.chunkOffset[i] = n, pos
		pos += n
	}
	if pos-headerSize-4*count != payload || pos > len(b) {
		return header{}, fmt.Errorf("chunk sizes sum to %d, header says %d, file has %d", pos-headerSize-4*count, payload, len(b)-headerSize-4*count)
	}
	return h, nil
}

// readRange decompresses only the chunks overlapping [off, off+n) of the bundle's content.
func readRange(b []byte, off, n int, d Decompressor) ([]byte, error) {
	h, err := parseHeader(b)
	if err != nil {
		return nil, err
	}
	if off < 0 || n < 0 || off+n > h.size {
		return nil, fmt.Errorf("range %d+%d outside bundle of %d bytes", off, n, h.size)
	}
	out := make([]byte, 0, n)
	for i := off / h.chunkSize; len(out) < n; i++ {
		start := i * h.chunkSize
		size := min(h.chunkSize, h.size-start)
		src := b[h.chunkOffset[i] : h.chunkOffset[i]+h.chunkSizes[i]]
		chunk, err := d.Decompress(src, size)
		if err != nil {
			return nil, fmt.Errorf("chunk %d: %w", i, err)
		}
		if len(chunk) != size {
			return nil, fmt.Errorf("chunk %d: decompressed %d bytes, want %d", i, len(chunk), size)
		}
		lo := max(off-start, 0)
		hi := min(off+n-start, size)
		out = append(out, chunk[lo:hi]...)
	}
	return out, nil
}

// Decompress returns the full content of a bundle file.
func Decompress(b []byte, d Decompressor) ([]byte, error) {
	h, err := parseHeader(b)
	if err != nil {
		return nil, err
	}
	return readRange(b, 0, h.size, d)
}
