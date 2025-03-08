package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/TFMV/blink/pkg/blink"
)

func main() {
	// Set verbose mode to see events in the console
	blink.SetVerbose(true)

	// Directory to watch
	path := "."

	// Configure the number of CPUs to use
	// This can help control resource usage on multi-core systems
	maxProcs := runtime.NumCPU()
	if maxProcs > 4 {
		maxProcs = 4 // Limit to 4 CPUs for this example
	}
	runtime.GOMAXPROCS(maxProcs)

	// Print startup information
	fmt.Printf("Blink Example\n")
	fmt.Printf("------------\n")
	fmt.Printf("Watching directory: %s\n", path)
	fmt.Printf("Serving events at: http://localhost:12345/events\n")
	fmt.Printf("Using %d CPUs\n", maxProcs)
	fmt.Printf("Press Ctrl+C to stop\n\n")

	// Create a channel to receive OS signals
	sigs := make(chan os.Signal, 1)

	// Register for SIGINT (Ctrl+C) and SIGTERM
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Start the event server in a goroutine
	done := make(chan bool, 1)
	go func() {
		// Start the event server with default settings
		blink.EventServer(
			path,                 // Directory to watch
			"*",                  // Allow all origins
			":12345",             // Listen on port 12345
			"/events",            // Event path
			100*time.Millisecond, // Refresh duration
		)

		// Signal that we're done (this should never happen unless EventServer returns)
		done <- true
	}()

	// Wait for a signal or for the server to exit
	select {
	case sig := <-sigs:
		fmt.Printf("\nReceived signal %v, shutting down...\n", sig)
	case <-done:
		fmt.Println("\nServer exited unexpectedly")
	}

	// Perform any cleanup here if needed
	fmt.Println("Goodbye!")
}
