package blink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// WebhookConfig defines the configuration for a webhook
type WebhookConfig struct {
	// URL to send the webhook to
	URL string
	// HTTP method to use (GET, POST, PUT, etc.)
	Method string
	// Headers to include in the request
	Headers map[string]string
	// Timeout for the HTTP request
	Timeout time.Duration
	// Debounce duration to avoid sending too many webhooks
	DebounceDuration time.Duration
	// Maximum number of retries for failed requests
	MaxRetries int
	// Filter to apply to events before sending webhooks
	Filter *EventFilter
}

// WebhookManager manages webhooks for file system events
type WebhookManager struct {
	// Configuration for the webhook
	Config WebhookConfig
	// HTTP client for sending webhooks
	client *http.Client
	// Map to track recent events for debouncing
	recentEvents map[string]time.Time
	// Mutex to protect the recentEvents map
	mu sync.Mutex
	// Channel to receive events
	eventChan chan fsnotify.Event
}

// WebhookPayload is the JSON payload sent to the webhook URL
type WebhookPayload struct {
	// Path of the file that changed
	Path string `json:"path"`
	// Type of event (create, write, remove, rename, chmod)
	EventType string `json:"event_type"`
	// Time the event occurred
	Time time.Time `json:"time"`
}

// NewWebhookManager creates a new webhook manager
func NewWebhookManager(config WebhookConfig) *WebhookManager {
	// Set default values if not provided
	if config.Method == "" {
		config.Method = "POST"
	}
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.DebounceDuration == 0 {
		config.DebounceDuration = 100 * time.Millisecond
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: config.Timeout,
	}

	// Create webhook manager
	manager := &WebhookManager{
		Config:       config,
		client:       client,
		recentEvents: make(map[string]time.Time),
		eventChan:    make(chan fsnotify.Event, 100),
	}

	// Start processing events
	go manager.processEvents()

	return manager
}

// HandleEvent handles a file system event
func (m *WebhookManager) HandleEvent(event fsnotify.Event) {
	// Apply filter if provided
	if m.Config.Filter != nil && !m.Config.Filter.ShouldInclude(event) {
		return
	}

	// Send event to channel for processing
	select {
	case m.eventChan <- event:
		// Successfully sent
	default:
		// Channel is full, log but don't block
		if LogError != nil {
			LogError(fmt.Errorf("webhook event channel is full, dropping event for %s", event.Name))
		}
	}
}

// processEvents processes events from the event channel
func (m *WebhookManager) processEvents() {
	for event := range m.eventChan {
		// Check if we should debounce this event
		if m.shouldDebounce(event) {
			continue
		}

		// Send webhook
		go m.sendWebhook(event)
	}
}

// shouldDebounce checks if an event should be debounced
func (m *WebhookManager) shouldDebounce(event fsnotify.Event) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	lastSeen, exists := m.recentEvents[event.Name]

	// If we've seen this event recently, debounce it
	if exists && now.Sub(lastSeen) < m.Config.DebounceDuration {
		return true
	}

	// Update the last seen time
	m.recentEvents[event.Name] = now

	// Clean up old events
	for path, t := range m.recentEvents {
		if now.Sub(t) > m.Config.DebounceDuration*10 {
			delete(m.recentEvents, path)
		}
	}

	return false
}

// sendWebhook sends a webhook for an event
func (m *WebhookManager) sendWebhook(event fsnotify.Event) {
	// Create payload
	payload := WebhookPayload{
		Path:      event.Name,
		EventType: eventTypeToString(event.Op),
		Time:      time.Now(),
	}

	// Marshal payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		if LogError != nil {
			LogError(fmt.Errorf("error marshaling webhook payload: %w", err))
		}
		return
	}

	// Create request
	req, err := http.NewRequest(m.Config.Method, m.Config.URL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		if LogError != nil {
			LogError(fmt.Errorf("error creating webhook request: %w", err))
		}
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Blink-Webhook")
	for key, value := range m.Config.Headers {
		req.Header.Set(key, value)
	}

	// Send request with retries
	var resp *http.Response
	for i := 0; i <= m.Config.MaxRetries; i++ {
		resp, err = m.client.Do(req)
		if err == nil {
			break
		}

		if i < m.Config.MaxRetries {
			// Wait before retrying
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
		}
	}

	// Check for errors
	if err != nil {
		if LogError != nil {
			LogError(fmt.Errorf("error sending webhook after %d retries: %w", m.Config.MaxRetries, err))
		}
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if LogError != nil {
			LogError(fmt.Errorf("webhook returned non-success status code: %d", resp.StatusCode))
		}
		return
	}

	// Log success
	if LogInfo != nil {
		LogInfo(fmt.Sprintf("Webhook sent successfully for %s (%s)", event.Name, eventTypeToString(event.Op)))
	}
}

// eventTypeToString converts an fsnotify.Op to a string
func eventTypeToString(op fsnotify.Op) string {
	switch {
	case op&fsnotify.Create != 0:
		return "create"
	case op&fsnotify.Write != 0:
		return "write"
	case op&fsnotify.Remove != 0:
		return "remove"
	case op&fsnotify.Rename != 0:
		return "rename"
	case op&fsnotify.Chmod != 0:
		return "chmod"
	default:
		return "unknown"
	}
}
