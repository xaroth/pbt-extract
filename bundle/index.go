package bundle

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"path"
)

// Archive reads files by path from a Bundles2 directory.
type Archive struct {
	fsys    fs.FS
	d       Decompressor
	bundles []string
	files   map[uint64]fileEntry
	// paths is the index's path section, decoded by Paths.
	paths []byte
}

type fileEntry struct {
	bundle, offset, size uint32
}

// Open reads Bundles2/_.index.bin from fsys, the root of a game install or archive.
func Open(fsys fs.FS, d Decompressor) (*Archive, error) {
	raw, err := fs.ReadFile(fsys, "Bundles2/_.index.bin")
	if err != nil {
		return nil, err
	}
	idx, err := Decompress(raw, d)
	if err != nil {
		return nil, fmt.Errorf("decompressing index: %w", err)
	}
	a := &Archive{fsys: fsys, d: d}
	if err := a.parseIndex(idx); err != nil {
		return nil, fmt.Errorf("parsing index: %w", err)
	}
	return a, nil
}

var errShort = errors.New("index truncated")

func (a *Archive) parseIndex(b []byte) error {
	le := binary.LittleEndian
	u32 := func() (uint32, error) {
		if len(b) < 4 {
			return 0, errShort
		}
		v := le.Uint32(b)
		b = b[4:]
		return v, nil
	}
	nb, err := u32()
	if err != nil {
		return err
	}
	a.bundles = make([]string, nb)
	for i := range a.bundles {
		n, err := u32()
		if err != nil {
			return err
		}
		if uint32(len(b)) < n+4 {
			return errShort
		}
		a.bundles[i] = string(b[:n])
		b = b[n+4:] // skip the bundle's uncompressed size
	}
	nf, err := u32()
	if err != nil {
		return err
	}
	if uint64(len(b)) < uint64(nf)*20 {
		return errShort
	}
	a.files = make(map[uint64]fileEntry, nf)
	for range nf {
		e := fileEntry{le.Uint32(b[8:]), le.Uint32(b[12:]), le.Uint32(b[16:])}
		if e.bundle >= nb {
			return fmt.Errorf("file refers to bundle %d of %d", e.bundle, nb)
		}
		a.files[le.Uint64(b)] = e
		b = b[20:]
	}
	a.paths = b
	return nil
}

// ReadFile returns the content of the file at path, for example "Data/Stats.datc64".
// Paths are case-insensitive. It returns an error wrapping fs.ErrNotExist for unknown paths.
func (a *Archive) ReadFile(name string) ([]byte, error) {
	e, ok := a.files[PathHash(name)]
	if !ok {
		return nil, &fs.PathError{Op: "read", Path: name, Err: fs.ErrNotExist}
	}
	bundleName := a.bundles[e.bundle]
	raw, err := fs.ReadFile(a.fsys, bundlePath(bundleName))
	if err != nil {
		return nil, err
	}
	out, err := readRange(raw, int(e.offset), int(e.size), a.d)
	if err != nil {
		return nil, fmt.Errorf("reading %s from bundle %s: %w", name, bundleName, err)
	}
	return out, nil
}

// BundleFile returns the bundle file that holds the file at name, relative to the root given to
// Open (for example "Bundles2/Folders/a.bundle.bin"), and false for unknown paths.
func (a *Archive) BundleFile(name string) (string, bool) {
	e, ok := a.files[PathHash(name)]
	if !ok {
		return "", false
	}
	return bundlePath(a.bundles[e.bundle]), true
}

func bundlePath(name string) string { return path.Join("Bundles2", name+".bundle.bin") }
