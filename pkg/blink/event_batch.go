package blink

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// EventBatcher batches file system events to reduce redundant processing
type EventBatcher struct {
	// Configuration
	handlerDelay time.Duration

	// State
	events          []fsnotify.Event
	lastHandlerTime time.Time
	handlerLock     sync.Mutex

	// Output channel
	eventChan chan []fsnotify.Event
}

// NewEventBatcher creates a new event batcher
func NewEventBatcher(handlerDelay time.Duration) *EventBatcher {
	if handlerDelay == 0 {
		handlerDelay = 100 * time.Millisecond
	}

	return &EventBatcher{
		handlerDelay:    handlerDelay,
		lastHandlerTime: time.Now(),
		eventChan:       make(chan []fsnotify.Event),
	}
}

// Events returns the channel that receives batched events
func (b *EventBatcher) Events() <-chan []fsnotify.Event {
	return b.eventChan
}

// Add adds an event to the batch
func (b *EventBatcher) Add(event fsnotify.Event) {
	b.handlerLock.Lock()
	defer b.handlerLock.Unlock()

	// Add event to batch
	b.events = append(b.events, event)

	// If a handler is already scheduled, we're done
	if len(b.events) > 1 {
		return
	}

	// Schedule a new handler
	go b.processEvents()
}

// processEvents processes the batched events after a delay
func (b *EventBatcher) processEvents() {
	// Wait for the handler delay to allow batching
	time.Sleep(b.handlerDelay)

	// Process the batch
	b.handlerLock.Lock()
	events := b.events
	b.events = nil
	b.lastHandlerTime = time.Now()
	b.handlerLock.Unlock()

	// Send events to channel
	if len(events) > 0 {
		b.eventChan <- events
	}
}

// AddBatch adds multiple events to the batch
func (b *EventBatcher) AddBatch(events []fsnotify.Event) {
	if len(events) == 0 {
		return
	}

	b.handlerLock.Lock()

	// If no events are currently batched, schedule processing
	needsScheduling := len(b.events) == 0

	// Add all events to the batch
	b.events = append(b.events, events...)

	b.handlerLock.Unlock()

	// Schedule processing if needed
	if needsScheduling {
		go b.processEvents()
	}
}

// Close closes the event batcher
func (b *EventBatcher) Close() {
	close(b.eventChan)
}
