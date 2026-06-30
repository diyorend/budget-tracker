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

// safeConn wraps a websocket connection with its own mutex.
// gorilla/websocket explicitly disallows concurrent calls to WriteMessage
// on the same connection from multiple goroutines — without this wrapper,
// two alerts firing close together for the same user can corrupt the frame.
type safeConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *safeConn) writeJSON(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// Broker manages WebSocket connections and Redis pub/sub.
type Broker struct {
	rdb *redis.Client
	mu  sync.RWMutex
	// userID -> set of connections
	conns map[string]map[*websocket.Conn]*safeConn
}

func NewBroker(rdb *redis.Client) *Broker {
	return &Broker{
		rdb:   rdb,
		conns: make(map[string]map[*websocket.Conn]*safeConn),
	}
}

// Register adds a WebSocket connection for a user.
func (b *Broker) Register(userID string, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.conns[userID] == nil {
		b.conns[userID] = make(map[*websocket.Conn]*safeConn)
	}
	b.conns[userID][conn] = &safeConn{conn: conn}
	slog.Info("ws registered", "user_id", userID, "total_conns", len(b.conns[userID]))
}

// Unregister removes a WebSocket connection.
func (b *Broker) Unregister(userID string, conn *websocket.Conn) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.conns[userID], conn)
	if len(b.conns[userID]) == 0 {
		delete(b.conns, userID)
	}
	slog.Info("ws unregistered", "user_id", userID)
}

// Publish sends an alert to Redis — called by the service layer.
func (b *Broker) Publish(ctx context.Context, userID string, msg domain.AlertMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	channel := "alerts:" + userID
	return b.rdb.Publish(ctx, channel, data).Err()
}

// Run subscribes to all alert channels and fans out to WebSocket clients.
// Call this in a goroutine from main.go — it blocks until ctx is cancelled.
func (b *Broker) Run(ctx context.Context) {
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

// fanOut sends a raw message to all connections for a user.
// Failed connections are collected under the read lock and cleaned up
// afterward under the write lock — never call Unregister (which takes
// the write lock) while still holding the read lock, that deadlocks.
func (b *Broker) fanOut(channel string, data []byte) {
	if len(channel) <= len("alerts:") {
		return
	}
	userID := channel[len("alerts:"):]

	b.mu.RLock()
	targets := make(map[*websocket.Conn]*safeConn, len(b.conns[userID]))
	for k, v := range b.conns[userID] {
		targets[k] = v
	}
	b.mu.RUnlock()

	var dead []*websocket.Conn
	for rawConn, sc := range targets {
		if err := sc.writeJSON(data); err != nil {
			slog.Error("ws write failed", "user_id", userID, "err", err)
			dead = append(dead, rawConn)
		}
	}

	for _, conn := range dead {
		b.Unregister(userID, conn)
		conn.Close()
	}
}
