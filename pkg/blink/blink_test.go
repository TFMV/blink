package blink

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewRecursiveWatcher tests the creation of a new recursive watcher
func TestNewRecursiveWatcher(t *testing.T) {
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

	// Create a watcher
	watcher, err := NewRecursiveWatcher(tempDir)
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Verify that the watcher was created successfully
	if watcher == nil {
		t.Fatal("Watcher is nil")
	}

	// Verify that the channels were created
	if watcher.Files == nil {
		t.Fatal("Files channel is nil")
	}
	if watcher.Folders == nil {
		t.Fatal("Folders channel is nil")
	}

	// Verify that the directories were added to the watcher
	// We need to drain the Folders channel to check
	folderCount := 0
	timeout := time.After(1 * time.Second)

	// We expect at least the root directory and the 3 subdirectories
	expectedFolders := len(subDirs) + 1

drainLoop:
	for {
		select {
		case <-watcher.Folders:
			folderCount++
		case <-timeout:
			break drainLoop
		default:
			if folderCount >= expectedFolders {
				break drainLoop
			}
			time.Sleep(10 * time.Millisecond)
		}
	}

	if folderCount < expectedFolders {
		t.Errorf("Expected at least %d folders, got %d", expectedFolders, folderCount)
	}
}

// TestShouldIgnoreFile tests the ShouldIgnoreFile function
func TestShouldIgnoreFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		want     bool
	}{
		{"Normal file", "file.txt", false},
		{"Hidden file", ".hidden", true},
		{"Underscore file", "_temp", true},
		{"Normal directory", "dir", false},
		{"Hidden directory", ".git", true},
		{"Underscore directory", "_build", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldIgnoreFile(tt.filename); got != tt.want {
				t.Errorf("ShouldIgnoreFile(%q) = %v, want %v", tt.filename, got, tt.want)
			}
		})
	}
}

// BenchmarkShouldIgnoreFile benchmarks the ShouldIgnoreFile function
func BenchmarkShouldIgnoreFile(b *testing.B) {
	filenames := []string{
		"file.txt",
		".hidden",
		"_temp",
		"dir",
		".git",
		"_build",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, filename := range filenames {
			ShouldIgnoreFile(filename)
		}
	}
}

// BenchmarkSubfolders benchmarks the Subfolders function
func BenchmarkSubfolders(b *testing.B) {
	// Create a temporary directory structure for benchmarking
	tempDir, err := os.MkdirTemp("", "blink-bench-")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a directory structure with 5 levels and 3 directories per level
	createDirTree(b, tempDir, 5, 3)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Subfolders(tempDir)
	}
}

// Helper function to create a directory tree for benchmarking
func createDirTree(b *testing.B, root string, depth, width int) {
	if depth <= 0 {
		return
	}

	for i := 0; i < width; i++ {
		dir := filepath.Join(root, fmt.Sprintf("dir%d", i))
		if err := os.Mkdir(dir, 0755); err != nil {
			b.Fatalf("Failed to create directory: %v", err)
		}
		createDirTree(b, dir, depth-1, width)
	}
}
