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

func TestDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write(t, dir, "schema.json", `{}`)
	write(t, dir, "tree/data.json", `[]`)
	d := input.Dir(dir)
	if b, err := d.Schema(); err != nil || string(b) != "{}" {
		t.Errorf("Schema: %q, %v", b, err)
	}
	if b, err := d.TreeFile("data.json"); err != nil || string(b) != "[]" {
		t.Errorf("TreeFile: %q, %v", b, err)
	}
	if _, err := d.TreeFile("ruthless.json"); err == nil {
		t.Error("TreeFile of a missing file: got nil error")
	}
}
