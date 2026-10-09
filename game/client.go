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
	// Incoming messages per second a client may send before it is
	// disconnected: about 20 states plus fire and hit reports, with slack.
	maxMessagesPerSecond = 120
)

// client is one websocket connection. gorilla/websocket allows a single
// concurrent writer per connection, so only writePump ever writes to conn:
// everything else queues frames on `send`, which never blocks the caller.
type client struct {
	conn      *websocket.Conn
	send      chan []byte
	closeOnce sync.Once
	done      chan struct{}

	// Rate limiting, only touched by the reader goroutine.
	windowStart time.Time
	windowCount int
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

// allowMessage counts one incoming message and reports whether the client is
// still within its rate limit.
func (c *client) allowMessage(now time.Time) bool {
	if now.Sub(c.windowStart) >= time.Second {
		c.windowStart = now
		c.windowCount = 0
	}
	c.windowCount++
	return c.windowCount <= maxMessagesPerSecond
}
