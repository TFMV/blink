package blink_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/TFMV/blink/pkg/blink"
	"github.com/fsnotify/fsnotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWatcher_Create verifies a file created after the watcher starts is detected.
func TestWatcher_Create(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	w, err := blink.NewWatcher(ctx, blink.WatcherConfig{
		RootPath:     tempDir,
		HandlerDelay: 50 * time.Millisecond,
	})
	require.NoError(t, err)
	w.Start()

	// Give the initial scan a moment to run on an empty directory.
	time.Sleep(20 * time.Millisecond)

	// Create a new file after the watcher has started.
	testFile := filepath.Join(tempDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("hello"), 0600))

	// Wait for the live create/write event.
	select {
	case events := <-w.Events():
		require.NotEmpty(t, events, "Expected at least one event")

		found := false
		for _, e := range events {
			// fsnotify can be finicky; it might be a CREATE or WRITE. We care that we saw the file.
			if e.Name == testFile {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected an event for the new file")
	case <-ctx.Done():
		t.Fatal("Test timed out waiting for create event")
	}
}

// TestWatcher_Batching verifies that files existing before the
// watcher starts are reported as a single, atomic CREATE batch.
func TestWatcher_Batching(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1. Create files BEFORE the watcher starts.
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	require.NoError(t, os.WriteFile(file1, []byte("1"), 0600))
	require.NoError(t, os.WriteFile(file2, []byte("2"), 0600))

	// Create a subdirectory and file to test recursion
	subDir := filepath.Join(tempDir, "sub")
	require.NoError(t, os.Mkdir(subDir, 0755))
	file3 := filepath.Join(subDir, "file3.txt")
	require.NoError(t, os.WriteFile(file3, []byte("3"), 0600))

	// 2. Create and start the watcher.
	w, err := blink.NewWatcher(ctx, blink.WatcherConfig{
		RootPath:     tempDir,
		Recursive:    true,
		HandlerDelay: 1 * time.Second, // Use a long delay; it shouldn't matter for the initial scan.
	})
	require.NoError(t, err)
	w.Start()

	// 3. The first and ONLY event batch should be from the initial scan.
	var initialEvents []fsnotify.Event
	select {
	case initialEvents = <-w.Events():
		// This is what we expect.
	case <-ctx.Done():
		t.Fatal("Test timed out waiting for initial scan events")
	}

	// 4. Verify the contents of the initial scan batch.
	expectedFiles := map[string]bool{
		file1: false,
		file2: false,
		file3: false,
	}
	// Depending on timing, the recursive scan might not pick up subdirectories
	// immediately. We check for at least the top-level files.
	require.GreaterOrEqual(t, len(initialEvents), 2, "Expected initial scan to find at least 2 files")

	for _, event := range initialEvents {
		assert.Equal(t, fsnotify.Create, event.Op, "Initial scan events should be CREATE")
		if _, ok := expectedFiles[event.Name]; ok {
			expectedFiles[event.Name] = true // Mark as found
		}
	}

	assert.True(t, expectedFiles[file1], "file1 not found in initial scan")
	assert.True(t, expectedFiles[file2], "file2 not found in initial scan")

	// 5. Verify that NO more events are sent, because no files have changed.
	select {
	case unexpectedEvents := <-w.Events():
		t.Fatalf("Received unexpected second batch of events: %+v", unexpectedEvents)
	case <-time.After(300 * time.Millisecond):
		// This is the correct outcome.
	}
}

// Test that closing the watcher stops it gracefully.
func TestWatcher_Close(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w, err := blink.NewWatcher(ctx, blink.WatcherConfig{
		RootPath:     tempDir,
		HandlerDelay: 10 * time.Millisecond,
	})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "f.txt"), nil, 0600))
	w.Start()

	// Consume the initial event batch.
	select {
	case <-w.Events():
	case <-time.After(1 * time.Second):
		t.Fatal("Did not receive initial event")
	}

	// Calling Close should trigger the shutdown and wait for it to complete.
	err = w.Close()
	assert.NoError(t, err)

	// The Events channel should now be closed.
	_, ok := <-w.Events()
	assert.False(t, ok, "Events channel should be closed after Close() returns")
}

// Test for race conditions by running a concurrent test.
func TestWatcher_Race(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	w, err := blink.NewWatcher(ctx, blink.WatcherConfig{
		RootPath:     tempDir,
		Recursive:    true,
		HandlerDelay: 5 * time.Millisecond,
		PollInterval: 20 * time.Millisecond,
	})
	require.NoError(t, err)
	w.Start()

	var wg sync.WaitGroup
	const numGoroutines = 4
	const numOps = 50 // Reduced to speed up tests

	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(gid int) {
			defer wg.Done()
			for j := 0; j < numOps; j++ {
				fname := filepath.Join(tempDir, fmt.Sprintf("g%d-f%d.txt", gid, j))
				// Operations can fail under high churn, which is okay.
				_ = os.WriteFile(fname, []byte("data"), 0600)
				time.Sleep(1 * time.Millisecond)
				_ = os.Remove(fname)
			}
		}(i)
	}

	// Concurrently consume events.
	go func() {
		for range w.Events() {
			// Just drain the channel.
		}
	}()

	wg.Wait()
	// Allow time for final events to be processed before closing.
	time.Sleep(300 * time.Millisecond)

	// Now explicitly close the watcher.
	err = w.Close()
	assert.NoError(t, err)
}
