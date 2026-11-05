package bundle_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/xaroth/pbt-extract/bundle"
)

// stored treats every chunk as uncompressed, so tests can build bundles without Oodle.
type stored struct{ calls atomic.Int32 }

func (s *stored) Decompress(src []byte, size int) ([]byte, error) {
	s.calls.Add(1)
	if len(src) != size {
		return nil, fmt.Errorf("stored chunk is %d bytes, want %d", len(src), size)
	}
	return bytes.Clone(src), nil
}

func makeBundle(tb testing.TB, content []byte, chunkSize int) []byte {
	tb.Helper()
	count := (len(content) + chunkSize - 1) / chunkSize
	le := binary.LittleEndian
	b := make([]byte, 60, 60+4*count+len(content))
	le.PutUint32(b[0:], uint32(len(content)))
	le.PutUint32(b[4:], uint32(len(content)))
	le.PutUint32(b[36:], uint32(count))
	le.PutUint32(b[40:], uint32(chunkSize))
	for i := range count {
		b = le.AppendUint32(b, uint32(min(chunkSize, len(content)-i*chunkSize)))
	}
	return append(b, content...)
}

type entry struct {
	path                 string
	bundle, offset, size uint32
}

func makeIndex(tb testing.TB, bundles []string, files []entry) []byte {
	tb.Helper()
	le := binary.LittleEndian
	b := le.AppendUint32(nil, uint32(len(bundles)))
	for _, name := range bundles {
		b = le.AppendUint32(b, uint32(len(name)))
		b = append(b, name...)
		b = le.AppendUint32(b, 0)
	}
	b = le.AppendUint32(b, uint32(len(files)))
	for _, f := range files {
		b = le.AppendUint64(b, bundle.PathHash(f.path))
		b = le.AppendUint32(b, f.bundle)
		b = le.AppendUint32(b, f.offset)
		b = le.AppendUint32(b, f.size)
	}
	return b
}

// Bundle "a" holds 20 bytes in chunks of 8: [0,8) [8,16) [16,20).
func testArchive(tb testing.TB) fstest.MapFS {
	tb.Helper()
	index := makeIndex(tb, []string{"Folders/a", "b", "NotOnDisk"}, []entry{
		{"Data/InOneChunk.datc64", 0, 2, 4},
		{"Data/AcrossChunks.datc64", 0, 6, 4},
		{"Data/LastChunk.datc64", 0, 16, 4},
		{"Data/Whole.datc64", 0, 0, 20},
		{"Data/Empty.datc64", 0, 8, 0},
		{"Metadata/Other.ot", 1, 2, 4},
		{"Data/PastEnd.datc64", 0, 18, 4},
		{"Data/NoBundleFile.datc64", 2, 0, 1},
	})
	return fstest.MapFS{
		"Bundles2/_.index.bin":          {Data: makeBundle(tb, index, 64)},
		"Bundles2/Folders/a.bundle.bin": {Data: makeBundle(tb, []byte("0123456789abcdefghij"), 8)},
		"Bundles2/b.bundle.bin":         {Data: makeBundle(tb, []byte("xxthree"), 4)},
	}
}

func TestReadFile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name, path string
		want       string
		wantChunks int32
	}{
		{"inside one chunk", "Data/InOneChunk.datc64", "2345", 1},
		{"across a chunk boundary", "Data/AcrossChunks.datc64", "6789", 2},
		{"last partial chunk", "Data/LastChunk.datc64", "ghij", 1},
		{"whole bundle", "Data/Whole.datc64", "0123456789abcdefghij", 3},
		{"empty file", "Data/Empty.datc64", "", 0},
		{"second bundle", "Metadata/Other.ot", "thre", 2},
		{"case-insensitive path", "metadata/OTHER.OT", "thre", 2},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			d := &stored{}
			a, err := bundle.Open(testArchive(t), d)
			if err != nil {
				t.Fatal(err)
			}
			d.calls.Store(0)
			got, err := a.ReadFile(testCase.path)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
			if n := d.calls.Load(); n != testCase.wantChunks {
				t.Errorf("got %d chunks decompressed, want %d", n, testCase.wantChunks)
			}
		})
	}
}

func TestReadFileErrors(t *testing.T) {
	t.Parallel()
	a, err := bundle.Open(testArchive(t), &stored{})
	if err != nil {
		t.Fatal(err)
	}
	testCases := []struct {
		name, path  string
		wantMissing bool
	}{
		{"unknown path", "Data/Missing.datc64", true},
		{"missing bundle file", "Data/NoBundleFile.datc64", true},
		{"range past bundle end", "Data/PastEnd.datc64", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := a.ReadFile(testCase.path)
			if err == nil {
				t.Fatal("got nil error, want an error")
			}
			if got := errors.Is(err, fs.ErrNotExist); got != testCase.wantMissing {
				t.Errorf("got errors.Is(err, fs.ErrNotExist) = %v, want %v (err: %v)", got, testCase.wantMissing, err)
			}
		})
	}
}

func TestOpenErrors(t *testing.T) {
	t.Parallel()
	good := makeBundle(t, makeIndex(t, []string{"a"}, []entry{{"x", 0, 0, 1}}), 16)
	badCount := bytes.Clone(good)
	binary.LittleEndian.PutUint32(badCount[36:], 9)
	badPayload := bytes.Clone(good)
	binary.LittleEndian.PutUint32(badPayload[4:], 99)
	testCases := []struct {
		name  string
		index []byte
	}{
		{"short header", good[:10]},
		{"chunk count does not fit size", badCount},
		{"payload size mismatch", badPayload},
		{"truncated chunk data", good[:len(good)-1]},
		{"empty index", makeBundle(t, nil, 16)},
		{"truncated file table", makeBundle(t, makeIndex(t, []string{"a"}, []entry{{"x", 0, 0, 1}})[:20], 16)},
		{"unknown bundle", makeBundle(t, makeIndex(t, []string{"a"}, []entry{{"x", 1, 0, 1}}), 16)},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			fsys := fstest.MapFS{"Bundles2/_.index.bin": {Data: testCase.index}}
			if _, err := bundle.Open(fsys, &stored{}); err == nil {
				t.Error("got nil error, want an error")
			}
		})
	}
	t.Run("no index", func(t *testing.T) {
		t.Parallel()
		if _, err := bundle.Open(fstest.MapFS{}, &stored{}); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("got %v, want fs.ErrNotExist", err)
		}
	})
}
