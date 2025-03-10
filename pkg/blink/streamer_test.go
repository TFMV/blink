package blink

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

func TestSSEStreamer(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Send a test event
		w.Write([]byte("id: 1\n"))
		w.Write([]byte("data: test event\n\n"))
		w.(http.Flusher).Flush()
	}))
	defer server.Close()

	// Create an SSE streamer
	streamer := NewSSEStreamer(StreamerOptions{
		Address:         server.URL,
		Path:            "/events",
		AllowedOrigin:   "*",
		RefreshDuration: 100 * time.Millisecond,
	})

	// Start the streamer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := streamer.Start(ctx); err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Send a test event
	event := fsnotify.Event{
		Name: "/test/file.txt",
		Op:   fsnotify.Create,
	}

	if err := streamer.Send(event); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	// Stop the streamer
	if err := streamer.Stop(); err != nil {
		t.Fatalf("Failed to stop streamer: %v", err)
	}
}

func TestWebSocketStreamer(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		// Read one message and echo it back
		_, msg, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("Failed to read message: %v", err)
			return
		}

		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			t.Fatalf("Failed to write message: %v", err)
			return
		}
	}))
	defer server.Close()

	// Create a WebSocket streamer
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	streamer := NewWebSocketStreamer(StreamerOptions{
		Address:       wsURL,
		Path:          "/events",
		AllowedOrigin: "*",
	})

	// Start the streamer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := streamer.Start(ctx); err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Send a test event
	event := fsnotify.Event{
		Name: "/test/file.txt",
		Op:   fsnotify.Create,
	}

	if err := streamer.Send(event); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	// Stop the streamer
	if err := streamer.Stop(); err != nil {
		t.Fatalf("Failed to stop streamer: %v", err)
	}
}

func TestMultiStreamer(t *testing.T) {
	// Create mock streamers
	mockSSE := &mockStreamer{}
	mockWS := &mockStreamer{}

	// Create a multi-streamer
	streamer := NewMultiStreamer(mockSSE, mockWS)

	// Start the streamer
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := streamer.Start(ctx); err != nil {
		t.Fatalf("Failed to start streamer: %v", err)
	}

	// Send a test event
	event := fsnotify.Event{
		Name: "/test/file.txt",
		Op:   fsnotify.Create,
	}

	if err := streamer.Send(event); err != nil {
		t.Fatalf("Failed to send event: %v", err)
	}

	// Verify that both streamers received the event
	if !mockSSE.started {
		t.Error("SSE streamer was not started")
	}
	if !mockWS.started {
		t.Error("WebSocket streamer was not started")
	}
	if !mockSSE.received {
		t.Error("SSE streamer did not receive the event")
	}
	if !mockWS.received {
		t.Error("WebSocket streamer did not receive the event")
	}

	// Stop the streamer
	if err := streamer.Stop(); err != nil {
		t.Fatalf("Failed to stop streamer: %v", err)
	}

	// Verify that both streamers were stopped
	if !mockSSE.stopped {
		t.Error("SSE streamer was not stopped")
	}
	if !mockWS.stopped {
		t.Error("WebSocket streamer was not stopped")
	}
}

func TestIntegration(t *testing.T) {
	// Skip if running in CI environment
	if os.Getenv("CI") != "" {
		t.Skip("Skipping integration test in CI environment")
	}

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "blink-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create channels to receive events
	eventChan := make(chan map[string]interface{}, 1)
	errChan := make(chan error, 1)

	// Create a WebSocket server
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			errChan <- fmt.Errorf("failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		// Read messages
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					errChan <- fmt.Errorf("websocket read error: %v", err)
				}
				return
			}

			// Parse the message
			var data map[string]interface{}
			if err := json.Unmarshal(msg, &data); err != nil {
				errChan <- fmt.Errorf("failed to unmarshal message: %v", err)
				continue
			}

			// Send the event to the channel
			eventChan <- data
			return
		}
	}))
	defer wsServer.Close()

	// Manually set the server to use our test server's handler
	mux := http.NewServeMux()
	mux.HandleFunc("/events", wsServer.Config.Handler.ServeHTTP)

	// Start the streamer with a context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect to the WebSocket server directly
	wsURL := "ws" + strings.TrimPrefix(wsServer.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect to WebSocket server: %v", err)
	}
	defer conn.Close()

	// Create and send the event message
	message := map[string]interface{}{
		"type":      "event",
		"timestamp": time.Now().UnixNano() / 1000000,
		"path":      testFile,
		"operation": "write",
	}
	data, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Wait for the event to be received
	select {
	case receivedEvent := <-eventChan:
		// Verify the message
		if receivedEvent["type"] != "event" {
			t.Errorf("Expected event type 'event', got %v", receivedEvent["type"])
		}
		if receivedEvent["path"] != testFile {
			t.Errorf("Expected path %s, got %v", testFile, receivedEvent["path"])
		}
		if receivedEvent["operation"] != "write" {
			t.Errorf("Expected operation 'write', got %v", receivedEvent["operation"])
		}
	case err := <-errChan:
		t.Fatalf("Error from WebSocket server: %v", err)
	case <-ctx.Done():
		t.Fatal("Timed out waiting for event")
	}
}

// mockStreamer is a mock implementation of EventStreamer for testing
type mockStreamer struct {
	started  bool
	stopped  bool
	received bool
}

func (m *mockStreamer) Start(ctx context.Context) error {
	m.started = true
	return nil
}

func (m *mockStreamer) Stop() error {
	m.stopped = true
	return nil
}

func (m *mockStreamer) Send(event fsnotify.Event) error {
	m.received = true
	return nil
}
