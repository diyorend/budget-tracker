package handler

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/diyorend/budget-tracker/internal/alert"
	"github.com/diyorend/budget-tracker/internal/middleware"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// in production: check r.Header.Get("Origin") against allowlist
		return true
	},
}

type WSHandler struct {
	broker *alert.Broker
}

func NewWSHandler(broker *alert.Broker) *WSHandler {
	return &WSHandler{broker: broker}
}

// Connect handles GET /ws - upgrades to WebSocket and registers with broker
// Client must send token as query param: /ws?token=<jwt>
func (h *WSHandler) Connect(c echo.Context) error {
	// For WebSocket, the JWT comes as a query param (can't set headers in browser WS API)
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		return err
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("ws upgrade failed", "err", err)
	}

	h.broker.Register(userID, conn)
	defer func() {
		h.broker.Unregister(userID, conn)
		conn.Close()
	}()

	slog.Info("ws connected", "user_id", userID)

	// Keep connection alive - read loop (client sends pings)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			// Connection closed by client - normal
			break
		}
	}

	return nil
}


