package blink

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/TFMV/blink/pkg/logger"
	"github.com/fsnotify/fsnotify"
	"github.com/gorilla/websocket"
)

// StreamMethod defines the method used for streaming events
type StreamMethod string

const (
	// StreamMethodSSE uses Server-Sent Events for streaming
	StreamMethodSSE StreamMethod = "sse"
	// StreamMethodWebSocket uses WebSockets for streaming
	StreamMethodWebSocket StreamMethod = "websocket"
	// StreamMethodBoth uses both SSE and WebSockets for streaming
	StreamMethodBoth StreamMethod = "both"
)

// EventStreamer is an interface for streaming events to clients
type EventStreamer interface {
	// Start initializes and starts the streamer
	Start(ctx context.Context) error

	// Stop gracefully shuts down the streamer
	Stop() error

	// Send delivers an event to all connected clients
	Send(event fsnotify.Event) error
}

// StreamerOptions contains configuration for event streamers
type StreamerOptions struct {
	// Address to listen on ([host][:port])
	Address string

	// Path for the event stream
	Path string

	// AllowedOrigin for CORS (Access-Control-Allow-Origin)
	AllowedOrigin string

	// RefreshDuration for SSE events
	RefreshDuration time.Duration

	// Filter for events
	Filter *EventFilter
}

// SSEStreamer implements EventStreamer using Server-Sent Events
type SSEStreamer struct {
	opts      StreamerOptions
	server    *http.Server
	events    TimeEventMap
	mutex     sync.Mutex
	clients   map[string]chan fsnotify.Event
	clientsMu sync.Mutex
}

// NewSSEStreamer creates a new SSE streamer
func NewSSEStreamer(opts StreamerOptions) *SSEStreamer {
	if opts.Path == "" {
		opts.Path = "/events"
	}
	if opts.Address == "" {
		opts.Address = ":12345"
	}
	if opts.AllowedOrigin == "" {
		opts.AllowedOrigin = "*"
	}
	if opts.RefreshDuration == 0 {
		opts.RefreshDuration = 100 * time.Millisecond
	}

	return &SSEStreamer{
		opts:    opts,
		events:  make(TimeEventMap),
		clients: make(map[string]chan fsnotify.Event),
	}
}

// Start initializes and starts the SSE streamer
func (s *SSEStreamer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc(s.opts.Path, s.handleSSE)

	s.server = &http.Server{
		Addr:    s.opts.Address,
		Handler: mux,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(err)
		}
	}()

	logger.Infof("SSE server started on %s%s", s.opts.Address, s.opts.Path)

	// Listen for context cancellation
	go func() {
		<-ctx.Done()
		if err := s.Stop(); err != nil {
			logger.Error(err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the SSE streamer
func (s *SSEStreamer) Stop() error {
	if s.server == nil {
		return nil
	}

	// Create a shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown the server
	err := s.server.Shutdown(ctx)
	return err
}

// Send delivers an event to all connected clients
func (s *SSEStreamer) Send(event fsnotify.Event) error {
	// Add the event to the map
	now := time.Now()

	s.mutex.Lock()
	s.events[now] = Event(event)
	// Remove old events
	RemoveOldEvents(&s.events, s.opts.RefreshDuration*10)
	s.mutex.Unlock()

	return nil
}

// handleSSE handles SSE connections
func (s *SSEStreamer) handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream;charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", s.opts.AllowedOrigin)

	var id uint64

	for {
		func() { // Use an anonymous function for defer
			s.mutex.Lock()
			defer s.mutex.Unlock()

			if len(s.events) > 0 {
				// Remove old keys
				RemoveOldEvents(&s.events, s.opts.RefreshDuration*10)

				// Sort the events by the registered time
				var keys timeKeys
				for k := range s.events {
					keys = append(keys, k)
				}
				sort.Sort(keys)

				prevname := ""
				for _, k := range keys {
					ev := s.events[k]

					// Apply filter if one exists
					if s.opts.Filter != nil && !s.opts.Filter.ShouldProcessEvent(fsnotify.Event(ev)) {
						continue
					}

					// Log the event
					logger.Infof("EVENT %s", ev.String())

					// Avoid sending several events for the same filename
					if ev.Name != prevname {
						// Send an event to the client
						WriteEvent(w, &id, ev.Name, true)
						id++
						prevname = ev.Name
					}
				}
			}
		}()

		// Wait for old events to be gone, and new to appear
		time.Sleep(s.opts.RefreshDuration)
	}
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	ID         string
	Connection *websocket.Conn
	SendChan   chan []byte
}

// WebSocketStreamer implements EventStreamer using WebSockets
type WebSocketStreamer struct {
	server    *http.Server
	upgrader  websocket.Upgrader
	clients   map[string]*WebSocketClient
	mutex     sync.RWMutex
	opts      StreamerOptions
	started   bool
	filter    *EventFilter
	clientsMu sync.Mutex
}

// NewWebSocketStreamer creates a new WebSocket streamer
func NewWebSocketStreamer(opts StreamerOptions) *WebSocketStreamer {
	if opts.Path == "" {
		opts.Path = "/ws"
	}
	if opts.Address == "" {
		opts.Address = ":12345"
	}

	streamer := &WebSocketStreamer{
		clients: make(map[string]*WebSocketClient),
		opts:    opts,
		filter:  opts.Filter,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				if opts.AllowedOrigin == "*" {
					return true
				}
				origin := r.Header.Get("Origin")
				return origin == opts.AllowedOrigin
			},
		},
	}

	return streamer
}

