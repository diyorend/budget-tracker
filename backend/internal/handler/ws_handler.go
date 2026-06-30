package handler

import (
	"log/slog"
	"net/http"

	"github.com/diyorend/budget-tracker/internal/alert"
	"github.com/diyorend/budget-tracker/internal/middleware"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// In production: check r.Header.Get("Origin") against an allowlist.
		return true
	},
}

type WSHandler struct {
	broker *alert.Broker
}

func NewWSHandler(broker *alert.Broker) *WSHandler {
	return &WSHandler{broker: broker}
}

// Connect handles GET /ws — upgrades to WebSocket and registers with broker.
// Client must send the JWT as a query param: /ws?token=<jwt>
func (h *WSHandler) Connect(c echo.Context) error {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		return err
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		// Upgrade already wrote its own HTTP error response — do not
		// proceed to register a nil connection or write to the response
		// again, that would panic.
		slog.Error("ws upgrade failed", "user_id", userID, "err", err)
		return nil
	}

	h.broker.Register(userID, conn)
	defer func() {
		h.broker.Unregister(userID, conn)
		conn.Close()
	}()

	slog.Info("ws connected", "user_id", userID)

	// Keep the connection alive — read loop. We don't expect the client to
	// send anything meaningful, but reading is what detects disconnects
	// and keeps gorilla's internal ping/pong handling running.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			slog.Info("ws disconnected", "user_id", userID, "reason", err)
			break
		}
	}

	return nil
}
