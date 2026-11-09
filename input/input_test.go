package input_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xaroth/pbt-extract/input"
)

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOpen(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "source.json", `{"game": "poe1", "version": "3.29.3.3", "source": {"steamManifest": "1"}}`)
	write(t, dir, "schema.json", `{}`)
	write(t, dir, "tree/data.json", `[]`)
	d, err := input.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if d.Game != "poe1" || d.Version != "3.29.3.3" || d.Source.Source["steamManifest"] != "1" {
		t.Errorf("got %+v", d.Source)
	}
	if b, err := d.Schema(); err != nil || string(b) != "{}" {
		t.Errorf("Schema: %q, %v", b, err)
	}
	if b, err := d.TreeFile("data.json"); err != nil || string(b) != "[]" {
		t.Errorf("TreeFile: %q, %v", b, err)
	}
}

func TestOpenErrors(t *testing.T) {
	t.Parallel()
	for name, content := range map[string]string{
		"no version": `{"game": "poe1"}`,
		"no game":    `{"version": "3.29.3.3"}`,
		"not json":   `{`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			write(t, dir, "source.json", content)
			if _, err := input.Open(dir); err == nil {
				t.Error("got nil error, want an error")
			}
		})
	}
	t.Run("no source.json", func(t *testing.T) {
		t.Parallel()
		if _, err := input.Open(t.TempDir()); err == nil {
			t.Error("got nil error, want an error")
		}
	})
}
