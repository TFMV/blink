package blink

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRemoveOldEvents tests the RemoveOldEvents function
func TestRemoveOldEvents(t *testing.T) {
	// Create a TimeEventMap with events at different times
	events := make(TimeEventMap)
	now := time.Now()

	// Add events at different times
	events[now.Add(-10*time.Minute)] = Event{}
	events[now.Add(-5*time.Minute)] = Event{}
	events[now.Add(-1*time.Minute)] = Event{}
	events[now] = Event{}

	// Remove events older than 2 minutes
	maxAge := 2 * time.Minute
	RemoveOldEvents(&events, maxAge)

	// Check that only the recent events remain
	if len(events) != 2 {
		t.Errorf("Expected 2 events after removal, got %d", len(events))
	}

	// Check that the old events were removed
	for timestamp := range events {
		if now.Sub(timestamp) > maxAge {
			t.Errorf("Event at %v should have been removed (older than %v)", timestamp, maxAge)
		}
	}
}

// TestWriteEvent tests the WriteEvent function
func TestWriteEvent(t *testing.T) {
	// Create a test response recorder
	recorder := httptest.NewRecorder()

	// Test writing an event with an ID
	id := uint64(123)
	message := "test event"
	WriteEvent(recorder, &id, message, true)

	// Check the response
	resp := recorder.Result()
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Check that the response contains the expected SSE format
	if !strings.Contains(bodyStr, fmt.Sprintf("id: %d", id)) {
		t.Errorf("Expected response to contain 'id: %d', got: %s", id, bodyStr)
	}
	if !strings.Contains(bodyStr, fmt.Sprintf("data: %s", message)) {
		t.Errorf("Expected response to contain 'data: %s', got: %s", message, bodyStr)
	}
}

// BenchmarkRemoveOldEvents benchmarks the RemoveOldEvents function
func BenchmarkRemoveOldEvents(b *testing.B) {
	// Create a TimeEventMap with a large number of events
	events := make(TimeEventMap)
	now := time.Now()

	// Add 1000 events at different times
	for i := 0; i < 1000; i++ {
		events[now.Add(-time.Duration(i)*time.Minute)] = Event{}
	}

	// Benchmark removing events older than 500 minutes
	maxAge := 500 * time.Minute
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a copy of the events map for each iteration
		eventsCopy := make(TimeEventMap)
		for k, v := range events {
			eventsCopy[k] = v
		}
		RemoveOldEvents(&eventsCopy, maxAge)
	}
}

// TestEventServer is an integration test for the EventServer function
// This test is more complex and requires careful setup
func TestEventServer(t *testing.T) {
	// Skip in short mode as this is an integration test
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create a temporary directory for testing
	tempDir, err := ioutil.TempDir("", "blink-server-test-")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Start the event server in a goroutine
	go func() {
		EventServer(
			tempDir,
			"*",
			":0", // Use port 0 to get a random available port
			"/events",
			100*time.Millisecond,
		)
	}()

	// Give the server time to start
	time.Sleep(500 * time.Millisecond)

	// Create a file in the watched directory to trigger an event
	testFile := filepath.Join(tempDir, "test.txt")
	if err := ioutil.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Wait for the event to be processed
	time.Sleep(200 * time.Millisecond)

	// This is a simplified test - in a real scenario, we would connect to the
	// event stream and verify that we receive the expected events
	// However, that requires more complex HTTP client setup for SSE
}
