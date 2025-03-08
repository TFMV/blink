package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/TFMV/blink/cmd/blink/cmd"
)

func main() {
	// Set up signal handling for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Run the command in a goroutine
	go func() {
		cmd.Execute()
	}()

	// Wait for a signal
	<-sigs

	// Perform any cleanup here if needed
	os.Exit(0)
}
