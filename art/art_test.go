package art_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/google/go-cmp/cmp"
	"github.com/xaroth/pbt-engine/dataset"

	"github.com/xaroth/pbt-extract/art"
	"github.com/xaroth/pbt-extract/treeexport"
)

type sprites = map[string]map[string]treeexport.Sprite

func sprite(file string, icons ...string) treeexport.Sprite {
	s := treeexport.Sprite{Filename: "https://example.com/image/" + file + "?abc", W: 8, H: 8, Coords: map[string]treeexport.Rect{}}
	for i, icon := range icons {
		s.Coords[icon] = treeexport.Rect{X: i, W: 1, H: 1}
	}
	return s
}

func allSprites() sprites {
	return sprites{
		"normalActive":          {"0.1": sprite("skills-0.jpg", "a.png"), "0.5": sprite("skills-1.jpg", "a.png")},
		"normalInactive":        {"0.1": sprite("skills-disabled-0.jpg", "a.png"), "0.5": sprite("skills-disabled-1.jpg", "a.png")},
		"notableActive":         {"0.1": sprite("skills-0.jpg", "n.png")},
		"notableInactive":       {"0.1": sprite("skills-disabled-0.jpg", "n.png")},
		"mastery":               {"0.1": sprite("mastery-0.png", "m.png")},
		"masteryInactive":       {"0.1": sprite("mastery-disabled-0.png", "mi.png")},
		"masteryConnected":      {"0.1": sprite("mastery-connected-0.png", "mi.png")},
		"masteryActiveSelected": {"0.1": sprite("mastery-active-selected-0.png", "ma.png")},
		"masteryActiveEffect":   {"0.1": sprite("mastery-active-effect-0.png", "me.png")},
		"groupBackground":       {"0.1": sprite("group-background-0.png", "PSGroupBackground1")},
	}
}

func assets() fstest.MapFS {
	fsys := fstest.MapFS{}
	for _, zooms := range allSprites() {
		for _, s := range zooms {
			name := s.Filename[strings.LastIndex(s.Filename, "/")+1 : strings.Index(s.Filename, "?")]
			fsys[name] = &fstest.MapFile{Data: []byte(name)}
		}
	}
	return fsys
}

func tree() *treeexport.Export {
	return &treeexport.Export{
		Tree: "Default",
		Nodes: map[string]treeexport.Node{
			"1": {Skill: 1, Icon: "a.png"},
			"2": {Skill: 2, IsNotable: true, Icon: "n.png"},
			"3": {Skill: 3, IsMastery: true, Icon: "m.png", InactiveIcon: "mi.png", ActiveIcon: "ma.png", ActiveEffectImage: "me.png"},
		},
		Groups: map[string]treeexport.Group{
			"1": {Background: &treeexport.Background{Image: "PSGroupBackground1"}},
			"2": {},
		},
	}
}

func TestBuild(t *testing.T) {
	t.Parallel()
	a, err := art.Build(sprites{
		"normalActive": {"0.5": sprite("skills-1.jpg", "a.png"), "0.1": sprite("skills-0.jpg", "a.png")},
	}, assets())
	if err != nil {
		t.Fatal(err)
	}
	want := dataset.ArtManifest{
		Files: []dataset.File{
			{Name: "skills-0.jpg", Size: 12, SHA256: "4be8d51a40418e58d2e29865e2c9912681d37285e50e5d48ab92a9109303c1a2"},
			{Name: "skills-1.jpg", Size: 12, SHA256: "fb8588ee534a591eea7659edee4974c4ad87b22f3fafb773ba78924fd07c3879"},
		},
		Sprites: map[string][]dataset.SpriteSheet{"normalActive": {
			{Zoom: 0.1, File: "skills-0.jpg", Width: 8, Height: 8, Coords: map[string]dataset.SheetRect{"a.png": {W: 1, H: 1}}},
			{Zoom: 0.5, File: "skills-1.jpg", Width: 8, Height: 8, Coords: map[string]dataset.SheetRect{"a.png": {W: 1, H: 1}}},
		}},
	}
	if diff := cmp.Diff(want, a.Manifest); diff != "" {
		t.Errorf("Manifest mismatch (-want +got):\n%s", diff)
	}

	dir := t.TempDir()
	if err := a.Write(dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, art.ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	got, err := dataset.ParseArtManifest(b)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("written manifest mismatch (-want +got):\n%s", diff)
	}
	for _, f := range want.Files {
		b, err := os.ReadFile(filepath.Join(dir, f.Name))
		if err != nil {
			t.Fatal(err)
		}
		if string(b) != f.Name {
			t.Errorf("%s: got %q, want %q", f.Name, b, f.Name)
		}
	}
}

func TestBuildChecksTrees(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		edit    func(sprites, *treeexport.Export)
		assets  fstest.MapFS
		wantErr string
	}{
		{name: "all resolve", edit: func(sprites, *treeexport.Export) {}},
		{
			name:    "icon missing at one zoom",
			edit:    func(s sprites, _ *treeexport.Export) { delete(s["normalInactive"]["0.5"].Coords, "a.png") },
			wantErr: "tree Default: a.png not in normalInactive sprites at zoom 0.5",
		},
		{
			name: "notable icon only on the normal sheets",
			edit: func(_ sprites, e *treeexport.Export) {
				n := e.Nodes["1"]
				n.IsNotable = true
				e.Nodes["1"] = n
			},
			wantErr: "a.png not in notableActive sprites",
		},
		{
			name:    "mastery kind missing",
			edit:    func(s sprites, _ *treeexport.Export) { delete(s, "masteryConnected") },
			wantErr: "no masteryConnected sprites for mi.png",
		},
		{
			name: "group background missing",
			edit: func(_ sprites, e *treeexport.Export) {
				e.Groups["2"] = treeexport.Group{Background: &treeexport.Background{Image: "PSGroupBackground9"}}
			},
			wantErr: "PSGroupBackground9 not in groupBackground sprites",
		},
		{
			name:    "sheet file missing",
			edit:    func(sprites, *treeexport.Export) {},
			assets:  fstest.MapFS{},
			wantErr: "file does not exist",
		},
		{
			name: "zoom not a number",
			edit: func(s sprites, _ *treeexport.Export) {
				s["normalActive"]["x"] = s["normalActive"]["0.1"]
			},
			wantErr: `zoom "x"`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			s, e := allSprites(), tree()
			testCase.edit(s, e)
			fsys := testCase.assets
			if fsys == nil {
				fsys = assets()
			}
			_, err := art.Build(s, fsys, e)
			if testCase.wantErr == "" {
				if err != nil {
					t.Fatalf("got %v, want no error", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Errorf("got error %v, want one containing %q", err, testCase.wantErr)
			}
		})
	}
}
