package models

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"unicode/utf8"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// A ship is designed in the 3D editor as layers of primitives. Every
// primitive is defined in a unit cube centred on its position ([-0.5, 0.5] on
// each axis before scale and rotation), which is what lets BoundingRadius
// below bound any part with the same formula whatever its type.
//
// The part types are mirrored in ships-vue-3d (src/shared/shipModel.ts):
// change both together.
const (
	PartBox      = "box"
	PartSphere   = "sphere"
	PartCylinder = "cylinder"
	PartCone     = "cone"
	PartWedge    = "wedge"
	PartTorus    = "torus"
)

var partTypes = map[string]bool{
	PartBox: true, PartSphere: true, PartCylinder: true,
	PartCone: true, PartWedge: true, PartTorus: true,
}

// Limits on what a project may contain. They keep a single design small
// enough to be sent to every player that joins a game, and rendered by a
// phone. Mirrored in ships-vue-3d's shipModel.ts so the editor can tell the
// user before the server refuses to save.
const (
	MaxProjectNameLength = 60
	MaxLayerNameLength   = 40
	MaxLayers            = 20
	MaxParts             = 400
	MaxCoordinate        = 50.0
	MinScale             = 0.01
	MaxScale             = 50.0
)

type Vec3 [3]float64

type Part struct {
	Type     string `json:"type" bson:"type"`
	Position Vec3   `json:"position" bson:"position"`
	// Rotation is an XYZ Euler angle in radians (three.js' default order).
	Rotation Vec3   `json:"rotation" bson:"rotation"`
	Scale    Vec3   `json:"scale" bson:"scale"`
	Color    string `json:"color" bson:"color"`
	// Mirror also renders the part reflected across the ship's X=0 plane,
	// so a symmetric ship only needs one side designed.
	Mirror bool `json:"mirror" bson:"mirror"`
}

type Layer3d struct {
	Name    string `json:"name" bson:"name"`
	Visible bool   `json:"visible" bson:"visible"`
	Parts   []Part `json:"parts" bson:"parts"`
}

// Project3d is a document of the `paintingProjects3d` collection. Unlike
// users and sessions it is NOT shared with the 2D game: 2D painting projects
// live in `paintingprojects` and have a different shape entirely.
type Project3d struct {
	Id           bson.ObjectID `json:"_id" bson:"_id,omitempty"`
	UserId       string        `json:"userId" bson:"userId"`
	Name         string        `json:"name" bson:"name"`
	DateCreated  int64         `json:"dateCreated" bson:"dateCreated"`
	DateModified int64         `json:"dateModified" bson:"dateModified"`
	Layers       []Layer3d     `json:"layers" bson:"layers"`
}

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Validate checks a project received from a client. It trims the names in
// place, so call it before saving.
func (p *Project3d) Validate() error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return errors.New("the project needs a name")
	}
	if utf8.RuneCountInString(p.Name) > MaxProjectNameLength {
		return fmt.Errorf("the project name can't be longer than %d characters", MaxProjectNameLength)
	}
	if len(p.Layers) == 0 {
		return errors.New("the project needs at least one layer")
	}
	if len(p.Layers) > MaxLayers {
		return fmt.Errorf("a project can't have more than %d layers", MaxLayers)
	}
	parts := 0
	for i := range p.Layers {
		layer := &p.Layers[i]
		layer.Name = strings.TrimSpace(layer.Name)
		if utf8.RuneCountInString(layer.Name) > MaxLayerNameLength {
			return fmt.Errorf("layer names can't be longer than %d characters", MaxLayerNameLength)
		}
		if layer.Parts == nil {
			layer.Parts = []Part{}
		}
		for j := range layer.Parts {
			if err := layer.Parts[j].validate(); err != nil {
				return fmt.Errorf("layer %q, part %d: %w", layer.Name, j+1, err)
			}
		}
		parts += len(layer.Parts)
	}
	if parts > MaxParts {
		return fmt.Errorf("a project can't have more than %d parts", MaxParts)
	}
	return nil
}

func (part *Part) validate() error {
	if !partTypes[part.Type] {
		return fmt.Errorf("unknown part type %q", part.Type)
	}
	if !colorPattern.MatchString(part.Color) {
		return fmt.Errorf("invalid color %q", part.Color)
	}
	for axis := 0; axis < 3; axis++ {
		if !isFinite(part.Position[axis]) || math.Abs(part.Position[axis]) > MaxCoordinate {
			return fmt.Errorf("position must be between -%g and %g", MaxCoordinate, MaxCoordinate)
		}
		if !isFinite(part.Rotation[axis]) {
			return errors.New("invalid rotation")
		}
		// Normalised so stored angles stay small however long an editor
		// session spun the part around.
		part.Rotation[axis] = math.Remainder(part.Rotation[axis], 2*math.Pi)
		if !isFinite(part.Scale[axis]) || part.Scale[axis] < MinScale || part.Scale[axis] > MaxScale {
			return fmt.Errorf("scale must be between %g and %g", MinScale, MaxScale)
		}
	}
	return nil
}

// VisibleParts returns the parts that make up the ship in game. A hidden layer
// is a work-in-progress aid in the editor, so it is left out of the ship.
func (p *Project3d) VisibleParts() []Part {
	parts := []Part{}
	for _, layer := range p.Layers {
		if layer.Visible {
			parts = append(parts, layer.Parts...)
		}
	}
	return parts
}

// BoundingRadius is the radius, around the ship's origin, of a sphere that
// contains every visible part. A part fits in its scaled unit cube, and
// rotating that cube can never move a point further from the part's centre
// than half the cube's diagonal - so |position| + |scale|/2 bounds it with no
// need to know the part's type or rotation. Mirroring flips X only, which
// leaves |position| unchanged.
func (p *Project3d) BoundingRadius() float64 {
	radius := 0.0
	for _, part := range p.VisibleParts() {
		r := length(part.Position) + length(part.Scale)/2
		if r > radius {
			radius = r
		}
	}
	return radius
}

func length(v Vec3) float64 {
	return math.Sqrt(v[0]*v[0] + v[1]*v[1] + v[2]*v[2])
}

func isFinite(f float64) bool {
	return !math.IsNaN(f) && !math.IsInf(f, 0)
}
