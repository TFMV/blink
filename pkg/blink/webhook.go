package blink

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/TFMV/blink/pkg/logger"
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
	// Time the event occurred (RFC3339 format)
	Timestamp string `json:"timestamp"`
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

// HandleEvent processes a file system event and sends it to the webhook
func (m *WebhookManager) HandleEvent(event fsnotify.Event) {
	// Skip if no URL is configured
	if m.Config.URL == "" {
		return
	}

	// If debounce is enabled, use the event channel
	if m.Config.DebounceDuration > 0 {
		select {
		case m.eventChan <- event:
			// Event added to channel
		default:
			// Channel is full, log and drop the event
			logger.Error(fmt.Errorf("webhook event channel is full, dropping event for %s", event.Name))
		}
		return
	}

	// Otherwise, send the webhook immediately
	m.sendWebhook(event)
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

// sendWebhook sends a webhook for the given event
func (m *WebhookManager) sendWebhook(event fsnotify.Event) {
	// Create the payload
	payload := WebhookPayload{
		Path:      event.Name,
		EventType: eventTypeToString(event.Op),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Marshal the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		logger.Error(fmt.Errorf("error marshaling webhook payload: %w", err))
		return
	}

	// Create the request
	req, err := http.NewRequest(m.Config.Method, m.Config.URL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		logger.Error(fmt.Errorf("error creating webhook request: %w", err))
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for key, value := range m.Config.Headers {
		req.Header.Set(key, value)
	}

	// Send the request with retries
	var resp *http.Response
	var sendErr error

	client := &http.Client{
		Timeout: m.Config.Timeout,
	}

	for i := 0; i <= m.Config.MaxRetries; i++ {
		resp, sendErr = client.Do(req)
		if sendErr == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			break
		}

		if i < m.Config.MaxRetries {
			// Wait before retrying
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}

	// Check for errors
	if sendErr != nil {
		logger.Error(fmt.Errorf("error sending webhook after %d retries: %w", m.Config.MaxRetries, sendErr))
		return
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		logger.Error(fmt.Errorf("webhook returned non-success status code: %d", resp.StatusCode))
		return
	}

	// Log success
	logger.Infof("Webhook sent successfully for %s (%s)", event.Name, eventTypeToString(event.Op))
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