// Start initializes and starts the WebSocket streamer
func (ws *WebSocketStreamer) Start(ctx context.Context) error {
	ws.mutex.Lock()
	if ws.started {
		ws.mutex.Unlock()
		return nil
	}
	ws.started = true
	ws.mutex.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc(ws.opts.Path, ws.handleWebSocket)

	ws.server = &http.Server{
		Addr:    ws.opts.Address,
		Handler: mux,
	}

	go func() {
		if err := ws.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(fmt.Errorf("WebSocket server error: %w", err))
		}
	}()

	logger.Infof("WebSocket server started on %s%s", ws.opts.Address, ws.opts.Path)

	// Listen for context cancellation
	go func() {
		<-ctx.Done()
		if err := ws.Stop(); err != nil {
			logger.Error(err)
		}
	}()

	return nil
}

// Stop gracefully shuts down the WebSocket streamer
func (ws *WebSocketStreamer) Stop() error {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	if !ws.started || ws.server == nil {
		return nil
	}

	// Close all client connections
	for _, client := range ws.clients {
		close(client.SendChan)
		client.Connection.Close()
	}
	ws.clients = make(map[string]*WebSocketClient)

	// Create a shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown the server
	err := ws.server.Shutdown(ctx)
	ws.started = false
	return err
}

// Send delivers an event to all connected clients
func (ws *WebSocketStreamer) Send(event fsnotify.Event) error {
	// Apply filter if one exists
	if ws.filter != nil && !ws.filter.ShouldProcessEvent(event) {
		return nil
	}

	// Create a message to send
	message := map[string]interface{}{
		"type":      "event",
		"timestamp": time.Now().UnixNano() / 1000000,
		"path":      event.Name,
		"operation": eventOpToString(event.Op),
	}

	// Marshal the message to JSON
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Send to all clients
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()

	for _, client := range ws.clients {
		// Non-blocking send to client's channel
		select {
		case client.SendChan <- data:
			// Successfully sent
		default:
			// Channel is full, log and continue
			logger.Error(fmt.Errorf("client %s send buffer full, dropping message", client.ID))
		}
	}

	return nil
}

// handleWebSocket handles WebSocket connections
func (ws *WebSocketStreamer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(fmt.Errorf("failed to upgrade connection: %w", err))
		return
	}

	// Generate a client ID
	clientID := fmt.Sprintf("%s-%d", r.RemoteAddr, time.Now().UnixNano())

	// Create a new client
	client := &WebSocketClient{
		ID:         clientID,
		Connection: conn,
		SendChan:   make(chan []byte, 256),
	}

	// Register the client
	ws.clientsMu.Lock()
	ws.clients[client.ID] = client
	ws.clientsMu.Unlock()

	// Start goroutines for writing and reading
	go ws.writeLoop(client)
	go ws.readLoop(client)
}

// writeLoop sends messages to the client
func (ws *WebSocketStreamer) writeLoop(client *WebSocketClient) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.Connection.Close()
	}()

	for {
		select {
		case message, ok := <-client.SendChan:
			client.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed, close the connection
				client.Connection.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Write the message
			if err := client.Connection.WriteMessage(websocket.TextMessage, message); err != nil {
				logger.Error(fmt.Errorf("error writing to client %s: %w", client.ID, err))
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			client.Connection.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Connection.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readLoop reads messages from the client
func (ws *WebSocketStreamer) readLoop(client *WebSocketClient) {
	defer func() {
		ws.mutex.Lock()
		delete(ws.clients, client.ID)
		ws.mutex.Unlock()
		client.Connection.Close()
	}()

	// Set read deadline and pong handler
	client.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Connection.SetPongHandler(func(string) error {
		client.Connection.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Read messages (mainly to detect disconnection)
	for {
		_, _, err := client.Connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logger.Error(fmt.Errorf("WebSocket read error: %w", err))
			}
			break
		}
	}
}

// MultiStreamer combines multiple EventStreamers
type MultiStreamer struct {
	streamers []EventStreamer
}

// NewMultiStreamer creates a new multi-streamer
func NewMultiStreamer(streamers ...EventStreamer) *MultiStreamer {
	return &MultiStreamer{
		streamers: streamers,
	}
}

// Start initializes and starts all streamers
func (m *MultiStreamer) Start(ctx context.Context) error {
	for _, streamer := range m.streamers {
		if err := streamer.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Stop gracefully shuts down all streamers
func (m *MultiStreamer) Stop() error {
	var lastErr error
	for _, streamer := range m.streamers {
		if err := streamer.Stop(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Send delivers an event to all streamers
func (m *MultiStreamer) Send(event fsnotify.Event) error {
	var lastErr error
	for _, streamer := range m.streamers {
		if err := streamer.Send(event); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
