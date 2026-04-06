package websocket

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var (
	wsClients      = make(map[int64]*websocket.Conn)
	wsClientsMutex = sync.RWMutex{}
)

func AddWebSocketConn(userID int64, conn *websocket.Conn) {
	wsClientsMutex.Lock()
	defer wsClientsMutex.Unlock()

	if oldConn, exists := wsClients[userID]; exists {
		_ = oldConn.Close()
	}

	wsClients[userID] = conn

	conn.SetReadLimit(1024)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
}

func GetWebSocketConn(userID int64) *websocket.Conn {
	wsClientsMutex.RLock()
	defer wsClientsMutex.RUnlock()

	conn, ok := wsClients[userID]
	if !ok {
		return nil
	}
	return conn
}

func RemoveWebSocketConn(userID int64) {
	wsClientsMutex.Lock()
	defer wsClientsMutex.Unlock()

	if conn, exists := wsClients[userID]; exists {
		_ = conn.Close()
		delete(wsClients, userID)
	}
}
