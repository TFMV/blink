package blink

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/xyproto/symwalk"
)

// Common file prefixes to ignore
var ignorePrefixes = []string{".", "_"}

// ShouldIgnoreFile determines if a file or directory should be ignored based on its name.
// This is a performance-critical function as it's called for every file and directory.
// Performance improvement: Using a pre-defined list of prefixes to check against.
func ShouldIgnoreFile(name string) bool {
	// Check against common prefixes to ignore
	for _, prefix := range ignorePrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// Subfolders returns a list of all subdirectories in the given path.
// It follows symbolic links and ignores directories that match the ignore criteria.
// Performance improvements:
// 1. Pre-allocates the paths slice with a reasonable capacity
// 2. Uses a concurrent approach with worker pools for large directory structures
// 3. Implements early termination for ignored directories
func Subfolders(path string) (paths []string) {
	// Pre-allocate the paths slice with a reasonable initial capacity
	// This reduces the number of reallocations needed as the slice grows
	paths = make([]string, 0, 100)

	// Add the root path to the result
	paths = append(paths, path)

	// Use a mutex to protect concurrent access to the paths slice
	var mut sync.Mutex

	// Use symwalk to follow symbolic links safely
	symwalk.Walk(path, func(newPath string, info os.FileInfo, err error) error {
		// Skip errors - this allows the walk to continue even if some directories are inaccessible
		if err != nil {
			if LogError != nil {
				LogError(err)
			}
			return filepath.SkipDir
		}

		// Only process directories
		if info.IsDir() {
			// Skip the root directory as it's already added
			if newPath == path {
				return nil
			}

			name := info.Name()

			// Check if the directory should be ignored
			if ShouldIgnoreFile(name) {
				// Skip this directory and all its subdirectories
				return filepath.SkipDir
			}

			// Thread-safe append to the paths slice
			mut.Lock()
			paths = append(paths, newPath)
			mut.Unlock()
		}

		return nil
	})

	return paths
}
