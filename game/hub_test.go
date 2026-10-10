package game

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"ships3d/models"
)

// fakeClock lets a test move time forward, e.g. past the respawn delay.
type fakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

type testServer struct {
	hub   *Hub
	clock *fakeClock
	url   string
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	clock := &fakeClock{now: time.Unix(1_000_000, 0)}
	loader := func(identity Identity, shipId string) (*models.Project3d, error) {
		if ship, ok := DefaultShip(shipId); ok {
			return ship, nil
		}
		return nil, ErrShipNotAllowed
	}
	identify := func(r *http.Request) Identity {
		if name := r.URL.Query().Get("user"); name != "" {
			return Identity{UserId: "id-" + name, Username: name}
		}
		return Identity{}
	}
	hub := NewHub(DefaultSettings(), loader, identify, func(*http.Request) bool { return true })
	hub.now = clock.Now
	server := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	t.Cleanup(server.Close)
	return &testServer{hub: hub, clock: clock, url: "ws" + strings.TrimPrefix(server.URL, "http")}
}

// testClient reads on a goroutine of its own: gorilla/websocket fails a
// connection for good after one read times out, so waiting for "nothing
// arrives" can't be done with a read deadline on the test goroutine.
type testClient struct {
	t        *testing.T
	conn     *websocket.Conn
	messages chan map[string]any
}

func (s *testServer) connect(t *testing.T, query string) *testClient {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(s.url+query, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	c := &testClient{t: t, conn: conn, messages: make(chan map[string]any, 256)}
	go func() {
		defer close(c.messages)
		for {
			var msg map[string]any
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			c.messages <- msg
		}
	}()
	return c
}

func (c *testClient) send(msg map[string]any) {
	c.t.Helper()
	if err := c.conn.WriteJSON(msg); err != nil {
		c.t.Fatal(err)
	}
}

// expect waits for a message of the given type, skipping the others
// (snapshots, other players' events).
func (c *testClient) expect(msgType string) map[string]any {
	c.t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case msg, ok := <-c.messages:
			if !ok {
				c.t.Fatalf("connection closed waiting for %q", msgType)
			}
			if msg["type"] == msgType {
				return msg
			}
		case <-timeout:
			c.t.Fatalf("timed out waiting for %q", msgType)
		}
	}
}

// expectNone checks that no message of the given type arrives for a moment.
func (c *testClient) expectNone(msgType string) {
	c.t.Helper()
	timeout := time.After(200 * time.Millisecond)
	for {
		select {
		case msg, ok := <-c.messages:
			if !ok {
				return
			}
			if msg["type"] == msgType {
				c.t.Fatalf("unexpected %q: %v", msgType, msg)
			}
		case <-timeout:
			return
		}
	}
}

func (c *testClient) join(shipId, name string) string {
	c.t.Helper()
	c.send(map[string]any{"type": msgJoin, "shipId": shipId, "name": name})
	welcome := c.expect(msgWelcome)
	return welcome["id"].(string)
}

// place moves the client's ship to a known position. Messages from one
// connection are handled in order, but this one races messages from others,
// so it waits a moment for the server to store it.
func (c *testClient) place(p [3]float64) {
	c.t.Helper()
	c.send(map[string]any{"type": msgState, "p": p, "q": []float64{0, 0, 0, 1}, "v": []float64{0, 0, 0}})
	time.Sleep(50 * time.Millisecond)
}

func TestJoinAnnouncesPlayers(t *testing.T) {
	s := newTestServer(t)
	alice := s.connect(t, "?user=alice")
	aliceId := alice.join("default:arrow", "ignored")

	bob := s.connect(t, "")
	bob.send(map[string]any{"type": msgJoin, "shipId": "default:saucer", "name": "  Bob\x07  "})
	welcome := bob.expect(msgWelcome)

	players := welcome["players"].([]any)
	if len(players) != 2 {
		t.Fatalf("welcome lists %d players, want 2", len(players))
	}
	me := players[0].(map[string]any)
	if me["name"] != "Bob" || me["guest"] != true {
		t.Errorf("guest name not cleaned up: %v", me["name"])
	}
	if welcome["settings"].(map[string]any)["shipRadius"] != DefaultSettings().ShipRadius {
		t.Error("settings missing from welcome")
	}

	joined := alice.expect(msgPlayerJoined)["player"].(map[string]any)
	if joined["name"] != "Bob" {
		t.Errorf("alice saw %v join", joined["name"])
	}
	if other := players[1].(map[string]any); other["id"] != aliceId || other["name"] != "alice" {
		t.Errorf("logged-in player should use their username, got %v", other["name"])
	}
}

func TestGuestCannotFlyAProject(t *testing.T) {
	s := newTestServer(t)
	guest := s.connect(t, "")
	guest.send(map[string]any{"type": msgJoin, "shipId": "64b000000000000000000000"})
	guest.expect(msgError)
	if n := s.hub.PlayerCount(); n != 0 {
		t.Errorf("%d players after a refused join", n)
	}
}

