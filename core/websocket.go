package core

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/exp/slog"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins by default (can be customized via middleware)
	},
}

// WSClient represents an active WebSocket connection.
type WSClient struct {
	ID    string
	hub   *WebSocketHub
	conn  *websocket.Conn
	send  chan []byte
	rooms map[string]bool
	mu    sync.Mutex
}

// WSMessage encapsulates a message payload targeted at an optional room/channel.
type WSMessage struct {
	Room    string      `json:"room,omitempty"`
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
}

// WebSocketHub manages all connected WebSocket clients, rooms, and broadcasts.
type WebSocketHub struct {
	clients    map[*WSClient]bool
	rooms      map[string]map[*WSClient]bool
	broadcast  chan []byte
	register   chan *WSClient
	unregister chan *WSClient
	mu         sync.RWMutex
	quit       chan struct{}
	once       sync.Once
}

// NewWebSocketHub initializes and starts a new WebSocket Hub.
func NewWebSocketHub() *WebSocketHub {
	hub := &WebSocketHub{
		clients:    make(map[*WSClient]bool),
		rooms:      make(map[string]map[*WSClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		quit:       make(chan struct{}),
	}
	go hub.run()
	return hub
}

func (h *WebSocketHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				// Remove client from all joined rooms
				for room := range client.rooms {
					if members, exists := h.rooms[room]; exists {
						delete(members, client)
						if len(members) == 0 {
							delete(h.rooms, room)
						}
					}
				}
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()

		case <-h.quit:
			h.mu.Lock()
			for client := range h.clients {
				close(client.send)
				_ = client.conn.Close()
			}
			h.clients = make(map[*WSClient]bool)
			h.rooms = make(map[string]map[*WSClient]bool)
			h.mu.Unlock()
			return
		}
	}
}

// JoinRoom adds a client to a specific room or channel.
func (h *WebSocketHub) JoinRoom(client *WSClient, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.rooms[room]; !exists {
		h.rooms[room] = make(map[*WSClient]bool)
	}
	h.rooms[room][client] = true

	client.mu.Lock()
	if client.rooms == nil {
		client.rooms = make(map[string]bool)
	}
	client.rooms[room] = true
	client.mu.Unlock()
}

// LeaveRoom removes a client from a specific room or channel.
func (h *WebSocketHub) LeaveRoom(client *WSClient, room string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if members, exists := h.rooms[room]; exists {
		delete(members, client)
		if len(members) == 0 {
			delete(h.rooms, room)
		}
	}

	client.mu.Lock()
	delete(client.rooms, room)
	client.mu.Unlock()
}

// Broadcast sends a raw message byte slice to all connected clients.
func (h *WebSocketHub) Broadcast(message []byte) {
	select {
	case h.broadcast <- message:
	default:
		slog.Warn("websocket broadcast channel full, message dropped")
	}
}

// BroadcastEvent marshals an event with payload and broadcasts to all clients.
func (h *WebSocketHub) BroadcastEvent(event string, payload interface{}) error {
	msgBytes, err := json.Marshal(WSMessage{Event: event, Payload: payload})
	if err != nil {
		return err
	}
	h.Broadcast(msgBytes)
	return nil
}

// BroadcastToRoom sends a message to all clients in a specific room.
func (h *WebSocketHub) BroadcastToRoom(room, event string, payload interface{}) error {
	msgBytes, err := json.Marshal(WSMessage{Room: room, Event: event, Payload: payload})
	if err != nil {
		return err
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	members, exists := h.rooms[room]
	if !exists {
		return nil
	}

	for client := range members {
		select {
		case client.send <- msgBytes:
		default:
			slog.Warn("client send channel full in room broadcast", "room", room)
		}
	}
	return nil
}

// Close gracefully stops the WebSocket hub and closes all connections.
func (h *WebSocketHub) Close() {
	h.once.Do(func() {
		close(h.quit)
	})
}

// UpgradeWebSocket upgrades an HTTP connection to a WebSocket connection and registers with the hub.
func (h *WebSocketHub) UpgradeWebSocket(w http.ResponseWriter, r *http.Request, onMessage func(client *WSClient, msg []byte), onClose func(client *WSClient)) (*WSClient, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	client := &WSClient{
		hub:   h,
		conn:  conn,
		send:  make(chan []byte, 256),
		rooms: make(map[string]bool),
	}

	h.register <- client

	// Start pump goroutines
	go client.writePump()
	go client.readPump(onMessage, onClose)

	return client, nil
}

func (c *WSClient) readPump(onMessage func(client *WSClient, msg []byte), onClose func(client *WSClient)) {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
		if onClose != nil {
			onClose(c)
		}
	}()

	c.conn.SetReadLimit(512 * 1024) // 512KB max message size
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		if onMessage != nil {
			onMessage(c, message)
		}
	}
}

func (c *WSClient) writePump() {
	ticker := time.NewTicker(25 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Send emits a raw message directly to this client.
func (c *WSClient) Send(message []byte) {
	select {
	case c.send <- message:
	default:
		slog.Warn("client send buffer full, dropping message")
	}
}

// SendEvent emits a JSON event directly to this client.
func (c *WSClient) SendEvent(event string, payload interface{}) error {
	msgBytes, err := json.Marshal(WSMessage{Event: event, Payload: payload})
	if err != nil {
		return err
	}
	c.Send(msgBytes)
	return nil
}
