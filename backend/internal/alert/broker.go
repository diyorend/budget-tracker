package alert

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/diyorend/budget-tracker/internal/domain"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// Broker manages WebSocket connections and Redis pub/sub
type Broker struct {
	rdb *redis.Client
	mu sync.RWMutex
	// userID -> set of WebSocket connections
	conns map[string]map[*websocket.Conn]struct{}
}

func NewBroker(rdb *redis.Client) *Broker {
	return &Broker{
		rdb: rdb,
		conns: make(map[string]map[*websocket.Conn]struct{}),
	}
}

// Register adds a WebSocket connection for a user
func (b *Broker) Register(userID string, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conns[userID] == nil {
		b.conns[userID] = make(map[*websocket.Conn]struct{})
	}
	b.conns[userID][conn] = struct{}{}
	slog.Info("ws registered", "user_id", userID, "total_conns", len(b.conns[userID]))
}

// Unregister removes a WebSocket connection
func (b *Broker) Unregister(userID string, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.conns[userID], conn)
	if len(b.conns[userID]) == 0 {
		delete(b.conns, userID)
	}
	slog.Info("ws unregistered", "user_id", userID)
}

// Publish sends an alert to Redis - called by the service layer
func (b *Broker) Publish(ctx context.Context, userID string, msg domain.AlertMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	channel := "alerts:" + userID
	return b.rdb.Publish(ctx, channel, data).Err()
}

// Run subscribes to all alert channels and fans out to WebSocket clients.
// Call this in a goroutine from main.go - it blocks until ctx is cancelled.
func (b *Broker) Run(ctx context.Context) {
	// Subscribe to a pattern so we catch all user channels
	pubsub := b.rdb.PSubscribe(ctx, "alerts:*")
	defer pubsub.Close()

	slog.Info("alert broker started")

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			slog.Info("alert broker stopping")
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			b.fanOut(msg.Channel, []byte(msg.Payload))
		}
	}
}

// fanOut sends a raw message to all connections for a user
func (b *Broker) fanOut(channel string, data []byte) {
	// channel format: "alerts:{userID}"
	if len(channel) < 8 {
		return
	}
	userID := channel[7:] // strip "alerts:"

	b.mu.RLock()
	conns := b.conns[userID]
	b.mu.RUnlock()

	for conn := range conns {
		// WriteMessage is not concurrent-safe - use a mutex per connection
		// For simplicity here we write directly; in production wrap conn in a struct with a mutex
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			slog.Error("ws write failed", "user_id", userID, "err", err)
			b.Unregister(userID, conn)
			conn.Close()
		}
	}
}
