package bridge

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/gaucho-racing/warden/warden/pkg/logger"
)

const (
	pingInterval = 30 * time.Second
	writeTimeout = 10 * time.Second
	sendBuffer   = 64
)

// Hub holds the plugin's WebSocket connections. There is normally exactly
// one, but a restart can briefly overlap the old and new game server, so
// everything is broadcast rather than assuming a single socket.
type Hub struct {
	mu    sync.Mutex
	conns map[*pluginConn]struct{}
}

type pluginConn struct {
	ws   *websocket.Conn
	send chan []byte
}

func NewHub() *Hub {
	return &Hub{conns: make(map[*pluginConn]struct{})}
}

// Serve upgrades the request and blocks until the connection closes, handing
// each event to onEvent. Auth has already happened in AuthChecker.
func (h *Hub) Serve(w http.ResponseWriter, r *http.Request, onEvent func(Event)) {
	ws, err := websocket.Accept(w, r, nil)
	if err != nil {
		logger.SugarLogger.Warnf("bridge: websocket accept failed: %v", err)
		return
	}
	// The request context ends with the handler, so the connection gets its own.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn := &pluginConn{ws: ws, send: make(chan []byte, sendBuffer)}
	h.add(conn)
	defer h.remove(conn)
	logger.SugarLogger.Infof("bridge: plugin connected from %s", r.RemoteAddr)

	go conn.writeLoop(ctx, cancel)

	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			logger.SugarLogger.Infof("bridge: plugin disconnected: %v", err)
			ws.CloseNow()
			return
		}
		var event Event
		if err := json.Unmarshal(data, &event); err != nil {
			logger.SugarLogger.Warnf("bridge: bad event from plugin: %v", err)
			continue
		}
		onEvent(event)
	}
}

func (c *pluginConn) writeLoop(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()
	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case data := <-c.send:
			writeCtx, done := context.WithTimeout(ctx, writeTimeout)
			err := c.ws.Write(writeCtx, websocket.MessageText, data)
			done()
			if err != nil {
				c.ws.CloseNow()
				return
			}
		case <-ticker.C:
			pingCtx, done := context.WithTimeout(ctx, writeTimeout)
			err := c.ws.Ping(pingCtx)
			done()
			if err != nil {
				c.ws.CloseNow()
				return
			}
		}
	}
}

// Broadcast queues a message for every connected plugin. A plugin that has
// fallen a full buffer behind loses messages rather than stalling Discord.
func (h *Hub) Broadcast(message any) {
	data, err := json.Marshal(message)
	if err != nil {
		logger.SugarLogger.Errorf("bridge: encode message: %v", err)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for conn := range h.conns {
		select {
		case conn.send <- data:
		default:
			logger.SugarLogger.Warnf("bridge: plugin send buffer full, dropping message")
		}
	}
}

// Count is how many game servers are connected. Normally one; briefly two
// across a restart, and zero whenever the game server is down.
func (h *Hub) Count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.conns)
}

func (h *Hub) add(conn *pluginConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[conn] = struct{}{}
}

func (h *Hub) remove(conn *pluginConn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, conn)
}
