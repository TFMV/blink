package blink

import (
	"errors"
	"os"
	"runtime"
	"sync"

	"github.com/TFMV/blink/pkg/logger"
	"github.com/fsnotify/fsnotify"
)

// RecursiveWatcher keeps the data for watching files and directories.
// It embeds fsnotify.Watcher and adds channels for tracking files and folders.
type RecursiveWatcher struct {
	*fsnotify.Watcher
	Files   chan string // Channel for file events
	Folders chan string // Channel for folder events
	// Adding a mutex for thread-safe operations
	mu sync.Mutex
}

// NewRecursiveWatcher creates a new RecursiveWatcher.
// Takes a path to a directory to watch recursively.
// Performance improvements:
// 1. Uses a worker pool for adding folders in parallel
// 2. Pre-allocates the paths slice with a reasonable capacity
// 3. Uses a mutex for thread-safe operations
func NewRecursiveWatcher(path string) (*RecursiveWatcher, error) {
	// Get all subfolders to watch
	folders := Subfolders(path)
	if len(folders) == 0 {
		return nil, errors.New("no directories to watch")
	}

	// Create a new fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	// Create a new RecursiveWatcher
	rw := &RecursiveWatcher{Watcher: watcher}

	// Buffer the channels to reduce blocking
	// Files channel is buffered to handle bursts of file events
	rw.Files = make(chan string, 100) // Increased buffer size for better performance during high activity

	// Folders channel is sized based on the number of folders to watch
	rw.Folders = make(chan string, len(folders))

	// Use a worker pool to add folders in parallel for better performance
	// This is especially beneficial when watching large directory structures
	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8 // Cap at 8 workers to avoid excessive goroutines
	}

	// Create a wait group to wait for all workers to finish
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	// Create a channel to distribute folders to workers
	folderChan := make(chan string, len(folders))

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for folder := range folderChan {
				if err := rw.AddFolder(folder); err != nil {
					// Log the error but continue with other folders
					if LogError != nil {
						LogError(err)
					}
				}
			}
		}()
	}

	// Send folders to workers
	for _, folder := range folders {
		folderChan <- folder
	}
	close(folderChan)

	// Wait for all workers to finish
	wg.Wait()

	return rw, nil
}

// AddFolder adds a directory to watch, non-recursively.
// It's thread-safe due to the mutex lock.
func (watcher *RecursiveWatcher) AddFolder(folder string) error {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()

	if err := watcher.Add(folder); err != nil {
		return err
	}

	// Only send to channel if there's space to avoid blocking
	select {
	case watcher.Folders <- folder:
		// Successfully sent
	default:
		// Channel is full, log but don't block
		if LogError != nil {
			LogError(errors.New("folders channel is full, skipping notification for: " + folder))
		}
	}

	return nil
}

// Close properly closes the watcher and its channels.
// This method should be called when the watcher is no longer needed
// to prevent resource leaks.
func (watcher *RecursiveWatcher) Close() error {
	watcher.mu.Lock()
	defer watcher.mu.Unlock()

	// Close the embedded watcher
	err := watcher.Watcher.Close()

	// Close the channels to signal any goroutines that might be reading from them
	close(watcher.Files)
	close(watcher.Folders)

	return err
}

// addFolderToWatch adds a folder to the watcher
func (w *Watcher) addFolderToWatch(folder string) error {
	// Check if the folder exists
	info, err := os.Stat(folder)
	if err != nil {
		return err
	}

	// Make sure it's a directory
	if !info.IsDir() {
		return nil
	}

	// Add the folder to the watcher
	if err := w.watcher.Add(folder); err != nil {
		logger.Error(err)
		return err
	}

	return nil
}