func TestHitKillsAndRespawnWaitsForDelay(t *testing.T) {
	s := newTestServer(t)
	shooter := s.connect(t, "?user=shooter")
	shooterId := shooter.join("default:arrow", "")
	target := s.connect(t, "?user=target")
	targetId := target.join("default:hammer", "")

	shooter.place([3]float64{0, 0, 0})
	target.place([3]float64{0, 0, 100})

	settings := DefaultSettings()
	shots := int(settings.MaxLife / settings.BulletDamage)
	for i := 0; i < shots; i++ {
		s.clock.Advance(time.Duration(settings.FireCooldownMs) * time.Millisecond)
		id := "b" + string(rune('a'+i))
		shooter.send(map[string]any{"type": msgFire, "id": id, "p": []float64{0, 0, 0}, "d": []float64{0, 0, 1}})
		target.expect(msgFire)
		shooter.send(map[string]any{"type": msgHit, "bulletId": id, "target": targetId})
		damage := target.expect(msgDamage)
		if want := settings.MaxLife - float64(i+1)*settings.BulletDamage; damage["life"] != want {
			t.Fatalf("life after hit %d = %v, want %v", i+1, damage["life"], want)
		}
	}
	died := target.expect(msgDied)
	if died["target"] != targetId || died["from"] != shooterId {
		t.Fatalf("died = %v", died)
	}

	// Too early: ignored.
	target.send(map[string]any{"type": msgRespawn})
	target.expectNone(msgRespawned)

	s.clock.Advance(time.Duration(settings.RespawnDelayMs) * time.Millisecond)
	target.send(map[string]any{"type": msgRespawn})
	if respawned := target.expect(msgRespawned); respawned["life"] != settings.MaxLife {
		t.Errorf("respawned with life %v", respawned["life"])
	}
}

func TestImplausibleHitsAreIgnored(t *testing.T) {
	s := newTestServer(t)
	shooter := s.connect(t, "?user=shooter")
	shooter.join("default:arrow", "")
	target := s.connect(t, "?user=target")
	targetId := target.join("default:hammer", "")

	shooter.place([3]float64{0, 0, 0})
	// Well off to the side of a bullet fired along +Z.
	target.place([3]float64{200, 0, 100})

	s.clock.Advance(time.Second)
	shooter.send(map[string]any{"type": msgFire, "id": "x", "p": []float64{0, 0, 0}, "d": []float64{0, 0, 1}})
	target.expect(msgFire)

	// A bullet that was never fired, a bullet aimed elsewhere, and
	// shooting yourself.
	shooter.send(map[string]any{"type": msgHit, "bulletId": "never-fired", "target": targetId})
	shooter.send(map[string]any{"type": msgHit, "bulletId": "x", "target": targetId})
	target.expectNone(msgDamage)
}

func TestFireCooldownAndOriginAreEnforced(t *testing.T) {
	s := newTestServer(t)
	shooter := s.connect(t, "?user=shooter")
	shooter.join("default:arrow", "")
	watcher := s.connect(t, "?user=watcher")
	watcher.join("default:arrow", "")
	shooter.place([3]float64{0, 0, 0})

	s.clock.Advance(time.Second)
	shooter.send(map[string]any{"type": msgFire, "id": "1", "p": []float64{0, 0, 0}, "d": []float64{1, 0, 0}})
	watcher.expect(msgFire)

	// Same instant: inside the cooldown.
	shooter.send(map[string]any{"type": msgFire, "id": "2", "p": []float64{0, 0, 0}, "d": []float64{1, 0, 0}})
	watcher.expectNone(msgFire)

	// Fired from somewhere the ship can't be.
	s.clock.Advance(time.Second)
	shooter.send(map[string]any{"type": msgFire, "id": "3", "p": []float64{900, 0, 0}, "d": []float64{1, 0, 0}})
	watcher.expectNone(msgFire)
}

func TestLeaveIsAnnounced(t *testing.T) {
	s := newTestServer(t)
	a := s.connect(t, "?user=a")
	a.join("default:arrow", "")
	b := s.connect(t, "?user=b")
	bId := b.join("default:arrow", "")
	a.expect(msgPlayerJoined)

	_ = b.conn.Close()
	if left := a.expect(msgPlayerLeft); left["id"] != bId {
		t.Errorf("playerLeft for %v, want %v", left["id"], bId)
	}
}

func TestDefaultShipsAreValid(t *testing.T) {
	for _, info := range DefaultShips() {
		ship, ok := DefaultShip(info.Id)
		if !ok {
			t.Fatalf("%s not found by id", info.Id)
		}
		project := *ship
		if err := project.Validate(); err != nil {
			t.Errorf("%s: %v", info.Id, err)
		}
		if project.BoundingRadius() <= 0 {
			t.Errorf("%s has no visible parts", info.Id)
		}
	}
}

func TestProtocolFieldNames(t *testing.T) {
	// Pins the wire names ships-vue-3d reads (src/game/protocol.ts).
	frame, _ := json.Marshal(damageMsg{Type: msgDamage, Target: "t", From: "f", BulletId: "b", Life: 3})
	want := `{"type":"damage","target":"t","from":"f","bulletId":"b","life":3}`
	if string(frame) != want {
		t.Errorf("damage = %s, want %s", frame, want)
	}
}

func TestRateLimitToleratesBurstsButNotFloods(t *testing.T) {
	start := time.Unix(1_000_000, 0)
	c := &client{}

	// A stalled mobile connection catching up: hundreds of messages at once.
	for i := 0; i < 300; i++ {
		if !c.allowMessage(start) {
			t.Fatalf("burst message %d refused", i)
		}
	}

	// A normal client afterwards (30/s, half the rate) is never refused.
	now := start
	for i := 0; i < 30*60; i++ {
		now = now.Add(time.Second / 30)
		if !c.allowMessage(now) {
			t.Fatalf("normal traffic refused after %v", now.Sub(start))
		}
	}

	// A flood (1000/s) empties the bucket within a second or so.
	refused := false
	for i := 0; i < 2000 && !refused; i++ {
		now = now.Add(time.Millisecond)
		refused = !c.allowMessage(now)
	}
	if !refused {
		t.Fatal("a sustained flood was never refused")
	}
}
