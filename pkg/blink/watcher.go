package blink

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Default delay before processing events to allow batching
const (
	defaultHandlerDelay = 100 * time.Millisecond
	defaultPollInterval = 4 * time.Second
	// Buffer size for event and error channels to prevent blocking.
	defaultChannelBufferSize = 256
)

// Watcher provides improved file watching capabilities
type Watcher struct {
	watcher *fsnotify.Watcher
	config  WatcherConfig

	// State management, protected by dirLock
	directories map[string]bool
	watches     map[string]bool
	dirLock     sync.Mutex

	// Event batching, protected by eventLock
	events    []fsnotify.Event
	eventLock sync.Mutex

	// Lifecycle and control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Channels for output
	errorChan chan error
	eventChan chan []fsnotify.Event

	// Polling for new files/directories
	pollInterval time.Duration

	// Pre-compiled filter masks
	includeEvents fsnotify.Op
	ignoreEvents  fsnotify.Op
}

// WatcherConfig holds configuration for the watcher
type WatcherConfig struct {
	RootPath              string
	IncludePatterns       []string
	ExcludePatterns       []string
	IncludeEvents         []string // e.g., ["create", "write"]
	IgnoreEvents          []string // e.g., ["chmod"]
	Recursive             bool
	HandlerDelay          time.Duration
	PollInterval          time.Duration
	DisableDefaultExcludes bool // New flag to disable default excludes
}

// NewWatcher creates a new file watcher.
func NewWatcher(ctx context.Context, config WatcherConfig) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	if config.HandlerDelay == 0 {
		config.HandlerDelay = defaultHandlerDelay
	}
	if config.PollInterval == 0 {
		config.PollInterval = defaultPollInterval
	}

	// If default excludes are not disabled, merge them with user-provided excludes.
	if !config.DisableDefaultExcludes {
		excludes := make([]string, 0, len(config.ExcludePatterns)+len(defaultDevExcludePatterns)+len(defaultDevExcludePaths))
		excludes = append(excludes, config.ExcludePatterns...)
		excludes = append(excludes, defaultDevExcludePatterns...)
		excludes = append(excludes, defaultDevExcludePaths...)
		config.ExcludePatterns = excludes
	}

	ctx, cancel := context.WithCancel(ctx)

	w := &Watcher{
		watcher:      fsWatcher,
		config:       config,
		directories:  make(map[string]bool),
		watches:      make(map[string]bool),
		ctx:          ctx,
		cancel:       cancel,
		errorChan:    make(chan error, defaultChannelBufferSize),
		eventChan:    make(chan []fsnotify.Event, defaultChannelBufferSize),
		pollInterval: config.PollInterval,
	}

	// Pre-compile event type filters for efficiency.
	w.includeEvents, err = compileEventTypes(config.IncludeEvents)
	if err != nil {
		return nil, fmt.Errorf("invalid include event type: %w", err)
	}
	w.ignoreEvents, err = compileEventTypes(config.IgnoreEvents)
	if err != nil {
		return nil, fmt.Errorf("invalid ignore event type: %w", err)
	}

	if config.RootPath != "" {
		w.addDirectory(config.RootPath)
	}

	return w, nil
}

// Start begins watching for file changes.
func (w *Watcher) Start() {
	w.wg.Add(1)
	go w.run()
}

// Close stops the watcher and cleans up resources gracefully.
func (w *Watcher) Close() error {
	w.cancel()
	w.wg.Wait()
	return nil
}

// Events returns a channel that receives batched file events.
func (w *Watcher) Events() <-chan []fsnotify.Event {
	return w.eventChan
}

// Errors returns a channel that receives errors.
func (w *Watcher) Errors() <-chan error {
	return w.errorChan
}

// run is the main event loop for the watcher.
func (w *Watcher) run() {
	defer w.wg.Done()
	defer w.watcher.Close()
	defer close(w.eventChan)
	defer close(w.errorChan)

	initialEvents := w.initialScan()
	if len(initialEvents) > 0 {
		select {
		case w.eventChan <- initialEvents:
		case <-w.ctx.Done():
			return
		}
	}

	pollTicker := time.NewTicker(w.pollInterval)
	defer pollTicker.Stop()

	debounceTimer := time.NewTimer(w.config.HandlerDelay)
	if !debounceTimer.Stop() {
		select {
		case <-debounceTimer.C:
		default:
		}
	}

	for {
		select {
		case <-w.ctx.Done():
			w.flushEvents()
			return
		case <-pollTicker.C:
			if w.scanForNewDirs() {
				debounceTimer.Reset(w.config.HandlerDelay)
			}
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if w.handleEvent(event) {
				debounceTimer.Reset(w.config.HandlerDelay)
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			select {
			case w.errorChan <- err:
			case <-w.ctx.Done():
			}
		case <-debounceTimer.C:
			w.flushEvents()
		}
	}
}

func (w *Watcher) initialScan() []fsnotify.Event {
	w.dirLock.Lock()
	defer w.dirLock.Unlock()

	var initialEvents []fsnotify.Event
	for dir := range w.directories {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || path == dir {
				return nil
			}
			if !w.shouldIncludePath(path) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if info.IsDir() {
				if !w.config.Recursive {
					return filepath.SkipDir
				}
				if err := w.watcher.Add(path); err == nil {
					w.watches[path] = true
				}
			} else {
				if _, watched := w.watches[path]; !watched {
					if w.shouldProcessEventType(fsnotify.Create) {
						initialEvents = append(initialEvents, fsnotify.Event{Name: path, Op: fsnotify.Create})
					}
					w.watches[path] = true
				}
			}
			return nil
		})
	}
	return initialEvents
}

