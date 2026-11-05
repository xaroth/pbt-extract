package bundle

import (
	"encoding/binary"
	"strings"
)

// PathHash returns the index key for a file path: MurmurHash64A of the lowercased path with
// seed 0x1337b33f. Patches before 3.21.2 used FNV-1a instead and are not supported.
func PathHash(path string) uint64 {
	return murmur64a([]byte(strings.ToLower(path)), 0x1337b33f)
}

func murmur64a(data []byte, seed uint64) uint64 {
	const m = 0xc6a4a7935bd1e995
	const r = 47
	h := seed ^ uint64(len(data))*m
	for len(data) >= 8 {
		k := binary.LittleEndian.Uint64(data)
		k *= m
		k ^= k >> r
		k *= m
		h ^= k
		h *= m
		data = data[8:]
	}
	if len(data) > 0 {
		var tail [8]byte
		copy(tail[:], data)
		h ^= binary.LittleEndian.Uint64(tail[:])
		h *= m
	}
	h ^= h >> r
	h *= m
	h ^= h >> r
	return h
}
