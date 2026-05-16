package chat

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Hub struct {
	clients map[uuid.UUID]map[*websocket.Conn]bool
	mu      sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uuid.UUID]map[*websocket.Conn]bool),
	}
}

func (h *Hub) Register(userID uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*websocket.Conn]bool)
	}
	h.clients[userID][conn] = true
	log.Printf("[WS] User %s connected", userID)
}

func (h *Hub) Unregister(userID uuid.UUID, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.clients[userID]; ok {
		if _, exists := conns[conn]; exists {
			delete(conns, conn)
			conn.Close()
			if len(conns) == 0 {
				delete(h.clients, userID)
			}
			log.Printf("[WS] User %s disconnected", userID)
		}
	}
}

func (h *Hub) BroadcastToUser(userID uuid.UUID, message interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	conns, ok := h.clients[userID]
	if !ok {
		return // user offline
	}

	data, err := json.Marshal(message)
	if err != nil {
		return
	}

	for conn := range conns {
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Printf("[WS] Error sending message to %s: %v", userID, err)
		}
	}
}
