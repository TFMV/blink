package blink

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

// TestWatcher tests the creation and basic functionality of the watcher
func TestWatcher(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "blink-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create some subdirectories
	subDirs := []string{
		filepath.Join(tempDir, "dir1"),
		filepath.Join(tempDir, "dir2"),
		filepath.Join(tempDir, "dir3"),
	}

	for _, dir := range subDirs {
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatalf("Failed to create subdirectory: %v", err)
		}
	}

	// Create a watcher configuration
	config := WatcherConfig{
		RootPath:        tempDir,
		Recursive:       true,
		HandlerDelay:    100 * time.Millisecond,
		PollInterval:    1 * time.Second,
		IncludePatterns: []string{"*.txt"},
		ExcludePatterns: []string{"*.tmp"},
	}

	// Create a watcher
	watcher, err := NewWatcher(config)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Verify that the watcher was created successfully
	if watcher == nil {
		t.Fatal("Watcher is nil")
	}

	// Start the watcher
	if err := watcher.Start(); err != nil {
		t.Fatalf("Failed to start watcher: %v", err)
	}

	// Create a file that should be watched
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait for events
	select {
	case events := <-watcher.Events():
		// Verify that we received an event for the test file
		found := false
		for _, event := range events {
			if event.Name == testFile && event.Op&fsnotify.Create != 0 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Did not receive event for test file")
		}
	case err := <-watcher.Errors():
		t.Errorf("Error from watcher: %v", err)
	case <-time.After(2 * time.Second):
		t.Error("Timed out waiting for events")
	}
}

// BenchmarkWatcher benchmarks the watcher implementation
func BenchmarkWatcher(b *testing.B) {
	// Create a temporary directory for benchmarking
	tempDir, err := os.MkdirTemp("", "blink-bench-")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a directory structure with 3 levels and 3 directories per level
	createBenchDirTree(b, tempDir, 3, 3)

	// Create a watcher configuration
	config := WatcherConfig{
		RootPath:     tempDir,
		Recursive:    true,
		HandlerDelay: 100 * time.Millisecond,
		PollInterval: 1 * time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		watcher, err := NewWatcher(config)
		if err != nil {
			b.Fatalf("Failed to create watcher: %v", err)
		}
		watcher.Start()
		watcher.Close()
	}
}

// BenchmarkEventBatcher benchmarks the event batcher
func BenchmarkEventBatcher(b *testing.B) {
	// Create a batcher with a short delay
	batcher := NewEventBatcher(10 * time.Millisecond)

	// Create some test events
	events := []fsnotify.Event{
		{Name: "file1.txt", Op: fsnotify.Create},
		{Name: "file2.txt", Op: fsnotify.Write},
		{Name: "file3.txt", Op: fsnotify.Remove},
		{Name: "file4.txt", Op: fsnotify.Rename},
		{Name: "file5.txt", Op: fsnotify.Chmod},
	}

	// Start a goroutine to consume events
	go func() {
		for range batcher.Events() {
			// Just consume events
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, event := range events {
			batcher.Add(event)
		}
	}
}

// BenchmarkEventFilter benchmarks the event filter
func BenchmarkEventFilter(b *testing.B) {
	// Create a filter
	filter := NewEventFilter()
	filter.SetIncludePatterns("*.js,*.css,*.html")
	filter.SetExcludePatterns("node_modules,*.tmp")
	filter.SetIncludeEvents("write,create")
	filter.SetIgnoreEvents("chmod")

	// Create some test events
	events := []fsnotify.Event{
		{Name: "file.js", Op: fsnotify.Create},
		{Name: "file.css", Op: fsnotify.Write},
		{Name: "file.html", Op: fsnotify.Remove},
		{Name: "file.tmp", Op: fsnotify.Rename},
		{Name: "node_modules/file.js", Op: fsnotify.Chmod},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, event := range events {
			filter.ShouldProcessEvent(event)
		}
	}
}

// BenchmarkEventBatching benchmarks the event batching process
func BenchmarkEventBatching(b *testing.B) {
	// Create a batcher with a short delay
	batcher := NewEventBatcher(1 * time.Millisecond)

	// Create some test events
	events := []fsnotify.Event{
		{Name: "file1.txt", Op: fsnotify.Create},
		{Name: "file2.txt", Op: fsnotify.Write},
		{Name: "file3.txt", Op: fsnotify.Remove},
		{Name: "file4.txt", Op: fsnotify.Rename},
		{Name: "file5.txt", Op: fsnotify.Chmod},
	}

	// Start a goroutine to consume events
	done := make(chan bool)
	go func() {
		count := 0
		for range batcher.Events() {
			count++
			if count >= b.N {
				done <- true
				return
			}
		}
	}()

	// Add events to the batcher
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		batcher.Add(events[i%len(events)])
	}

	// Wait for all events to be processed
	select {
	case <-done:
		// All events processed
	case <-time.After(10 * time.Second):
		b.Fatal("Timed out waiting for events to be processed")
	}
}

