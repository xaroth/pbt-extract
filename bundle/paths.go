package bundle

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// pathRepSize is the size of one directory entry of the path section: path hash, offset and size
// of its paths in the path data, and the size of its subtree.
const pathRepSize = 20

// Paths returns the path of every file in the index, as the index spells them. The index
// stores them per directory in a bundle of its own, so the first call decompresses it.
func (a *Archive) Paths() ([]string, error) {
	b := a.paths
	if len(b) < 4 {
		return nil, fmt.Errorf("path section: %w", errShort)
	}
	n := binary.LittleEndian.Uint32(b)
	b = b[4:]
	if uint64(len(b)) < uint64(n)*pathRepSize {
		return nil, fmt.Errorf("path section: %w", errShort)
	}
	reps, rest := b[:n*pathRepSize], b[n*pathRepSize:]
	data, err := Decompress(rest, a.d)
	if err != nil {
		return nil, fmt.Errorf("decompressing path data: %w", err)
	}
	var out []string
	for i := range int(n) {
		off := binary.LittleEndian.Uint32(reps[i*pathRepSize+8:])
		size := binary.LittleEndian.Uint32(reps[i*pathRepSize+12:])
		if uint64(off)+uint64(size) > uint64(len(data)) {
			return nil, fmt.Errorf("directory %d: paths [%d, %d) past the path data (%d bytes)", i, off, off+size, len(data))
		}
		ps, err := decodePaths(data[off : off+size])
		if err != nil {
			return nil, fmt.Errorf("directory %d: %w", i, err)
		}
		out = append(out, ps...)
	}
	return out, nil
}

// decodePaths decodes one directory's path data: a sequence of (code, string) pairs. Code 0
// toggles between a section of base strings and a section of paths; any other code is 1 plus the
// index of an earlier base string the string extends, or past them for a string on its own.
func decodePaths(b []byte) ([]string, error) {
	var out, bases []string
	base := false
	for len(b) >= 4 {
		code := binary.LittleEndian.Uint32(b)
		b = b[4:]
		if code == 0 {
			base = !base
			if base {
				bases = bases[:0]
			}
			continue
		}
		z := bytes.IndexByte(b, 0)
		if z < 0 {
			return nil, fmt.Errorf("path string without terminator: %w", errShort)
		}
		s := string(b[:z])
		b = b[z+1:]
		if i := int(code - 1); i < len(bases) {
			s = bases[i] + s
		}
		if base {
			bases = append(bases, s)
		} else {
			out = append(out, s)
		}
	}
	return out, nil
}
