package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/diyorend/budget-tracker/internal/alert"
	"github.com/diyorend/budget-tracker/internal/domain"
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

// BudgetStatusProvider is the read-only slice of BudgetService that the ws
// handler needs for the connect-time catch-up. Defined here (consumer side)
// so this handler doesn't have to depend on the concrete service type.
type BudgetStatusProvider interface {
	GetStatus(ctx context.Context, userID string, month time.Time) ([]*domain.BudgetStatus, error)
}

type WSHandler struct {
	broker    *alert.Broker
	budgetSvc BudgetStatusProvider
}

func NewWSHandler(broker *alert.Broker, budgetSvc BudgetStatusProvider) *WSHandler {
	return &WSHandler{broker: broker, budgetSvc: budgetSvc}
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

	// Catch-up: alerts are published fire-and-forget over Redis pub/sub —
	// if this connection wasn't open at the exact moment a transaction
	// pushed a budget past 80%/100%, that alert was fanned out to zero
	// listeners and lost for good. So on every (re)connect, replay the
	// user's current standing for any category already past the line.
	// Route it through broker.Publish rather than writing to conn
	// directly so it goes through the same safeConn-guarded write path
	// as live alerts — no risk of two goroutines writing this socket at
	// once.
	go h.sendCatchUpAlerts(userID)

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

// sendCatchUpAlerts fetches the user's current budget standing and
// re-publishes an alert for any category already at/above 80% — see the
// comment at the call site for why this is needed at all.
func (h *WSHandler) sendCatchUpAlerts(userID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	statuses, err := h.budgetSvc.GetStatus(ctx, userID, time.Now())
	if err != nil {
		slog.Error("ws catch-up: failed to load budget status", "user_id", userID, "err", err)
		return
	}

	for _, s := range statuses {
		alertMsg := domain.NewBudgetAlert(s.Category, s.Spent, s.LimitAmount, s.Percentage)
		if alertMsg == nil {
			continue
		}
		if err := h.broker.Publish(ctx, userID, *alertMsg); err != nil {
			slog.Error("ws catch-up: failed to publish alert", "user_id", userID, "category", s.Category, "err", err)
		}
	}
}
