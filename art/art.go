// Package art builds the passive tree's art from the tree export: its sprite sheets and a
// manifest that places each icon on a sheet at each zoom level.
package art

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/treeexport"
)

// ManifestName is the manifest's file name in the art directory.
const ManifestName = "manifest.json"

// Manifest lists the sheet files and, per sprite kind, where each icon is on them.
type Manifest struct {
	Files []dataset.File `json:"files"`
	// Sprites holds per kind, such as "normalActive", one sheet per zoom level, by ascending zoom.
	Sprites map[string][]Sheet `json:"sprites"`
}

// Sheet places the icons of one kind at one zoom level on File.
type Sheet struct {
	Zoom   float64                    `json:"zoom"`
	File   string                     `json:"file"`
	Width  int                        `json:"width"`
	Height int                        `json:"height"`
	Coords map[string]treeexport.Rect `json:"coords"`
}

// Art is a built manifest with the content of the files it lists.
type Art struct {
	Manifest Manifest
	Files    map[string][]byte
}

// Build reads the sheets that sprites name from assets, the export's assets folder, and checks
// that every node icon and group background of each tree resolves on the sheets of its kind at
// every zoom level. The class illustrations of a tree are not in the export's sheets.
func Build(sprites map[string]map[string]treeexport.Sprite, assets fs.FS, trees ...*treeexport.Export) (*Art, error) {
	a := &Art{Manifest: Manifest{Sprites: make(map[string][]Sheet, len(sprites))}, Files: map[string][]byte{}}
	for kind, zooms := range sprites {
		for z, s := range zooms {
			zoom, err := strconv.ParseFloat(z, 64)
			if err != nil {
				return nil, fmt.Errorf("sprite %s: zoom %q: %w", kind, z, err)
			}
			name, err := sheetName(s.Filename)
			if err != nil {
				return nil, fmt.Errorf("sprite %s at %s: %w", kind, z, err)
			}
			if _, ok := a.Files[name]; !ok {
				b, err := fs.ReadFile(assets, name)
				if err != nil {
					return nil, fmt.Errorf("sprite %s at %s: %w", kind, z, err)
				}
				a.Files[name] = b
			}
			a.Manifest.Sprites[kind] = append(a.Manifest.Sprites[kind], Sheet{Zoom: zoom, File: name, Width: s.W, Height: s.H, Coords: s.Coords})
		}
		slices.SortFunc(a.Manifest.Sprites[kind], func(x, y Sheet) int { return cmp.Compare(x.Zoom, y.Zoom) })
	}
	for _, name := range slices.Sorted(maps.Keys(a.Files)) {
		sum := sha256.Sum256(a.Files[name])
		a.Manifest.Files = append(a.Manifest.Files, dataset.File{Name: name, Size: len(a.Files[name]), SHA256: hex.EncodeToString(sum[:])})
	}
	var errs []error
	for _, t := range trees {
		errs = append(errs, a.Manifest.checkTree(t))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return a, nil
}

// sheetName is the base name of a sheet URL, which is the file's name in the assets folder.
func sheetName(filename string) (string, error) {
	u, err := url.Parse(filename)
	if err != nil {
		return "", err
	}
	name := path.Base(u.Path)
	if name == "." || name == "/" {
		return "", fmt.Errorf("sheet %q has no file name", filename)
	}
	return name, nil
}

// iconKinds returns the sprite kinds that hold a node's icon, by the node's type.
func iconKinds(n treeexport.Node) []string {
	switch {
	case n.IsKeystone:
		return []string{"keystoneActive", "keystoneInactive"}
	case n.IsNotable:
		return []string{"notableActive", "notableInactive"}
	case n.IsMastery:
		return []string{"mastery"}
	}
	return []string{"normalActive", "normalInactive"}
}

func (m Manifest) checkTree(t *treeexport.Export) error {
	var errs []error
	need := func(icon string, kinds ...string) {
		if icon == "" {
			return
		}
		for _, k := range kinds {
			if err := m.resolve(k, icon); err != nil {
				errs = append(errs, fmt.Errorf("tree %s: %w", t.Tree, err))
			}
		}
	}
	for _, id := range slices.Sorted(maps.Keys(t.Nodes)) {
		n := t.Nodes[id]
		need(n.Icon, iconKinds(n)...)
		need(n.InactiveIcon, "masteryInactive", "masteryConnected")
		need(n.ActiveIcon, "masteryActiveSelected")
		need(n.ActiveEffectImage, "masteryActiveEffect")
	}
	for _, id := range slices.Sorted(maps.Keys(t.Groups)) {
		if b := t.Groups[id].Background; b != nil {
			need(b.Image, "groupBackground")
		}
	}
	return errors.Join(errs...)
}

func (m Manifest) resolve(kind, icon string) error {
	sheets := m.Sprites[kind]
	if len(sheets) == 0 {
		return fmt.Errorf("no %s sprites for %s", kind, icon)
	}
	for _, s := range sheets {
		if _, ok := s.Coords[icon]; !ok {
			return fmt.Errorf("%s not in %s sprites at zoom %v", icon, kind, s.Zoom)
		}
	}
	return nil
}

// Write writes the sheets and the manifest into dir.
func (a *Art) Write(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, f := range a.Manifest.Files {
		if err := os.WriteFile(filepath.Join(dir, f.Name), a.Files[f.Name], 0o644); err != nil {
			return err
		}
	}
	b, err := json.MarshalIndent(a.Manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ManifestName), append(b, '\n'), 0o644)
}
