package blink

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/TFMV/blink/pkg/logger"
	"github.com/fsnotify/fsnotify"
)

type (
	// TimeEventMap stores filesystem events
	TimeEventMap map[time.Time]Event
)

// Customizable functions, as exported variables. Can be se to "nil".

// LogInfo logs a message as information
var LogInfo = func(msg string) {
	logger.Info(msg)
}

// LogError logs a message as an error, but does not end the program
var LogError = func(err error) {
	logger.Error(err)
}

// FatalExit ends the program after logging a message
var FatalExit = func(err error) {
	logger.Fatal(err)
}

// Exists checks if the given path exists, using os.Stat
var Exists = func(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SetVerbose can be used to enable or disable logging of incoming events
func SetVerbose(enabled bool) {
	if enabled {
		LogInfo = func(msg string) {
			logger.Info(msg)
		}
	} else {
		LogInfo = nil
	}
}

// RemoveOldEvents can remove old filesystem events, after a certain duration.
// Needs to be called within a mutex!
func RemoveOldEvents(events *TimeEventMap, maxAge time.Duration) {
	now := time.Now()
	// Cutoff time
	longTimeAgo := now.Add(-maxAge)
	// Loop through the events and delete the old ones
	for t := range *events {
		if t.Before(longTimeAgo) {
			delete(*events, t)
		}
	}
}

// CollectFileChangeEvents collects file change events from the watcher
func CollectFileChangeEvents(watcher *Watcher, mut *sync.Mutex, events TimeEventMap, maxAge time.Duration, filter *EventFilter, webhookManager *WebhookManager) {
	// Start the watcher
	if err := watcher.Start(); err != nil {
		FatalExit(err)
	}

	// Process events from the watcher
	go func() {
		for {
			select {
			case eventBatch := <-watcher.Events():
				// Process each event in the batch
				for _, fsEvent := range eventBatch {
					// Convert to our Event type
					event := Event(fsEvent)

					// Apply filter if provided
					if filter != nil && !filter.ShouldProcessEvent(fsEvent) {
						if LogInfo != nil {
							LogInfo(fmt.Sprintf("Filtered event: %s", event))
						}
						continue
					}

					// Log the event if verbose logging is enabled
					if LogInfo != nil {
						LogInfo(fmt.Sprintf("Event: %s", event))
					}

					// Add the event to the map
					now := time.Now()
					mut.Lock()
					events[now] = event
					// Remove old events
					RemoveOldEvents(&events, maxAge)
					mut.Unlock()

					// Send webhook if configured
					if webhookManager != nil {
						webhookManager.HandleEvent(fsnotify.Event(event))
					}
				}

			case err := <-watcher.Errors():
				if err != nil && LogError != nil {
					LogError(err)
				}
			}
		}
	}()
}

func GenFileChangeEvents(events TimeEventMap, mut *sync.Mutex, maxAge time.Duration, allowed string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream;charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", allowed)

		var id uint64

		for {
			func() { // Use an anonymous function, just for using "defer"
				mut.Lock()
				defer mut.Unlock()
				if len(events) > 0 {
					// Remove old keys
					RemoveOldEvents(&events, maxAge)
					// Sort the events by the registered time
					var keys timeKeys
					for k := range events {
						keys = append(keys, k)
					}
					sort.Sort(keys)
					prevname := ""
					for _, k := range keys {
						ev := events[k]
						if LogInfo != nil {
							LogInfo("EVENT " + ev.String())
						}
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
			time.Sleep(maxAge)
		}
	}
}

// WriteEvent writes SSE events to the given ResponseWriter.
// id can be nil.
func WriteEvent(w http.ResponseWriter, id *uint64, message string, flush bool) {
	var buf bytes.Buffer
	if id != nil {
		buf.WriteString(fmt.Sprintf("id: %v\n", *id))
	}
	for _, msg := range strings.Split(message, "\n") {
		buf.WriteString(fmt.Sprintf("data: %s\n", msg))
	}
	buf.WriteString("\n")
	_, err := io.Copy(w, &buf)
	if err != nil {
		log.Printf("Error writing event: %v", err)
	}
	if flush {
		Flush(w)
	}
}

// Flush can flush the given ResponseWriter.
// Returns false if it wasn't an http.Flusher.
func Flush(w http.ResponseWriter) bool {
	flusher, ok := w.(http.Flusher)
	if ok {
		flusher.Flush()
	}
	return ok
}

// EventServer starts a server that serves events over SSE and/or WebSockets
func EventServer(path, allowed, eventAddr, eventPath string, refreshDuration time.Duration, options ...Option) {
	// Check if the path exists
	if !Exists(path) {
		FatalExit(errors.New("path does not exist: " + path))
	}

	// Parse options
	opts := &Options{}
	for _, option := range options {
		option(opts)
	}

	// Create a new watcher
	config := WatcherConfig{
		RootPath:        path,
		Recursive:       true,
		HandlerDelay:    100 * time.Millisecond,
		PollInterval:    4 * time.Second,
		IncludePatterns: nil,
		ExcludePatterns: nil,
		IncludeEvents:   nil,
		IgnoreEvents:    nil,
	}

	// Apply filter options if provided
	if opts.Filter != nil {
		// Convert filter patterns to string arrays
		if len(opts.Filter.includePatterns) > 0 {
			config.IncludePatterns = make([]string, len(opts.Filter.includePatterns))
			for i, pattern := range opts.Filter.includePatterns {
				// Use pattern as string directly since glob.Glob doesn't have String() method
				config.IncludePatterns[i] = fmt.Sprintf("%v", pattern)
			}
		}

		if len(opts.Filter.excludePatterns) > 0 {
			config.ExcludePatterns = make([]string, len(opts.Filter.excludePatterns))
			for i, pattern := range opts.Filter.excludePatterns {
				// Use pattern as string directly since glob.Glob doesn't have String() method
				config.ExcludePatterns[i] = fmt.Sprintf("%v", pattern)
			}
		}

		// Convert event types to string arrays
		if len(opts.Filter.includeEvents) > 0 {
			events := make([]string, 0, len(opts.Filter.includeEvents))
			for op := range opts.Filter.includeEvents {
				events = append(events, eventOpToString(op))
			}
			config.IncludeEvents = events
		}

		if len(opts.Filter.ignoreEvents) > 0 {
			events := make([]string, 0, len(opts.Filter.ignoreEvents))
			for op := range opts.Filter.ignoreEvents {
				events = append(events, eventOpToString(op))
			}
			config.IgnoreEvents = events
		}
	}

	watcher, err := NewWatcher(config)
	if err != nil {
		FatalExit(err)
	}
	defer watcher.Close()

	// Create a webhook manager if configured
	var webhookManager *WebhookManager
	if opts.WebhookURL != "" {
		webhookManager = NewWebhookManager(WebhookConfig{
			URL:              opts.WebhookURL,
			Method:           opts.WebhookMethod,
			Headers:          opts.WebhookHeaders,
			Timeout:          opts.WebhookTimeout,
			DebounceDuration: opts.WebhookDebounceDuration,
			MaxRetries:       opts.WebhookMaxRetries,
		})
	}

	// Create a context for the streamers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create the appropriate streamer based on the stream method
	var streamer EventStreamer

	streamerOpts := StreamerOptions{
		Address:         eventAddr,
		Path:            eventPath,
		AllowedOrigin:   allowed,
		RefreshDuration: refreshDuration,
		Filter:          opts.Filter,
	}

	switch opts.StreamMethod {
	case StreamMethodSSE:
		streamer = NewSSEStreamer(streamerOpts)
	case StreamMethodWebSocket:
		streamer = NewWebSocketStreamer(streamerOpts)
	case StreamMethodBoth:
		// For WebSocket, use a different path by appending "/ws" to the eventPath
		wsOpts := streamerOpts
		wsOpts.Path = eventPath + "/ws"

		sseStreamer := NewSSEStreamer(streamerOpts)
		wsStreamer := NewWebSocketStreamer(wsOpts)

		streamer = NewMultiStreamer(sseStreamer, wsStreamer)
	default:
		// Default to SSE for backward compatibility
		streamer = NewSSEStreamer(streamerOpts)
	}

	// Start the streamer
	if err := streamer.Start(ctx); err != nil {
		FatalExit(err)
	}

	// Collect events from the watcher and send them to the streamer
	go func() {
		for {
			select {
			case events := <-watcher.Events():
				for _, event := range events {
					// Check if the event should be filtered
					shouldProcess := true
					if opts.Filter != nil {
						shouldProcess = opts.Filter.ShouldProcessEvent(event)
						if !shouldProcess {
							logger.Debugf("Filtered event: %s %s", event.Op, event.Name)
						}
					}

					// Only process the event if it passes the filter
					if shouldProcess {
						// Print the event to the console
						if opts.ShowEvents {
							// Format the event for display
							var eventType string
							switch event.Op {
							case fsnotify.Create:
								eventType = "CREATE"
							case fsnotify.Write:
								eventType = "WRITE"
							case fsnotify.Remove:
								eventType = "REMOVE"
							case fsnotify.Rename:
								eventType = "RENAME"
							case fsnotify.Chmod:
								eventType = "CHMOD"
							default:
								eventType = "UNKNOWN"
							}

							// Get relative path if possible
							relPath := event.Name
							if absPath, err := filepath.Abs(path); err == nil {
								if rel, err := filepath.Rel(absPath, event.Name); err == nil {
									relPath = rel
								}
							}

							// Log the event with colors using zerolog
							logger.Event(eventType, relPath)
						}

						// Send the event to the streamer
						if err := streamer.Send(event); err != nil && LogError != nil {
							LogError(err)
						}

						// Send webhook if configured
						if webhookManager != nil {
							webhookManager.HandleEvent(event)
						}
					}
				}

			case err := <-watcher.Errors():
				if err != nil && LogError != nil {
					LogError(err)
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	// Start the watcher
	if err := watcher.Start(); err != nil {
		FatalExit(err)
	}

	// Block until context is canceled
	<-ctx.Done()
}

// eventOpToString converts an fsnotify.Op to a string
func eventOpToString(op fsnotify.Op) string {
	switch op {
	case fsnotify.Create:
		return "create"
	case fsnotify.Write:
		return "write"
	case fsnotify.Remove:
		return "remove"
	case fsnotify.Rename:
		return "rename"
	case fsnotify.Chmod:
		return "chmod"
	default:
		return ""
	}
}

// Options contains all options for the EventServer
type Options struct {
	// Filter to apply to events
	Filter *EventFilter
	// Webhook URL to send events to
	WebhookURL string
	// HTTP method to use for webhooks
	WebhookMethod string
	// Headers to include in webhook requests
	WebhookHeaders map[string]string
	// Timeout for webhook requests
	WebhookTimeout time.Duration
	// Debounce duration for webhooks
	WebhookDebounceDuration time.Duration
	// Maximum number of retries for webhook requests
	WebhookMaxRetries int
	// Stream method to use
	StreamMethod StreamMethod
	// Show events in the console
	ShowEvents bool
}

// Option is a function that configures Options
type Option func(*Options)

// WithFilter creates an Option that sets the event filter
func WithFilter(filter *EventFilter) Option {
	return func(o *Options) {
		o.Filter = filter
	}
}

// WithWebhook creates an Option that sets the webhook URL and method
func WithWebhook(url, method string, headers map[string]string, timeout, debounceDuration time.Duration, maxRetries int) Option {
	return func(o *Options) {
		o.WebhookURL = url
		o.WebhookMethod = method
		o.WebhookHeaders = headers
		o.WebhookTimeout = timeout
		o.WebhookDebounceDuration = debounceDuration
		o.WebhookMaxRetries = maxRetries
	}
}

// WithWebhookHeaders creates an Option that sets the webhook headers
func WithWebhookHeaders(headers map[string]string) Option {
	return func(o *Options) {
		o.WebhookHeaders = headers
	}
}

// WithWebhookTimeout creates an Option that sets the webhook timeout
func WithWebhookTimeout(timeout time.Duration) Option {
	return func(o *Options) {
		o.WebhookTimeout = timeout
	}
}

// WithWebhookDebounce creates an Option that sets the webhook debounce duration
func WithWebhookDebounce(duration time.Duration) Option {
	return func(o *Options) {
		o.WebhookDebounceDuration = duration
	}
}

// WithWebhookRetries creates an Option that sets the webhook max retries
func WithWebhookRetries(retries int) Option {
	return func(o *Options) {
		o.WebhookMaxRetries = retries
	}
}

// WithStreamMethod creates an Option that sets the stream method
func WithStreamMethod(method StreamMethod) Option {
	return func(o *Options) {
		o.StreamMethod = method
	}
}

// WithShowEvents creates an Option that sets whether to show events in the console
func WithShowEvents(show bool) Option {
	return func(o *Options) {
		o.ShowEvents = show
	}
}

// FilterOption is a function that configures an EventFilter
type FilterOption func(*EventFilter)

// WithIncludePatterns creates a FilterOption that sets the include patterns
func WithIncludePatterns(patterns string) FilterOption {
	return func(f *EventFilter) {
		f.SetIncludePatterns(patterns)
	}
}

// WithExcludePatterns creates a FilterOption that sets the exclude patterns
func WithExcludePatterns(patterns string) FilterOption {
	return func(f *EventFilter) {
		f.SetExcludePatterns(patterns)
	}
}

// WithIncludeEvents creates a FilterOption that sets the include event types
func WithIncludeEvents(events string) FilterOption {
	return func(f *EventFilter) {
		f.SetIncludeEvents(events)
	}
}

// WithIgnoreEvents creates a FilterOption that sets the ignore event types
func WithIgnoreEvents(events string) FilterOption {
	return func(f *EventFilter) {
		f.SetIgnoreEvents(events)
	}
}
