package blink

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/TFMV/blink/pkg/logger"
	"github.com/fsnotify/fsnotify"
)

// CustomFilterFunc is a function type for custom path filtering
type CustomFilterFunc func(path string, isDir bool) bool

// EventFilter provides filtering capabilities for file system events
type EventFilter struct {
	// Include/exclude patterns
	includePatterns []string
	excludePatterns []string

	// Event types to include/ignore
	includeEvents map[fsnotify.Op]bool
	ignoreEvents  map[fsnotify.Op]bool

	// Custom filter functions
	customFilters []CustomFilterFunc
}

// NewEventFilter creates a new event filter
func NewEventFilter() *EventFilter {
	return &EventFilter{
		includePatterns: []string{},
		excludePatterns: []string{},
		includeEvents:   make(map[fsnotify.Op]bool),
		ignoreEvents:    make(map[fsnotify.Op]bool),
		customFilters:   []CustomFilterFunc{},
	}
}

// AddCustomFilter adds a custom filter function
func (f *EventFilter) AddCustomFilter(filter CustomFilterFunc) {
	f.customFilters = append(f.customFilters, filter)
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

// GetExcludePatterns returns the current exclude patterns as a comma-separated string
func (f *EventFilter) GetExcludePatterns() string {
	return strings.Join(f.excludePatterns, ",")
}

// ShouldIncludePath checks if a path should be included based on patterns
func (f *EventFilter) ShouldIncludePath(path string) bool {
	// Normalize path for consistent matching
	normalizedPath := filepath.ToSlash(path)
	baseName := filepath.Base(normalizedPath)

	// If no include patterns are specified, include everything
	includeAll := len(f.includePatterns) == 0

	// Check exclude patterns first
	for _, pattern := range f.excludePatterns {
		// Try to match against the base name first (simple case)
		matched, err := filepath.Match(pattern, baseName)
		if err == nil && matched {
			logger.Debugf("Path excluded by base name pattern: %s (pattern: %s)", path, pattern)
			return false
		}

		// Handle ** pattern (recursive matching)
		if strings.Contains(pattern, "**") {
			// Convert ** to a more specific check
			parts := strings.Split(pattern, "**")
			allPartsMatch := true

			// Check if all parts of the pattern exist in the path in the correct order
			lastIndex := 0
			for _, part := range parts {
				if part == "" {
					continue
				}

				index := strings.Index(normalizedPath[lastIndex:], part)
				if index == -1 {
					allPartsMatch = false
					break
				}
				lastIndex += index + len(part)
			}

			if allPartsMatch {
				logger.Debugf("Path excluded by ** pattern: %s (pattern: %s)", path, pattern)
				return false
			}
		}

		// Handle * pattern (single directory matching)
		if strings.Contains(pattern, "*") {
			// Try matching against the full path
			// This is a simplified approach - for complex cases, consider using a library like doublestar
			if matched, _ := filepath.Match(pattern, normalizedPath); matched {
				logger.Debugf("Path excluded by full path pattern: %s (pattern: %s)", path, pattern)
				return false
			}

			// Try matching against path segments
			pathParts := strings.Split(normalizedPath, "/")
			for _, part := range pathParts {
				if matched, _ := filepath.Match(pattern, part); matched {
					logger.Debugf("Path excluded by path segment pattern: %s (pattern: %s, segment: %s)", path, pattern, part)
					return false
				}
			}
		}

		// Direct substring match for non-glob patterns
		if !strings.Contains(pattern, "*") && strings.Contains(normalizedPath, pattern) {
			logger.Debugf("Path excluded by substring match: %s (pattern: %s)", path, pattern)
			return false
		}
	}

	// If we're including all and nothing was excluded, include the path
	if includeAll {
		return true
	}

	// Check include patterns
	for _, pattern := range f.includePatterns {
		// Try to match against the base name first (simple case)
		matched, err := filepath.Match(pattern, baseName)
		if err == nil && matched {
			logger.Debugf("Path included by base name pattern: %s (pattern: %s)", path, pattern)
			return true
		}

		// Handle ** pattern (recursive matching)
		if strings.Contains(pattern, "**") {
			// Convert ** to a more specific check
			parts := strings.Split(pattern, "**")
			allPartsMatch := true

			// Check if all parts of the pattern exist in the path in the correct order
			lastIndex := 0
			for _, part := range parts {
				if part == "" {
					continue
				}

				index := strings.Index(normalizedPath[lastIndex:], part)
				if index == -1 {
					allPartsMatch = false
					break
				}
				lastIndex += index + len(part)
			}

			if allPartsMatch {
				logger.Debugf("Path included by ** pattern: %s (pattern: %s)", path, pattern)
				return true
			}
		}

		// Handle * pattern (single directory matching)
		if strings.Contains(pattern, "*") {
			// Try matching against the full path
			if matched, _ := filepath.Match(pattern, normalizedPath); matched {
				logger.Debugf("Path included by full path pattern: %s (pattern: %s)", path, pattern)
				return true
			}

			// Try matching against path segments
			pathParts := strings.Split(normalizedPath, "/")
			for _, part := range pathParts {
				if matched, _ := filepath.Match(pattern, part); matched {
					logger.Debugf("Path included by path segment pattern: %s (pattern: %s, segment: %s)", path, pattern, part)
					return true
				}
			}
		}

		// Direct substring match for non-glob patterns
		if !strings.Contains(pattern, "*") && strings.Contains(normalizedPath, pattern) {
			logger.Debugf("Path included by substring match: %s (pattern: %s)", path, pattern)
			return true
		}
	}

	return false
}

// ShouldProcessEvent checks if an event should be processed based on event type and path
func (f *EventFilter) ShouldProcessEvent(event fsnotify.Event) bool {
	// Debug the event being processed
	logger.Debugf("Processing event: %s, path: %s", event.Op, event.Name)

	// Check custom filters first - these have highest priority
	for i, filter := range f.customFilters {
		if filter != nil {
			// Get file info to determine if it's a directory
			isDir := false
			fileInfo, err := os.Stat(event.Name)
			if err == nil {
				isDir = fileInfo.IsDir()
			} else {
				logger.Debugf("Error getting file info for %s: %v", event.Name, err)
			}

			// Apply the custom filter
			result := filter(event.Name, isDir)
			logger.Debugf("Custom filter %d result for %s: %v", i, event.Name, result)
			if !result {
				logger.Debugf("Event excluded by custom filter %d: %s", i, event.Name)
				return false
			}
		}
	}

	// Check if the path should be included based on patterns
	if !f.ShouldIncludePath(event.Name) {
		logger.Debugf("Event excluded by pattern: %s", event.Name)
		return false
	}

	// Check if the event type should be ignored
	if len(f.ignoreEvents) > 0 {
		for op := range f.ignoreEvents {
			if event.Op&op != 0 {
				logger.Debugf("Event excluded by ignored event type: %s %s", event.Op, event.Name)
				return false
			}
		}
	}

	// Check if the event type should be included
	if len(f.includeEvents) > 0 {
		for op := range f.includeEvents {
			if event.Op&op != 0 {
				logger.Debugf("Event included by event type: %s %s", event.Op, event.Name)
				return true
			}
		}
		// If include events are specified but none match, exclude the event
		logger.Debugf("Event excluded because no include event types matched: %s %s", event.Op, event.Name)
		return false
	}

	// If no include events are specified, include all events that weren't ignored
	logger.Debugf("Event included by default: %s %s", event.Op, event.Name)
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
			// Normalize pattern to use forward slashes
			p = filepath.ToSlash(p)

			// Handle special case for **/ prefix (match any directory)
			if strings.HasPrefix(p, "**/") && !strings.HasPrefix(p, "**/*") {
				// Add both the original pattern and a version without the **/ prefix
				// This helps match both absolute and relative paths
				result = append(result, p)
				result = append(result, p[3:])
				continue
			}

			// Handle special case for /** suffix (match any subdirectory)
			if strings.HasSuffix(p, "/**") && !strings.HasSuffix(p, "/**/*") {
				// Add both the original pattern and a version with /** replaced by /**/*
				// This helps match files in subdirectories
				result = append(result, p)
				result = append(result, p+"/*")
				continue
			}

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
