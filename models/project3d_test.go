package models

import (
	"math"
	"strings"
	"testing"
)

func validProject() Project3d {
	return Project3d{
		Name: "  Falcon  ",
		Layers: []Layer3d{{Name: "Hull", Visible: true, Parts: []Part{
			{Type: PartBox, Position: Vec3{0, 0, 0}, Scale: Vec3{1, 1, 1}, Color: "#aabbcc"},
		}}},
	}
}

func TestValidateAcceptsAndTrims(t *testing.T) {
	p := validProject()
	if err := p.Validate(); err != nil {
		t.Fatalf("valid project rejected: %v", err)
	}
	if p.Name != "Falcon" {
		t.Errorf("name not trimmed: %q", p.Name)
	}
}

func TestValidateRejects(t *testing.T) {
	cases := map[string]func(p *Project3d){
		"empty name":     func(p *Project3d) { p.Name = "   " },
		"long name":      func(p *Project3d) { p.Name = strings.Repeat("x", MaxProjectNameLength+1) },
		"no layers":      func(p *Project3d) { p.Layers = nil },
		"unknown type":   func(p *Project3d) { p.Layers[0].Parts[0].Type = "teapot" },
		"bad color":      func(p *Project3d) { p.Layers[0].Parts[0].Color = "red" },
		"far position":   func(p *Project3d) { p.Layers[0].Parts[0].Position[1] = MaxCoordinate + 1 },
		"zero scale":     func(p *Project3d) { p.Layers[0].Parts[0].Scale[2] = 0 },
		"NaN rotation":   func(p *Project3d) { p.Layers[0].Parts[0].Rotation[0] = math.NaN() },
		"too many parts": func(p *Project3d) { p.Layers[0].Parts = make([]Part, MaxParts+1) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := validProject()
			mutate(&p)
			if err := p.Validate(); err == nil {
				t.Error("invalid project accepted")
			}
		})
	}
}

func TestValidateNormalisesRotation(t *testing.T) {
	p := validProject()
	p.Layers[0].Parts[0].Rotation = Vec3{5 * math.Pi, 0, -3 * math.Pi}
	if err := p.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, r := range p.Layers[0].Parts[0].Rotation {
		if math.Abs(r) > math.Pi+1e-9 {
			t.Errorf("rotation %v not normalised", r)
		}
	}
}

func TestBoundingRadius(t *testing.T) {
	p := validProject()
	p.Layers[0].Parts = []Part{
		{Type: PartBox, Position: Vec3{3, 4, 0}, Scale: Vec3{2, 2, 2}, Color: "#000000"},
	}
	// |position| = 5, half diagonal of a 2x2x2 cube = sqrt(3).
	want := 5 + math.Sqrt(3)
	if got := p.BoundingRadius(); math.Abs(got-want) > 1e-9 {
		t.Errorf("BoundingRadius = %v, want %v", got, want)
	}
}

func TestBoundingRadiusIgnoresHiddenLayers(t *testing.T) {
	p := validProject()
	p.Layers = append(p.Layers, Layer3d{Visible: false, Parts: []Part{
		{Type: PartBox, Position: Vec3{40, 0, 0}, Scale: Vec3{1, 1, 1}, Color: "#000000"},
	}})
	if r := p.BoundingRadius(); r > 1 {
		t.Errorf("hidden layer counted: radius %v", r)
	}
	p.Layers[0].Visible = false
	p.Layers[1].Visible = false
	if r := p.BoundingRadius(); r != 0 {
		t.Errorf("a ship with nothing visible has radius %v, want 0", r)
	}
}
