package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

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
	fmt.Printf("Using %d CPUs\n", maxProcs)
	fmt.Printf("Press Ctrl+C to stop\n\n")

	// Create a new filesystem watcher
	watcher, err := blink.NewRecursiveWatcher(path)
	if err != nil {
		fmt.Printf("Error creating watcher: %v\n", err)
		os.Exit(1)
	}
	defer watcher.Close()

	// Create a channel to receive OS signals
	sigs := make(chan os.Signal, 1)

	// Register for SIGINT (Ctrl+C) and SIGTERM
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Create a channel to track events
	events := make(chan blink.Event, 100)

	// Start a goroutine to collect events
	go func() {
		for {
			select {
			case ev := <-watcher.Events:
				event := blink.Event(ev)
				events <- event
				fmt.Printf("Event: %s %s\n", event.Op.String(), event.Name)
			case err := <-watcher.Errors:
				fmt.Printf("Error: %v\n", err)
			}
		}
	}()

	fmt.Println("Watching for file changes. Events will be printed below:")
	fmt.Println("------------------------------------------------------")

	// Wait for a signal or for the watcher to exit
	<-sigs

	fmt.Println("\nShutting down...")
}
