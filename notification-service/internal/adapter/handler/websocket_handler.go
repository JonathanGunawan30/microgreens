package handler

import (
	"net/http"
	ws "notification-service/utils/websocket"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type WebSocketHandler interface {
	WebSocketHandler(c echo.Context) error
}

type webSocketHandler struct {
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewWebSocketHandler(e *echo.Echo) WebSocketHandler {
	wsHandler := &webSocketHandler{}

	e.Use(middleware.Recover())
	e.GET("/ws", wsHandler.WebSocketHandler)

	return wsHandler
}

// WebSocketHandler godoc
// @Summary WebSocket connection
// @Description Establish a WebSocket connection for real-time notifications
// @Tags websocket
// @Param user_id query int true "User ID"
// @Success 101 {string} string "Switching Protocols"
// @Failure 400 {string} string "Bad Request - Invalid user_id"
// @Router /ws [get]
func (w *webSocketHandler) WebSocketHandler(c echo.Context) error {
	userIDStr := c.QueryParam("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid user_id")
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}

	client := ws.AddWebSocketConn(int64(userID), conn)
	defer ws.RemoveWebSocketConn(int64(userID), client)

	done := make(chan struct{})
	defer close(done)

	go func() {
		ticker := time.NewTicker(ws.PingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := client.WritePing(); err != nil {
					_ = conn.Close()
					return
				}
			case <-done:
				return
			}
		}
	}()

	for {
		if _, _, err := conn.NextReader(); err != nil {
			break
		}
	}

	return nil

}
