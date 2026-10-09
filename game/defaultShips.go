package game

import (
	"math"
	"strings"

	"ships3d/models"
)

// DefaultShipPrefix marks a built-in ship id, as opposed to the ObjectID of a
// player's own project. Guests can only fly built-in ships.
const DefaultShipPrefix = "default:"

// Ships face +Z (their nose) with +Y up. The editor shows the same axes.
var defaultShips = []models.Project3d{
	{
		Name: "Arrow",
		Layers: []models.Layer3d{{Name: "Hull", Visible: true, Parts: []models.Part{
			part(models.PartBox, v(0, 0, 0), v(0, 0, 0), v(1.2, 0.8, 4), "#9aa7b8", false),
			part(models.PartCone, v(0, 0, 2.6), v(math.Pi/2, 0, 0), v(1.2, 1.2, 1.2), "#9aa7b8", false),
			part(models.PartSphere, v(0, 0.5, 0.6), v(0, 0, 0), v(0.7, 0.5, 1.2), "#4fc3f7", false),
			part(models.PartWedge, v(1.6, 0, -0.6), v(0, 0, 0), v(2.2, 0.2, 2), "#e05a47", true),
			part(models.PartCylinder, v(0.5, 0, -2.2), v(math.Pi/2, 0, 0), v(0.5, 0.6, 0.5), "#555b66", true),
		}}},
	},
	{
		Name: "Hammer",
		Layers: []models.Layer3d{{Name: "Hull", Visible: true, Parts: []models.Part{
			part(models.PartBox, v(0, 0, 0), v(0, 0, 0), v(2.5, 1.4, 3.5), "#6d7b4d", false),
			part(models.PartBox, v(0, 0, 2.2), v(0, 0, 0), v(3.5, 1, 0.8), "#8c9a63", false),
			part(models.PartCylinder, v(1.4, 0, 2.8), v(math.Pi/2, 0, 0), v(0.3, 1.6, 0.3), "#333333", true),
			part(models.PartSphere, v(0, 0.8, 0.5), v(0, 0, 0), v(1, 0.6, 1.2), "#ffd54f", false),
			part(models.PartCylinder, v(0.8, 0, -2), v(math.Pi/2, 0, 0), v(0.8, 0.8, 0.8), "#444444", true),
		}}},
	},
	{
		Name: "Saucer",
		Layers: []models.Layer3d{{Name: "Hull", Visible: true, Parts: []models.Part{
			part(models.PartSphere, v(0, 0, 0), v(0, 0, 0), v(4, 0.9, 4), "#b0bec5", false),
			part(models.PartTorus, v(0, 0, 0), v(math.Pi/2, 0, 0), v(5, 5, 1.5), "#ff7043", false),
			part(models.PartSphere, v(0, 0.5, 0), v(0, 0, 0), v(1.6, 1.2, 1.6), "#80deea", false),
			part(models.PartCone, v(0, 0, 2.4), v(math.Pi/2, 0, 0), v(0.6, 0.8, 0.6), "#ff7043", false),
		}}},
	},
}

func v(x, y, z float64) models.Vec3 { return models.Vec3{x, y, z} }

func part(kind string, position, rotation, scale models.Vec3, color string, mirror bool) models.Part {
	return models.Part{Type: kind, Position: position, Rotation: rotation, Scale: scale, Color: color, Mirror: mirror}
}

// ShipInfo is a ship as the game sees it: an id, a name and the design.
type ShipInfo struct {
	Id     string           `json:"id"`
	Name   string           `json:"name"`
	Layers []models.Layer3d `json:"layers"`
}

func defaultShipId(project models.Project3d) string {
	return DefaultShipPrefix + strings.ToLower(project.Name)
}

// DefaultShips lists the built-in ships, offered to every player.
func DefaultShips() []ShipInfo {
	ships := make([]ShipInfo, 0, len(defaultShips))
	for _, project := range defaultShips {
		ships = append(ships, ShipInfo{Id: defaultShipId(project), Name: project.Name, Layers: project.Layers})
	}
	return ships
}

// DefaultShip returns the built-in ship with the given id.
func DefaultShip(id string) (*models.Project3d, bool) {
	for i := range defaultShips {
		if defaultShipId(defaultShips[i]) == id {
			project := defaultShips[i]
			return &project, true
		}
	}
	return nil, false
}
