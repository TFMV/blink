package blink

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type (
	// TimeEventMap stores filesystem events
	TimeEventMap map[time.Time]Event
)

// Customizable functions, as exported variables. Can be se to "nil".

// LogInfo logs a message as information
var LogInfo = func(msg string) {
	log.Println(msg)
}

// LogError logs a message as an error, but does not end the program
var LogError = func(err error) {
	log.Println(err.Error())
}

// FatalExit ends the program after logging a message
var FatalExit = func(err error) {
	log.Fatalln(err)
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
			log.Println(msg)
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

// CollectFileChangeEvents gathers filesystem events in a way that web handle functions can use.
// Performance improvements:
// 1. Uses a buffered channel for event processing to handle bursts of events
// 2. Implements event debouncing to reduce duplicate events
// 3. Uses a separate goroutine for event processing to avoid blocking the main thread
// 4. Implements periodic cleanup of old events to prevent memory leaks
func CollectFileChangeEvents(watcher *RecursiveWatcher, mut *sync.Mutex, events TimeEventMap, maxAge time.Duration, filter *EventFilter, webhookManager *WebhookManager) {
	// Create a buffered channel for event processing
	// This allows handling bursts of events without blocking
	eventChan := make(chan fsnotify.Event, 100)

	// Start a goroutine to collect events from the watcher
	go func() {
		for {
			select {
			case ev := <-watcher.Events:
				// Apply filter if provided
				if filter != nil && !filter.ShouldInclude(ev) {
					// Skip this event
					continue
				}

				// Send the event to the processing channel
				// Use non-blocking send to avoid getting stuck if the channel is full
				select {
				case eventChan <- ev:
					// Successfully sent
				default:
					// Channel is full, log but don't block
					if LogError != nil {
						LogError(errors.New("event channel is full, dropping event"))
					}
				}

				// Send to webhook manager if provided
				if webhookManager != nil {
					webhookManager.HandleEvent(ev)
				}
			case err := <-watcher.Errors:
				if LogError != nil {
					LogError(err)
				}
			}
		}
	}()

	// Start a goroutine to process events
	go func() {
		// Create a map to track recent events for debouncing
		// This helps reduce duplicate events for the same file
		recentEvents := make(map[string]time.Time)

		// Create a ticker for periodic cleanup of old events
		cleanupTicker := time.NewTicker(maxAge)
		defer cleanupTicker.Stop()

		// Debounce duration - only process one event per file within this time window
		debounceDuration := 50 * time.Millisecond

		for {
			select {
			case ev := <-eventChan:
				// Check if we've seen this file recently (debouncing)
				now := time.Now()
				lastSeen, exists := recentEvents[ev.Name]

				// Only process if we haven't seen this file recently
				if !exists || now.Sub(lastSeen) > debounceDuration {
					// Update the last seen time
					recentEvents[ev.Name] = now

					// Process the event
					mut.Lock()
					// Remove old events
					RemoveOldEvents(&events, maxAge)
					// Save the event with the current time
					events[now] = Event(ev)
					mut.Unlock()

					// Log the event if verbose mode is enabled
					if LogInfo != nil {
						LogInfo("File event: " + ev.String())
					}
				}

			case <-cleanupTicker.C:
				// Periodically clean up old events from the debounce map
				now := time.Now()
				for file, lastSeen := range recentEvents {
					if now.Sub(lastSeen) > maxAge {
						delete(recentEvents, file)
					}
				}

				// Also clean up the events map
				mut.Lock()
				RemoveOldEvents(&events, maxAge)
				mut.Unlock()
			}
		}
	}()
}

// GenFileChangeEvents creates an SSE event whenever a file in the server directory changes.
//
// Uses the following HTTP headers:
//
//	Content-Type: text/event-stream;charset=utf-8
//	Cache-Control: no-cache
//	Connection: keep-alive
//	Access-Control-Allow-Origin: (custom value)
//
// The "Access-Control-Allow-Origin" header uses the value that is passed in the "allowed" argument.
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
	io.Copy(w, &buf)
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

// EventServer serves events on a dedicated port.
// addr is the host address ([host][:port])
// The filesystem events are gathered independently of that.
// Allowed can be "*" or a hostname and sets a header in the SSE stream.
func EventServer(path, allowed, eventAddr, eventPath string, refreshDuration time.Duration, options ...Option) {

	if !Exists(path) {
		if FatalExit != nil {
			FatalExit(errors.New(path + " does not exist, can't watch"))
		}
	}

	// Create a new filesystem watcher
	rw, err := NewRecursiveWatcher(path)
	if err != nil {
		if FatalExit != nil {
			FatalExit(err)
		}
	}

	var mut sync.Mutex
	events := make(TimeEventMap)

	// Create and configure options
	opts := &Options{}
	for _, option := range options {
		option(opts)
	}

	// Create webhook manager if webhook URL is provided
	var webhookManager *WebhookManager
	if opts.WebhookURL != "" {
		webhookConfig := WebhookConfig{
			URL:              opts.WebhookURL,
			Method:           opts.WebhookMethod,
			Headers:          opts.WebhookHeaders,
			Timeout:          opts.WebhookTimeout,
			DebounceDuration: opts.WebhookDebounceDuration,
			MaxRetries:       opts.WebhookMaxRetries,
			Filter:           opts.Filter,
		}
		webhookManager = NewWebhookManager(webhookConfig)
		if LogInfo != nil {
			LogInfo(fmt.Sprintf("Webhook configured for URL: %s", opts.WebhookURL))
		}
	}

	// Collect the events for the last n seconds, repeatedly
	// Runs in the background
	CollectFileChangeEvents(rw, &mut, events, refreshDuration, opts.Filter, webhookManager)

	// Serve events
	go func() {
		eventMux := http.NewServeMux()
		// Fire off events whenever a file in the server directory changes
		eventMux.HandleFunc(eventPath, GenFileChangeEvents(events, &mut, refreshDuration, allowed))
		eventServer := &http.Server{
			Addr:    eventAddr,
			Handler: eventMux,
		}
		if err := eventServer.ListenAndServe(); err != nil {
			// If we can't serve HTTP on this port, give up
			if FatalExit != nil {
				FatalExit(err)
			}
		}
	}()
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
}

// Option is a function that configures Options
type Option func(*Options)

// WithFilter creates an Option that sets the event filter
func WithFilter(filter *EventFilter) Option {
	return func(o *Options) {
		o.Filter = filter
	}
}

// WithWebhook creates an Option that configures a webhook
func WithWebhook(url string, method string) Option {
	return func(o *Options) {
		o.WebhookURL = url
		o.WebhookMethod = method
	}
}

// WithWebhookHeaders creates an Option that sets webhook headers
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

// WithWebhookRetries creates an Option that sets the maximum number of webhook retries
func WithWebhookRetries(maxRetries int) Option {
	return func(o *Options) {
		o.WebhookMaxRetries = maxRetries
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
