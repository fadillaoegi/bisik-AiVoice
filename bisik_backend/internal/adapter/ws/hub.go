package ws

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub menyiarkan event ke semua klien yang menonton satu sesi
// (PWA petugas + dashboard supervisor + app mobile).
type Hub struct {
	mu    sync.RWMutex
	rooms map[string]map[*Client]bool
	log   *slog.Logger
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
	role string // "officer" | "supervisor"
}

func NewHub(log *slog.Logger) *Hub {
	return &Hub{rooms: make(map[string]map[*Client]bool), log: log}
}

func (h *Hub) Join(sessionID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.rooms[sessionID] == nil {
		h.rooms[sessionID] = make(map[*Client]bool)
	}
	h.rooms[sessionID][c] = true
}

func (h *Hub) Leave(sessionID string, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if room, ok := h.rooms[sessionID]; ok {
		delete(room, c)
		if len(room) == 0 {
			delete(h.rooms, sessionID)
		}
	}
	close(c.send)
}

// Publish memenuhi usecase.Broadcaster.
func (h *Hub) Publish(sessionID string, event any) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[sessionID] {
		select {
		case c.send <- payload:
		default: // klien lambat, jangan blokir pipeline audio
		}
	}
}

// PublishTo mengirim hanya ke peran tertentu.
// Dipakai untuk bisikan: nasabah tidak boleh dengar, supervisor tidak perlu.
func (h *Hub) PublishTo(sessionID, role string, event any) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[sessionID] {
		if c.role != role {
			continue
		}
		select {
		case c.send <- payload:
		default:
		}
	}
}