// BenchmarkCompareWatchers compares the performance of the watcher with the recursive watcher
func BenchmarkCompareWatchers(b *testing.B) {
	// Create a temporary directory for benchmarking
	tempDir, err := os.MkdirTemp("", "blink-bench-")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a directory structure with 3 levels and 3 directories per level
	createBenchDirTree(b, tempDir, 3, 3)

	b.Run("RecursiveWatcher", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			watcher, err := NewRecursiveWatcher(tempDir)
			if err != nil {
				b.Fatalf("Failed to create watcher: %v", err)
			}
			watcher.Close()
		}
	})

	b.Run("Watcher", func(b *testing.B) {
		config := WatcherConfig{
			RootPath:     tempDir,
			Recursive:    true,
			HandlerDelay: 100 * time.Millisecond,
		}
		for i := 0; i < b.N; i++ {
			watcher, err := NewWatcher(config)
			if err != nil {
				b.Fatalf("Failed to create watcher: %v", err)
			}
			watcher.Start()
			watcher.Close()
		}
	})
}

// BenchmarkFilterPerformance benchmarks the performance of different filtering scenarios
func BenchmarkFilterPerformance(b *testing.B) {
	// Create a filter
	filter := NewEventFilter()

	// Benchmark with different filter configurations
	b.Run("NoFilters", func(b *testing.B) {
		event := fsnotify.Event{Name: "file.js", Op: fsnotify.Create}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filter.ShouldProcessEvent(event)
		}
	})

	b.Run("IncludePatterns", func(b *testing.B) {
		filter.SetIncludePatterns("*.js,*.css,*.html")
		event := fsnotify.Event{Name: "file.js", Op: fsnotify.Create}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filter.ShouldProcessEvent(event)
		}
	})

	b.Run("ExcludePatterns", func(b *testing.B) {
		filter.SetExcludePatterns("node_modules,*.tmp")
		event := fsnotify.Event{Name: "file.js", Op: fsnotify.Create}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filter.ShouldProcessEvent(event)
		}
	})

	b.Run("IncludeEvents", func(b *testing.B) {
		filter.SetIncludeEvents("write,create")
		event := fsnotify.Event{Name: "file.js", Op: fsnotify.Create}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filter.ShouldProcessEvent(event)
		}
	})

	b.Run("IgnoreEvents", func(b *testing.B) {
		filter.SetIgnoreEvents("chmod")
		event := fsnotify.Event{Name: "file.js", Op: fsnotify.Create}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filter.ShouldProcessEvent(event)
		}
	})

	b.Run("AllFilters", func(b *testing.B) {
		filter.SetIncludePatterns("*.js,*.css,*.html")
		filter.SetExcludePatterns("node_modules,*.tmp")
		filter.SetIncludeEvents("write,create")
		filter.SetIgnoreEvents("chmod")
		event := fsnotify.Event{Name: "file.js", Op: fsnotify.Create}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filter.ShouldProcessEvent(event)
		}
	})
}

// Helper function to create a directory tree for benchmarking
func createBenchDirTree(b *testing.B, root string, depth, width int) {
	if depth <= 0 {
		return
	}

	for i := 0; i < width; i++ {
		dir := filepath.Join(root, fmt.Sprintf("dir%d", i))
		if err := os.Mkdir(dir, 0755); err != nil {
			b.Fatalf("Failed to create directory: %v", err)
		}
		createBenchDirTree(b, dir, depth-1, width)
	}
}
