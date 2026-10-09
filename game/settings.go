package game

// Settings are the rules of the game. The server is their single source of
// truth: they are sent to every client in the `welcome` message, so the
// browser's flight model and hit tests can never drift from the values the
// server validates against. Units are world units, seconds and milliseconds.
type Settings struct {
	TickRate int `json:"tickRate"`
	// WorldRadius bounds the playable space: a sphere around the origin.
	WorldRadius float64 `json:"worldRadius"`
	// ShipRadius is the size every ship is normalised to, whatever its design
	// measures, and also its hit sphere. Normalising keeps a huge design from
	// being an easy target and a tiny one from being unhittable.
	ShipRadius float64 `json:"shipRadius"`
	MaxLife    float64 `json:"maxLife"`
	// Flight model, per second. Speed is along the ship's nose; reverse is
	// allowed but slower.
	MaxSpeed     float64 `json:"maxSpeed"`
	MinSpeed     float64 `json:"minSpeed"`
	Acceleration float64 `json:"acceleration"`
	PitchRate    float64 `json:"pitchRate"`
	YawRate      float64 `json:"yawRate"`
	RollRate     float64 `json:"rollRate"`
	// Weapons.
	BulletSpeed    float64 `json:"bulletSpeed"`
	BulletTtlMs    int     `json:"bulletTtlMs"`
	BulletDamage   float64 `json:"bulletDamage"`
	FireCooldownMs int     `json:"fireCooldownMs"`
	RespawnDelayMs int     `json:"respawnDelayMs"`
}

func DefaultSettings() Settings {
	return Settings{
		TickRate:       20,
		WorldRadius:    1500,
		ShipRadius:     4,
		MaxLife:        10,
		MaxSpeed:       80,
		MinSpeed:       -20,
		Acceleration:   40,
		PitchRate:      1.4,
		YawRate:        1.0,
		RollRate:       2.2,
		BulletSpeed:    300,
		BulletTtlMs:    2000,
		BulletDamage:   1,
		FireCooldownMs: 150,
		RespawnDelayMs: 3000,
	}
}