func (w *Watcher) scanForNewDirs() bool {
	w.dirLock.Lock()
	defer w.dirLock.Unlock()

	var newDirsFound bool
	for dir := range w.directories {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil || !info.IsDir() || path == dir {
				return nil
			}
			if !w.config.Recursive {
				return filepath.SkipDir
			}
			if w.shouldIncludePath(path) {
				if _, watched := w.watches[path]; !watched {
					if err := w.watcher.Add(path); err == nil {
						w.watches[path] = true
						newDirsFound = true
					}
				}
			}
			return nil
		})
	}
	return newDirsFound
}

func (w *Watcher) handleEvent(event fsnotify.Event) bool {
	info, err := os.Stat(event.Name)
	isDir := err == nil && info.IsDir()

	if isDir {
		w.handleDirectoryEvent(event)
		return false
	} else {
		return w.handleFileEvent(event)
	}
}

func (w *Watcher) handleFileEvent(event fsnotify.Event) bool {
	if !w.shouldIncludePath(event.Name) || !w.shouldProcessEventType(event.Op) {
		return false
	}

	w.queueEvent(event)

	if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		w.dirLock.Lock()
		delete(w.watches, event.Name)
		w.dirLock.Unlock()
	}
	return true
}

func (w *Watcher) handleDirectoryEvent(event fsnotify.Event) {
	if !w.config.Recursive || !w.shouldIncludePath(event.Name) {
		return
	}
	if event.Op&fsnotify.Create != 0 {
		w.addDirectory(event.Name)
	} else if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		w.removeDirectory(event.Name)
	}
}

func (w *Watcher) queueEvent(event fsnotify.Event) {
	w.eventLock.Lock()
	defer w.eventLock.Unlock()
	w.events = append(w.events, event)
}

func (w *Watcher) flushEvents() {
	w.eventLock.Lock()
	if len(w.events) == 0 {
		w.eventLock.Unlock()
		return
	}
	eventsToSend := w.events
	w.events = nil
	w.eventLock.Unlock()

	select {
	case w.eventChan <- eventsToSend:
	case <-w.ctx.Done():
	}
}

func (w *Watcher) addDirectory(path string) {
	w.dirLock.Lock()
	defer w.dirLock.Unlock()

	if _, ok := w.directories[path]; !ok {
		w.directories[path] = true
		if err := w.watcher.Add(path); err == nil {
			w.watches[path] = true
		}
	}
}

func (w *Watcher) removeDirectory(path string) {
	w.dirLock.Lock()
	defer w.dirLock.Unlock()

	delete(w.directories, path)
	delete(w.watches, path)
}

// shouldIncludePath checks if a path should be included based on patterns.
func (w *Watcher) shouldIncludePath(path string) bool {
	// Use the full path for matching, to make ** patterns work correctly.
	normalizedPath := filepath.ToSlash(path)

	for _, pattern := range w.config.ExcludePatterns {
		// Handle base name matching and full path matching
		base := filepath.Base(normalizedPath)
		if (strings.Contains(pattern, "/") || strings.Contains(pattern, "**")) {
			if matched, _ := filepath.Match(pattern, normalizedPath); matched {
				return false
			}
		} else {
			if matched, _ := filepath.Match(pattern, base); matched {
				return false
			}
		}
	}

	if len(w.config.IncludePatterns) > 0 {
		base := filepath.Base(normalizedPath)
		for _, pattern := range w.config.IncludePatterns {
			if matched, _ := filepath.Match(pattern, base); matched {
				return true
			}
		}
		return false
	}

	return true
}

func (w *Watcher) shouldProcessEventType(op fsnotify.Op) bool {
	if w.ignoreEvents&op != 0 {
		return false
	}

	if w.includeEvents != 0 {
		return w.includeEvents&op != 0
	}

	return true
}

func compileEventTypes(eventNames []string) (fsnotify.Op, error) {
	var op fsnotify.Op
	for _, name := range eventNames {
		switch strings.ToLower(name) {
		case "create":
			op |= fsnotify.Create
		case "write":
			op |= fsnotify.Write
		case "remove":
			op |= fsnotify.Remove
		case "rename":
			op |= fsnotify.Rename
		case "chmod":
			op |= fsnotify.Chmod
		default:
			return 0, fmt.Errorf("unknown event type: %q", name)
		}
	}
	return op, nil
}
