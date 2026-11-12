// Package treeexport reads the official passive tree export (data.json in
// https://github.com/grindinggear/skilltree-export).
package treeexport

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// Export is the part of a tree export that datasets use.
type Export struct {
	Tree                  string          `json:"tree"`
	Classes               []Class         `json:"classes"`
	AlternateAscendancies []Ascendancy    `json:"alternate_ascendancies"`
	Nodes                 map[string]Node `json:"-"`
	JewelSlots            []uint32        `json:"jewelSlots"`
	Points                struct {
		Total      int32 `json:"totalPoints"`
		Ascendancy int32 `json:"ascendancyPoints"`
	} `json:"points"`
	Groups    map[string]Group `json:"groups"`
	Constants struct {
		SkillsPerOrbit []uint32  `json:"skillsPerOrbit"`
		OrbitRadii     []float64 `json:"orbitRadii"`
	} `json:"constants"`
	// ExtraImages are the class illustrations, keyed by class index.
	ExtraImages map[string]ExtraImage `json:"extraImages"`
	MinX        float64               `json:"min_x"`
	MinY        float64               `json:"min_y"`
	MaxX        float64               `json:"max_x"`
	MaxY        float64               `json:"max_y"`
}

// Group is a cluster of nodes drawn around one centre.
type Group struct {
	X          float64     `json:"x"`
	Y          float64     `json:"y"`
	Orbits     []uint8     `json:"orbits"`
	Background *Background `json:"background"`
	IsProxy    bool        `json:"isProxy"`
}

// Background is a group's background art.
type Background struct {
	Image       string  `json:"image"`
	IsHalfImage bool    `json:"isHalfImage"`
	OffsetX     float64 `json:"offsetX"`
	OffsetY     float64 `json:"offsetY"`
}

// ExtraImage is an illustration drawn at a point of the tree.
type ExtraImage struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Image string  `json:"image"`
}

// Class is a character class entry; its index is the class id.
type Class struct {
	Name             string       `json:"name"`
	BaseStrength     int32        `json:"base_str"`
	BaseDexterity    int32        `json:"base_dex"`
	BaseIntelligence int32        `json:"base_int"`
	Ascendancies     []Ascendancy `json:"ascendancies"`
}

// Ascendancy is an ascendancy or alternate ascendancy.
type Ascendancy struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Node is a passive tree node.
type Node struct {
	Skill                uint32          `json:"skill"`
	Name                 string          `json:"name"`
	Stats                []string        `json:"stats"`
	AscendancyName       string          `json:"ascendancyName"`
	ClassStartIndex      *int            `json:"classStartIndex"`
	IsNotable            bool            `json:"isNotable"`
	IsKeystone           bool            `json:"isKeystone"`
	IsMastery            bool            `json:"isMastery"`
	IsJewelSocket        bool            `json:"isJewelSocket"`
	IsAscendancyStart    bool            `json:"isAscendancyStart"`
	IsProxy              bool            `json:"isProxy"`
	IsMultipleChoice     bool            `json:"isMultipleChoice"`
	IsMultipleChoiceOpt  bool            `json:"isMultipleChoiceOption"`
	MasteryEffects       []MasteryEffect `json:"masteryEffects"`
	Out                  []string        `json:"out"`
	In                   []string        `json:"in"`
	GrantedPassivePoints int32           `json:"grantedPassivePoints"`
	// Group is 0 for a node not drawn on the tree; group keys start at 1.
	Group             uint32 `json:"group"`
	Orbit             uint8  `json:"orbit"`
	OrbitIndex        uint16 `json:"orbitIndex"`
	Icon              string `json:"icon"`
	InactiveIcon      string `json:"inactiveIcon"`
	ActiveIcon        string `json:"activeIcon"`
	ActiveEffectImage string `json:"activeEffectImage"`
}

// MasteryEffect is one selectable mastery effect.
type MasteryEffect struct {
	Effect uint32   `json:"effect"`
	Stats  []string `json:"stats"`
}

// Parse decodes an export. The "root" node, which only lists class starts, is dropped.
func Parse(b []byte) (*Export, error) {
	var raw struct {
		Export
		Nodes map[string]json.RawMessage `json:"nodes"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	e := raw.Export
	e.Nodes = make(map[string]Node, len(raw.Nodes))
	for id, rm := range raw.Nodes {
		if id == "root" {
			continue
		}
		var n Node
		if err := json.Unmarshal(rm, &n); err != nil {
			return nil, fmt.Errorf("node %s: %w", id, err)
		}
		if strconv.FormatUint(uint64(n.Skill), 10) != id {
			return nil, fmt.Errorf("node %s: skill %d differs from its key", id, n.Skill)
		}
		e.Nodes[id] = n
	}
	if len(e.Nodes) == 0 {
		return nil, fmt.Errorf("no nodes")
	}
	return &e, nil
}
