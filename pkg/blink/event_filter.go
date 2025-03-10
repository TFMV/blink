package blink

import (
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// EventFilter provides filtering capabilities for file system events
type EventFilter struct {
	// Include/exclude patterns
	includePatterns []string
	excludePatterns []string

	// Event types to include/ignore
	includeEvents map[fsnotify.Op]bool
	ignoreEvents  map[fsnotify.Op]bool
}

// NewEventFilter creates a new event filter
func NewEventFilter() *EventFilter {
	return &EventFilter{
		includePatterns: []string{},
		excludePatterns: []string{},
		includeEvents:   make(map[fsnotify.Op]bool),
		ignoreEvents:    make(map[fsnotify.Op]bool),
	}
}

// SetIncludePatterns sets the patterns for files to include
func (f *EventFilter) SetIncludePatterns(patterns string) {
	f.includePatterns = parsePatterns(patterns)
}

// SetExcludePatterns sets the patterns for files to exclude
func (f *EventFilter) SetExcludePatterns(patterns string) {
	f.excludePatterns = parsePatterns(patterns)
}

// SetIncludeEvents sets the event types to include
func (f *EventFilter) SetIncludeEvents(events string) {
	f.includeEvents = parseEvents(events)
}

// SetIgnoreEvents sets the event types to ignore
func (f *EventFilter) SetIgnoreEvents(events string) {
	f.ignoreEvents = parseEvents(events)
}

// ShouldIncludePath checks if a path should be included based on patterns
func (f *EventFilter) ShouldIncludePath(path string) bool {
	// If no include patterns are specified, include everything
	includeAll := len(f.includePatterns) == 0

	// Check exclude patterns first
	for _, pattern := range f.excludePatterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return false
		}
	}

	// If we're including all and nothing was excluded, include the path
	if includeAll {
		return true
	}

	// Check include patterns
	for _, pattern := range f.includePatterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err == nil && matched {
			return true
		}
	}

	return false
}

// ShouldProcessEvent checks if an event should be processed based on event type and path
func (f *EventFilter) ShouldProcessEvent(event fsnotify.Event) bool {
	// Check if the path should be included
	if !f.ShouldIncludePath(event.Name) {
		return false
	}

	// Check if the event type should be ignored
	if len(f.ignoreEvents) > 0 {
		for op := range f.ignoreEvents {
			if event.Op&op != 0 {
				return false
			}
		}
	}

	// Check if the event type should be included
	if len(f.includeEvents) > 0 {
		for op := range f.includeEvents {
			if event.Op&op != 0 {
				return true
			}
		}
		// If include events are specified but none match, exclude the event
		return false
	}

	// If no include events are specified, include all events that weren't ignored
	return true
}

// parsePatterns converts a comma-separated string of patterns to a slice of strings
func parsePatterns(patterns string) []string {
	if patterns == "" {
		return []string{}
	}

	patternList := strings.Split(patterns, ",")
	result := make([]string, 0, len(patternList))

	for _, p := range patternList {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}

	return result
}

// parseEvents converts a comma-separated string of event types to a map
func parseEvents(events string) map[fsnotify.Op]bool {
	result := make(map[fsnotify.Op]bool)

	if events == "" {
		return result
	}

	eventList := strings.Split(events, ",")

	for _, e := range eventList {
		e = strings.TrimSpace(e)
		switch strings.ToLower(e) {
		case "create":
			result[fsnotify.Create] = true
		case "write":
			result[fsnotify.Write] = true
		case "remove":
			result[fsnotify.Remove] = true
		case "rename":
			result[fsnotify.Rename] = true
		case "chmod":
			result[fsnotify.Chmod] = true
		}
	}

	return result
}
