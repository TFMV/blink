package blink

import (
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// EventFilter defines a filter for file system events
type EventFilter struct {
	// Include patterns (e.g., "*.js,*.css")
	IncludePatterns []string
	// Exclude patterns (e.g., "node_modules,*.tmp")
	ExcludePatterns []string
	// Include event types (e.g., "write,create")
	IncludeEvents []fsnotify.Op
	// Ignore event types (e.g., "chmod")
	IgnoreEvents []fsnotify.Op
}

// NewEventFilter creates a new event filter
func NewEventFilter() *EventFilter {
	return &EventFilter{
		IncludePatterns: []string{},
		ExcludePatterns: []string{},
		IncludeEvents:   []fsnotify.Op{},
		IgnoreEvents:    []fsnotify.Op{},
	}
}

// SetIncludePatterns sets the include patterns
func (f *EventFilter) SetIncludePatterns(patterns string) {
	if patterns == "" {
		f.IncludePatterns = []string{}
		return
	}
	f.IncludePatterns = splitAndTrim(patterns)
}

// SetExcludePatterns sets the exclude patterns
func (f *EventFilter) SetExcludePatterns(patterns string) {
	if patterns == "" {
		f.ExcludePatterns = []string{}
		return
	}
	f.ExcludePatterns = splitAndTrim(patterns)
}

// SetIncludeEvents sets the include event types
func (f *EventFilter) SetIncludeEvents(events string) {
	if events == "" {
		f.IncludeEvents = []fsnotify.Op{}
		return
	}
	f.IncludeEvents = parseEventTypes(events)
}

// SetIgnoreEvents sets the ignore event types
func (f *EventFilter) SetIgnoreEvents(events string) {
	if events == "" {
		f.IgnoreEvents = []fsnotify.Op{}
		return
	}
	f.IgnoreEvents = parseEventTypes(events)
}

// ShouldInclude checks if an event should be included based on the filter
func (f *EventFilter) ShouldInclude(event fsnotify.Event) bool {
	// Check event type filters
	if len(f.IncludeEvents) > 0 {
		included := false
		for _, op := range f.IncludeEvents {
			if event.Op&op != 0 {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	for _, op := range f.IgnoreEvents {
		if event.Op&op != 0 {
			return false
		}
	}

	// Check file pattern filters
	if len(f.IncludePatterns) > 0 {
		included := false
		for _, pattern := range f.IncludePatterns {
			matched, err := filepath.Match(pattern, filepath.Base(event.Name))
			if err == nil && matched {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	for _, pattern := range f.ExcludePatterns {
		// Check if the pattern matches the base filename
		matched, err := filepath.Match(pattern, filepath.Base(event.Name))
		if err == nil && matched {
			return false
		}

		// Also check if the pattern is a directory that is part of the path
		if strings.Contains(event.Name, "/"+pattern+"/") || strings.HasPrefix(event.Name, pattern+"/") {
			return false
		}
	}

	return true
}

// Helper function to split a comma-separated string and trim spaces
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Helper function to parse event types from a string
func parseEventTypes(s string) []fsnotify.Op {
	parts := splitAndTrim(s)
	result := make([]fsnotify.Op, 0, len(parts))
	for _, part := range parts {
		switch strings.ToLower(part) {
		case "create":
			result = append(result, fsnotify.Create)
		case "write":
			result = append(result, fsnotify.Write)
		case "remove":
			result = append(result, fsnotify.Remove)
		case "rename":
			result = append(result, fsnotify.Rename)
		case "chmod":
			result = append(result, fsnotify.Chmod)
		}
	}
	return result
}
