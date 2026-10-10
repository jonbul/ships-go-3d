package game

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// writeTimeout bounds one frame write, so a client that stops reading
	// can't hold its writer goroutine forever.
	writeTimeout = 5 * time.Second
	// pongTimeout is how long a silent connection is kept. Pings go out at
	// pingPeriod, well inside it, so a healthy client always answers in time.
	pongTimeout = 60 * time.Second
	pingPeriod  = 25 * time.Second
	// sendBuffer is how many outgoing frames may queue for one client. A
	// client that falls this far behind is disconnected rather than allowed
	// to slow down - or grow the memory of - everybody else.
	sendBuffer = 128
	// maxMessageSize caps one incoming frame. Clients only ever send small
	// messages; ship designs are loaded server-side, never uploaded here.
	maxMessageSize = 16 * 1024
	// Flood protection, as a token bucket: a client may keep up messageRate
	// messages a second indefinitely (a normal one sends ~20 states plus its
	// shots and hit reports, ~30), and send up to messageBurst at once.
	//
	// The burst allowance is what matters on phones: a mobile connection
	// that stalls for a few seconds delivers everything the browser queued
	// meanwhile in one go. A plain per-second count (120/s, as this used to
	// be) disconnected players on mobile data within seconds of starting.
	messageRate  = 60.0
	messageBurst = 400.0
	// rateLimitGrace gives the "too many messages" notice a chance to reach
	// the client before its connection is closed.
	rateLimitGrace = 250 * time.Millisecond
)

// client is one websocket connection. gorilla/websocket allows a single
// concurrent writer per connection, so only writePump ever writes to conn:
// everything else queues frames on `send`, which never blocks the caller.
type client struct {
	conn      *websocket.Conn
	send      chan []byte
	closeOnce sync.Once
	done      chan struct{}

	// Token bucket (see messageRate), only touched by the reader goroutine.
	tokens   float64
	tokensAt time.Time
}

func newClient(conn *websocket.Conn) *client {
	return &client{conn: conn, send: make(chan []byte, sendBuffer), done: make(chan struct{})}
}

// enqueue queues a frame without ever blocking. If the client's buffer is
// full it is too slow to keep up, and is closed.
func (c *client) enqueue(frame []byte) {
	select {
	case <-c.done:
	case c.send <- frame:
	default:
		c.close()
	}
}

func (c *client) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.conn.Close()
	})
}

func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.close()
	}()
	for {
		select {
		case <-c.done:
			return
		case frame := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.TextMessage, frame); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// allowMessage spends a token for one incoming message, and reports whether
// the client had one: false means it has been flooding for long enough to
// empty a full bucket.
func (c *client) allowMessage(now time.Time) bool {
	if c.tokensAt.IsZero() {
		c.tokens = messageBurst
	} else {
		c.tokens = min(messageBurst, c.tokens+now.Sub(c.tokensAt).Seconds()*messageRate)
	}
	c.tokensAt = now
	if c.tokens < 1 {
		return false
	}
	c.tokens--
	return true
}
