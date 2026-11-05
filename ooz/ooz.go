// Package ooz decompresses Oodle chunks by running the external ooz program
// (https://github.com/zao/ooz). ooz is GPL-3.0, so it runs as a separate process and is not
// part of this module.
package ooz

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
)

// Ooz runs the ooz binary at Path, or "ooz" from PATH when Path is empty.
type Ooz struct {
	Path string
}

// Decompress decodes src into exactly size bytes. It starts one process per call.
func (o Ooz) Decompress(src []byte, size int) ([]byte, error) {
	// ooz reads a file, not stdin: an 8-byte little-endian decompressed size, then the stream.
	f, err := os.CreateTemp("", "ooz-*.bin")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	var hdr [8]byte
	binary.LittleEndian.PutUint64(hdr[:], uint64(size))
	_, err = f.Write(append(hdr[:], src...))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return nil, err
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(cmp.Or(o.Path, "ooz"), "-d", "-c", f.Name())
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("running ooz: %w: %s", err, bytes.TrimSpace(stderr.Bytes()))
	}
	if stdout.Len() != size {
		return nil, fmt.Errorf("ooz wrote %d bytes, want %d", stdout.Len(), size)
	}
	return stdout.Bytes(), nil
}
