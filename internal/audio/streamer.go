package audio

import (
	"context"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

// Streamer reads audio data from a source and broadcasts to WebSocket clients.
type Streamer struct {
	mu      sync.RWMutex
	clients map[*audioClient]struct{}
	buf     []byte
}

type audioClient struct {
	conn *websocket.Conn
	send chan []byte
}

func NewStreamer() *Streamer {
	return &Streamer{
		clients: make(map[*audioClient]struct{}),
	}
}

// HandleWS accepts a WebSocket connection for audio streaming.
func (s *Streamer) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("[audio] ws accept error: %v", err)
		return
	}

	client := &audioClient{
		conn: conn,
		send: make(chan []byte, 64),
	}

	s.mu.Lock()
	s.clients[client] = struct{}{}
	s.mu.Unlock()

	log.Printf("[audio] Client connected (%d total)", s.ClientCount())

	// Write pump
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.clients, client)
			s.mu.Unlock()
			conn.Close(websocket.StatusNormalClosure, "")
			log.Printf("[audio] Client disconnected (%d remaining)", s.ClientCount())
		}()

		for data := range client.send {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := conn.Write(ctx, websocket.MessageBinary, data)
			cancel()
			if err != nil {
				return
			}
		}
	}()

	// Read pump (just to detect disconnection)
	for {
		_, _, err := conn.Read(context.Background())
		if err != nil {
			close(client.send)
			return
		}
	}
}

// StreamFrom reads from a reader and broadcasts audio chunks to all clients.
func (s *Streamer) StreamFrom(r io.Reader) {
	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			s.broadcast(chunk)
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("[audio] Read error: %v", err)
			}
			return
		}
	}
}

func (s *Streamer) broadcast(data []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		select {
		case client.send <- data:
		default:
			// Client too slow, skip frame
		}
	}
}

// HandleHTTPStream serves audio as an HTTP chunked response (fallback for non-WS clients).
func (s *Streamer) HandleHTTPStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	client := &audioClient{
		send: make(chan []byte, 64),
	}

	s.mu.Lock()
	s.clients[client] = struct{}{}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.clients, client)
		s.mu.Unlock()
	}()

	ctx := r.Context()
	for {
		select {
		case data, ok := <-client.send:
			if !ok {
				return
			}
			if _, err := w.Write(data); err != nil {
				return
			}
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

func (s *Streamer) ClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}
