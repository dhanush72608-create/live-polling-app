package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// PollHub manages all active WebSocket connections watching a single poll
type PollHub struct {
	pollID  string
	clients map[*websocket.Conn]bool
	mu      sync.Mutex
}

var (
	hubsMu sync.Mutex
	hubs   = make(map[string]*PollHub)
)

// HandleWebSocket upgrades HTTP to WebSocket and registers the connection
func HandleWebSocket(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		pollID := c.Param("id")
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			return
		}

		hubsMu.Lock()
		hub, exists := hubs[pollID]
		if !exists {
			hub = &PollHub{
				pollID:  pollID,
				clients: make(map[*websocket.Conn]bool),
			}
			hubs[pollID] = hub
			go startRedisSubscriber(rdb, hub)
		}
		hub.mu.Lock()
		hub.clients[conn] = true
		hub.mu.Unlock()
		hubsMu.Unlock()

		defer func() {
			hub.mu.Lock()
			delete(hub.clients, conn)
			hub.mu.Unlock()
			conn.Close()
		}()

		// Keep connection open and detect client disconnects
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	}
}

// startRedisSubscriber listens to the Redis channel for a poll and broadcasts to connected clients
func startRedisSubscriber(rdb *redis.Client, hub *PollHub) {
	ctx := context.Background()
	pubsub := rdb.Subscribe(ctx, "poll:"+hub.pollID)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		hub.mu.Lock()
		for client := range hub.clients {
			_ = client.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := client.WriteMessage(websocket.TextMessage, []byte(msg.Payload)); err != nil {
				client.Close()
				delete(hub.clients, client)
			}
		}
		hub.mu.Unlock()
	}
}

// BroadcastVote increments Redis counter and publishes the new tally
func BroadcastVote(ctx context.Context, rdb *redis.Client, pollID, optionID string) error {
	// 1. Atomically increment the option vote in Redis
	rdb.HIncrBy(ctx, "poll_counts:"+pollID, optionID, 1)

	// 2. Retrieve all updated counts for this poll
	counts, err := rdb.HGetAll(ctx, "poll_counts:"+pollID).Result()
	if err != nil {
		return err
	}

	payload, err := json.Marshal(counts)
	if err != nil {
		return err
	}

	// 3. Publish update to Redis Pub/Sub channel
	return rdb.Publish(ctx, "poll:"+pollID, payload).Err()
}