package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketHubBroadcastAndRooms(t *testing.T) {
	hub := NewWebSocketHub()
	defer hub.Close()

	// Setup a test HTTP server with WebSocket endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := hub.UpgradeWebSocket(w, r, func(client *WSClient, msg []byte) {
			if string(msg) == "join:pos" {
				hub.JoinRoom(client, "pos")
			}
		}, nil)
		if err != nil {
			t.Errorf("UpgradeWebSocket failed: %v", err)
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Client 1 connects
	c1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Client 1 connection failed: %v", err)
	}
	defer c1.Close()

	// Client 2 connects
	c2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Client 2 connection failed: %v", err)
	}
	defer c2.Close()

	time.Sleep(50 * time.Millisecond)

	// Client 1 sends message to join "pos" room
	_ = c1.WriteMessage(websocket.TextMessage, []byte("join:pos"))
	time.Sleep(50 * time.Millisecond)

	// Broadcast an event to all clients
	err = hub.BroadcastEvent("ping", map[string]string{"message": "hello all"})
	if err != nil {
		t.Fatalf("BroadcastEvent failed: %v", err)
	}

	// Verify Client 1 receives broadcast
	_ = c1.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, msg, err := c1.ReadMessage()
	if err != nil {
		t.Fatalf("Client 1 did not receive broadcast: %v", err)
	}
	if !strings.Contains(string(msg), "hello all") {
		t.Errorf("unexpected message received: %s", string(msg))
	}

	// Broadcast to "pos" room
	err = hub.BroadcastToRoom("pos", "order.ready", map[string]int{"order_id": 101})
	if err != nil {
		t.Fatalf("BroadcastToRoom failed: %v", err)
	}

	// Client 1 should receive it (member of "pos")
	_ = c1.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	_, roomMsg, err := c1.ReadMessage()
	if err != nil {
		t.Fatalf("Client 1 did not receive room message: %v", err)
	}
	if !strings.Contains(string(roomMsg), "order.ready") {
		t.Errorf("unexpected room message received: %s", string(roomMsg))
	}
}
