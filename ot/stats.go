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

// chain returns the template at path+ext and those it extends, nearest first.
func chain(r Reader, path, ext string) ([]*File, error) {
	var files []*File
	for p := path; p != "nothing"; {
		if len(files) == maxDepth {
			return nil, fmt.Errorf("%s: extends chain longer than %d", path, maxDepth)
		}
		b, err := r.ReadFile(p + ext)
		if err != nil {
			return nil, err
		}
		f, err := Parse(b)
		if err != nil {
			return nil, fmt.Errorf("%s%s: %w", p, ext, err)
		}
		if f.Extends == "" {
			return nil, fmt.Errorf("%s%s: no extends", p, ext)
		}
		files = append(files, f)
		p = f.Extends
	}
	return files, nil
}

// Stats returns the Stats block of the object at path (without .ot) merged over its extends
// chain. A stat set in a child replaces the parent's value.
func Stats(r Reader, path string) (map[string]int, error) {
	files, err := chain(r, path, ".ot")
	if err != nil {
		return nil, err
	}
	stats := map[string]int{}
	for _, c := range slices.Backward(files) {
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

// ItemTags returns the tags of the item template at path (without .it): the tag fields of its
// Base block, then those of each template it extends, nearest first, each tag once. A
// remove_tag field drops the tag from the templates further up the chain.
func ItemTags(r Reader, path string) ([]string, error) {
	files, err := chain(r, path, ".it")
	if err != nil {
		return nil, err
	}
	var tags []string
	removed := map[string]bool{}
	for _, f := range files {
		b, _ := f.Block("Base")
		for _, fd := range b.Fields {
			if fd.Key == "tag" && !removed[fd.Value] && !slices.Contains(tags, fd.Value) {
				tags = append(tags, fd.Value)
			}
		}
		for _, fd := range b.Fields {
			if fd.Key == "remove_tag" {
				removed[fd.Value] = true
			}
		}
	}
	return tags, nil
}
