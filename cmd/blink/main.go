package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/TFMV/blink/pkg/blink"
)

func main() {
	// Define command-line flags
	path := flag.String("path", ".", "Directory path to watch for changes")
	allowed := flag.String("allowed-origin", "*", "Value for Access-Control-Allow-Origin header")
	eventAddr := flag.String("event-addr", ":12345", "Address to serve events on ([host][:port])")
	eventPath := flag.String("event-path", "/events", "URL path for the event stream")
	refreshDuration := flag.Duration("refresh", 100*time.Millisecond, "Refresh duration for events")
	verbose := flag.Bool("verbose", false, "Enable verbose logging")
	maxProcs := flag.Int("max-procs", runtime.NumCPU(), "Maximum number of CPUs to use (default: all available)")
	help := flag.Bool("help", false, "Show help")

	// Parse command-line flags
	flag.Parse()

	// Show help if requested
	if *help {
		fmt.Println("Blink - File system watcher with event server")
		fmt.Println("\nUsage:")
		flag.PrintDefaults()
		os.Exit(0)
	}

	// Set the maximum number of CPUs to use
	// This can help control resource usage on multi-core systems
	runtime.GOMAXPROCS(*maxProcs)

	// Set verbose mode if requested
	blink.SetVerbose(*verbose)

	// Print startup information
	fmt.Printf("Blink File System Watcher\n")
	fmt.Printf("-------------------------\n")
	fmt.Printf("Watching directory: %s\n", *path)
	fmt.Printf("Serving events at: http://%s%s\n", *eventAddr, *eventPath)
	fmt.Printf("Refresh duration: %v\n", *refreshDuration)
	fmt.Printf("Verbose mode: %v\n", *verbose)
	fmt.Printf("Using %d CPUs\n", *maxProcs)
	fmt.Printf("Press Ctrl+C to exit\n\n")

	// Create a channel to receive OS signals
	sigs := make(chan os.Signal, 1)

	// Register for SIGINT (Ctrl+C) and SIGTERM
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Start the event server in a goroutine
	done := make(chan bool, 1)
	go func() {
		// Start the event server
		blink.EventServer(*path, *allowed, *eventAddr, *eventPath, *refreshDuration)

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
