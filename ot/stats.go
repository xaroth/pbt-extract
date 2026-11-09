package ot

import (
	"fmt"
	"slices"
	"strconv"
)

// Reader reads a game file by path. *bundle.Archive and fstest.MapFS satisfy it.
type Reader interface {
	ReadFile(name string) ([]byte, error)
}

// maxDepth bounds the extends chain so a cycle fails instead of looping.
const maxDepth = 32

// Stats returns the Stats block of the object at path (without .ot) merged over its extends
// chain. A stat set in a child replaces the parent's value.
func Stats(r Reader, path string) (map[string]int, error) {
	var chain []*File
	for p := path; p != "nothing"; {
		if len(chain) == maxDepth {
			return nil, fmt.Errorf("%s: extends chain longer than %d", path, maxDepth)
		}
		b, err := r.ReadFile(p + ".ot")
		if err != nil {
			return nil, err
		}
		f, err := Parse(b)
		if err != nil {
			return nil, fmt.Errorf("%s.ot: %w", p, err)
		}
		if f.Extends == "" {
			return nil, fmt.Errorf("%s.ot: no extends", p)
		}
		chain = append(chain, f)
		p = f.Extends
	}

	stats := map[string]int{}
	for _, c := range slices.Backward(chain) {
		b, ok := c.Block("Stats")
		if !ok {
			continue
		}
		for _, fd := range b.Fields {
			v, err := strconv.Atoi(fd.Value)
			if err != nil {
				return nil, fmt.Errorf("stat %s: %w", fd.Key, err)
			}
			stats[fd.Key] = v
		}
	}
	return stats, nil
}
