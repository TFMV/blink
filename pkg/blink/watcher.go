package blink

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Default delay before processing events to allow batching
const defaultHandlerDelay = 100 * time.Millisecond

// Watcher provides improved file watching capabilities
type Watcher struct {
	// Underlying fsnotify watcher
	watcher *fsnotify.Watcher

	// Configuration
	config WatcherConfig

	// State management
	directories map[string]bool
	watches     map[string]bool
	dirLock     sync.Mutex
	handlerLock sync.Mutex

	// Event handling
	events          []fsnotify.Event
	lastHandlerTime time.Time

	// Control channels
	closeChan chan bool
	errorChan chan error
	eventChan chan []fsnotify.Event

	// Polling for new files/directories
	pollInterval time.Duration
}

// WatcherConfig holds configuration for the watcher
type WatcherConfig struct {
	// Root directory to watch
	RootPath string

	// Patterns to include/exclude
	IncludePatterns []string
	ExcludePatterns []string

	// Event types to include/ignore
	IncludeEvents []string
	IgnoreEvents  []string

	// Whether to watch recursively
	Recursive bool

	// Delay before handling events (for batching)
	HandlerDelay time.Duration

	// Polling interval for checking new files
	PollInterval time.Duration
}

// NewWatcher creates a new file watcher
func NewWatcher(config WatcherConfig) (*Watcher, error) {
	// Create fsnotify watcher
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	// Set default handler delay if not specified
	if config.HandlerDelay == 0 {
		config.HandlerDelay = defaultHandlerDelay
	}

	// Set default polling interval if not specified
	if config.PollInterval == 0 {
		config.PollInterval = 4 * time.Second
	}

	// Create watcher
	watcher := &Watcher{
		watcher:         fsWatcher,
		config:          config,
		directories:     make(map[string]bool),
		watches:         make(map[string]bool),
		lastHandlerTime: time.Now(),
		closeChan:       make(chan bool),
		errorChan:       make(chan error),
		eventChan:       make(chan []fsnotify.Event),
		pollInterval:    config.PollInterval,
	}

	// Add root directory
	watcher.addDirectory(config.RootPath)

	return watcher, nil
}

// Start begins watching for file changes
func (w *Watcher) Start() error {
	// Add initial watches
	newWatchPaths := w.addWatches()

	// Schedule create events for initial files
	w.scheduleCreateEvents(newWatchPaths)

	// Start goroutine for handling events
	go w.watchLoop()

	return nil
}

// Close stops the watcher
func (w *Watcher) Close() error {
	w.closeChan <- true
	return w.watcher.Close()
}

// Events returns a channel that receives batched file events
func (w *Watcher) Events() <-chan []fsnotify.Event {
	return w.eventChan
}

// Errors returns a channel that receives errors
func (w *Watcher) Errors() <-chan error {
	return w.errorChan
}

// watchLoop is the main event loop
func (w *Watcher) watchLoop() {
	for {
		select {
		case <-time.After(w.pollInterval):
			// Poll for new files/directories
			newWatchPaths := w.addWatches()
			w.scheduleCreateEvents(newWatchPaths)

		case event := <-w.watcher.Events:
			// Handle file system event
			if err := w.handleEvent(event); err != nil {
				w.errorChan <- err
			}

		case err := <-w.watcher.Errors:
			// Forward errors
			if err != nil {
				w.errorChan <- err
			}

		case <-w.closeChan:
			// Exit loop when closed
			return
		}
	}
}

// handleEvent processes a single fsnotify event
func (w *Watcher) handleEvent(event fsnotify.Event) error {
	// Check if this is a directory
	isDir := w.isDirectory(event.Name)

	// Handle based on whether it's a file or directory
	if isDir {
		return w.handleDirectoryEvent(event)
	}

	return w.handleFileEvent(event)
}

