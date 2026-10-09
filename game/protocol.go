package game

// The websocket protocol. Every message, both ways, is one JSON object with a
// "type" field. It is mirrored by hand in ships-vue-3d (src/game/protocol.ts):
// change both together.
//
// Client -> server:
//   join     {shipId, name}      once, right after connecting
//   state    {p, q, v}           own position/orientation/velocity, ~20/s
//   fire     {id, p, d}          a bullet fired from p along unit vector d
//   hit      {bulletId, target}  one of our bullets hit `target`
//   respawn  {}                  after death, once respawnDelayMs has passed
//
// Server -> client:
//   welcome      {id, settings, players}  answer to join; players includes us
//   playerJoined {player}
//   playerLeft   {id}
//   snapshot     {players: [{id, p, q, v}]}  moved players, every tick
//   fire         {id, owner, p, d}           a bullet someone else fired
//   damage       {target, from, bulletId, life}
//   died         {target, from}
//   respawned    {id, life, p, q}
//   error        {message}

const (
	msgJoin    = "join"
	msgState   = "state"
	msgFire    = "fire"
	msgHit     = "hit"
	msgRespawn = "respawn"

	msgWelcome      = "welcome"
	msgPlayerJoined = "playerJoined"
	msgPlayerLeft   = "playerLeft"
	msgSnapshot     = "snapshot"
	msgDamage       = "damage"
	msgDied         = "died"
	msgRespawned    = "respawned"
	msgError        = "error"
)

type envelope struct {
	Type string `json:"type"`
}

type joinMsg struct {
	ShipId string `json:"shipId"`
	Name   string `json:"name"`
}

type stateMsg struct {
	P vec3 `json:"p"`
	Q quat `json:"q"`
	V vec3 `json:"v"`
}

type fireMsg struct {
	Id string `json:"id"`
	P  vec3   `json:"p"`
	D  vec3   `json:"d"`
}

type hitMsg struct {
	BulletId string `json:"bulletId"`
	Target   string `json:"target"`
}

// PlayerInfo is everything a client needs to show a player, sent once when
// they join (or when we join); after that only snapshots, damage and deaths
// update it.
type PlayerInfo struct {
	Id     string   `json:"id"`
	Name   string   `json:"name"`
	Guest  bool     `json:"guest"`
	Ship   ShipInfo `json:"ship"`
	Scale  float64  `json:"scale"`
	Life   float64  `json:"life"`
	Dead   bool     `json:"dead"`
	Kills  int      `json:"kills"`
	Deaths int      `json:"deaths"`
	P      vec3     `json:"p"`
	Q      quat     `json:"q"`
	V      vec3     `json:"v"`
}

type welcomeMsg struct {
	Type     string       `json:"type"`
	Id       string       `json:"id"`
	Settings Settings     `json:"settings"`
	Players  []PlayerInfo `json:"players"`
}

type playerJoinedMsg struct {
	Type   string     `json:"type"`
	Player PlayerInfo `json:"player"`
}

type playerLeftMsg struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type playerState struct {
	Id string `json:"id"`
	P  vec3   `json:"p"`
	Q  quat   `json:"q"`
	V  vec3   `json:"v"`
}

type snapshotMsg struct {
	Type    string        `json:"type"`
	Players []playerState `json:"players"`
}

type fireOutMsg struct {
	Type  string `json:"type"`
	Id    string `json:"id"`
	Owner string `json:"owner"`
	P     vec3   `json:"p"`
	D     vec3   `json:"d"`
}

type damageMsg struct {
	Type     string  `json:"type"`
	Target   string  `json:"target"`
	From     string  `json:"from"`
	BulletId string  `json:"bulletId"`
	Life     float64 `json:"life"`
}

type diedMsg struct {
	Type   string `json:"type"`
	Target string `json:"target"`
	From   string `json:"from"`
}

type respawnedMsg struct {
	Type string  `json:"type"`
	Id   string  `json:"id"`
	Life float64 `json:"life"`
	P    vec3    `json:"p"`
	Q    quat    `json:"q"`
}

type errorMsg struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}
