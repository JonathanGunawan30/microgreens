package websocket

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	PongWait   = 60 * time.Second
	PingPeriod = 30 * time.Second
	WriteWait  = 10 * time.Second
)

// ClientConn wraps a WebSocket connection with a write mutex to prevent concurrent write panics.
type ClientConn struct {
	Conn *websocket.Conn
	mu   sync.Mutex
}

func (c *ClientConn) WriteJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.Conn.SetWriteDeadline(time.Now().Add(WriteWait))
	return c.Conn.WriteJSON(v)
}

func (c *ClientConn) WritePing() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(WriteWait))
}

var (
	wsClients      = make(map[int64]map[*ClientConn]struct{})
	wsClientsMutex = sync.RWMutex{}
)

func AddWebSocketConn(userID int64, conn *websocket.Conn) *ClientConn {
	client := &ClientConn{Conn: conn}

	conn.SetReadLimit(1024)
	_ = conn.SetReadDeadline(time.Now().Add(PongWait))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(PongWait))
		return nil
	})

	wsClientsMutex.Lock()
	defer wsClientsMutex.Unlock()

	if wsClients[userID] == nil {
		wsClients[userID] = make(map[*ClientConn]struct{})
	}
	wsClients[userID][client] = struct{}{}

	return client
}

func GetWebSocketConns(userID int64) []*ClientConn {
	wsClientsMutex.RLock()
	defer wsClientsMutex.RUnlock()

	connsMap, ok := wsClients[userID]
	if !ok || len(connsMap) == 0 {
		return nil
	}

	conns := make([]*ClientConn, 0, len(connsMap))
	for client := range connsMap {
		conns = append(conns, client)
	}
	return conns
}

func RemoveWebSocketConn(userID int64, client *ClientConn) {
	if client == nil {
		return
	}

	wsClientsMutex.Lock()
	if connsMap, exists := wsClients[userID]; exists {
		delete(connsMap, client)
		if len(connsMap) == 0 {
			delete(wsClients, userID)
		}
	}
	wsClientsMutex.Unlock()

	_ = client.Conn.Close()
}