// handleFileEvent handles events for regular files
func (w *Watcher) handleFileEvent(event fsnotify.Event) error {
	// Check if the file matches our patterns
	if !w.shouldIncludePath(event.Name) {
		return nil
	}

	// Check if the event type should be processed
	if !w.shouldProcessEventType(event.Op) {
		return nil
	}

	// Schedule the event for handling
	w.scheduleHandler(event)

	// If file was removed or renamed, update our watch list
	if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		w.dirLock.Lock()
		delete(w.watches, event.Name)
		w.dirLock.Unlock()
	}

	return nil
}

// handleDirectoryEvent handles events for directories
func (w *Watcher) handleDirectoryEvent(event fsnotify.Event) error {
	// Only process if we're watching recursively
	if !w.config.Recursive {
		return nil
	}

	// Check if the directory matches our patterns
	if !w.shouldIncludePath(event.Name) {
		return nil
	}

	// Handle based on event type
	if event.Op&fsnotify.Create != 0 {
		// New directory created, add it to our watch list
		w.addDirectory(event.Name)
	} else if event.Op&fsnotify.Remove != 0 {
		// Directory removed, remove from our watch list
		w.removeDirectory(event.Name)
	}

	return nil
}

// scheduleHandler batches events and schedules them for processing
func (w *Watcher) scheduleHandler(event fsnotify.Event) {
	w.handlerLock.Lock()
	defer w.handlerLock.Unlock()

	// Add event to batch
	w.events = append(w.events, event)

	// If a handler is already scheduled, we're done
	if len(w.events) > 1 {
		return
	}

	// Schedule a new handler
	go func() {
		// Wait for the handler delay to allow batching
		time.Sleep(w.config.HandlerDelay)

		// Process the batch
		w.handlerLock.Lock()
		events := w.events
		w.events = nil
		w.lastHandlerTime = time.Now()
		w.handlerLock.Unlock()

		// Send events to channel
		if len(events) > 0 {
			w.eventChan <- events
		}
	}()
}

// addWatches adds watches for all files in watched directories
func (w *Watcher) addWatches() []string {
	w.dirLock.Lock()
	defer w.dirLock.Unlock()

	var newWatchPaths []string

	// Process each directory
	for dir := range w.directories {
		// Walk the directory
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip files we can't access
			}

			// Skip directories if not recursive
			if info.IsDir() && path != dir && !w.config.Recursive {
				return filepath.SkipDir
			}

			// Check if path should be included
			if !w.shouldIncludePath(path) {
				if info.IsDir() && path != dir {
					return filepath.SkipDir
				}
				return nil
			}

			// Add watch if not already watching
			if !w.watches[path] {
				if err := w.watcher.Add(path); err == nil {
					w.watches[path] = true
					if !info.IsDir() {
						newWatchPaths = append(newWatchPaths, path)
					}
				}
			}

			return nil
		})
	}

	return newWatchPaths
}

// scheduleCreateEvents schedules CREATE events for new files
func (w *Watcher) scheduleCreateEvents(paths []string) {
	for _, path := range paths {
		w.scheduleHandler(fsnotify.Event{
			Name: path,
			Op:   fsnotify.Create,
		})
	}
}

// addDirectory adds a directory to the watch list
func (w *Watcher) addDirectory(path string) {
	w.dirLock.Lock()
	defer w.dirLock.Unlock()

	w.directories[path] = true

	// Add watch for the directory itself
	if err := w.watcher.Add(path); err == nil {
		w.watches[path] = true
	}
}

// removeDirectory removes a directory from the watch list
func (w *Watcher) removeDirectory(path string) {
	w.dirLock.Lock()
	defer w.dirLock.Unlock()

	delete(w.directories, path)
	delete(w.watches, path)
}

// isDirectory checks if a path is a directory
func (w *Watcher) isDirectory(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// shouldIncludePath checks if a path should be included based on patterns
func (w *Watcher) shouldIncludePath(path string) bool {
	// TODO: Implement pattern matching logic using w.config.IncludePatterns and w.config.ExcludePatterns
	// For now, include everything
	return true
}

// shouldProcessEventType checks if an event type should be processed
func (w *Watcher) shouldProcessEventType(eventType fsnotify.Op) bool {
	// TODO: Implement event type filtering using w.config.IncludeEvents and w.config.IgnoreEvents
	// For now, process all events
	return true
}
