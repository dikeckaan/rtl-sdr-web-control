package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

type Client struct {
	conn   *websocket.Conn
	topics map[string]bool
	send   chan []byte
	hub    *Hub
}

type Message struct {
	Topic string `json:"topic"`
	Data  any    `json:"data"`
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[*Client]struct{}),
	}
}

func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("[ws] accept error: %v", err)
		return
	}

	topicsParam := r.URL.Query().Get("topics")
	topics := make(map[string]bool)
	if topicsParam != "" {
		for _, t := range strings.Split(topicsParam, ",") {
			topics[strings.TrimSpace(t)] = true
		}
	}

	client := &Client{
		conn:   conn,
		topics: topics,
		send:   make(chan []byte, 256),
		hub:    h,
	}

	h.mu.Lock()
	h.clients[client] = struct{}{}
	h.mu.Unlock()

	go client.writePump()
	client.readPump()
}

func (h *Hub) Broadcast(topic string, data any) {
	msg := Message{Topic: topic, Data: data}
	payload, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[ws] marshal error: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if len(client.topics) == 0 || client.topics[topic] {
			select {
			case client.send <- payload:
			default:
				// Client buffer full, skip
			}
		}
	}
}

func (h *Hub) BroadcastBinary(topic string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.topics[topic] {
			select {
			case client.send <- data:
			default:
			}
		}
	}
}

func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
	close(c.send)
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (c *Client) readPump() {
	defer func() {
		c.hub.removeClient(c)
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		_, msg, err := c.conn.Read(context.Background())
		if err != nil {
			break
		}
		// Handle subscription changes
		var sub struct {
			Subscribe   []string `json:"subscribe"`
			Unsubscribe []string `json:"unsubscribe"`
		}
		if json.Unmarshal(msg, &sub) == nil {
			for _, t := range sub.Subscribe {
				c.topics[t] = true
			}
			for _, t := range sub.Unsubscribe {
				delete(c.topics, t)
			}
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := c.conn.Write(ctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				return
			}
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := c.conn.Ping(ctx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}
