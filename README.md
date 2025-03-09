[![Build](https://github.com/TFMV/blink/actions/workflows/ci.yml/badge.svg)](https://github.com/TFMV/blink/actions/workflows/ci.yml)
[![Go 1.24](https://img.shields.io/badge/Go-1.24-blue)](https://golang.org/doc/go1.24)
[![Go Report Card](https://goreportcard.com/badge/github.com/TFMV/blink)](https://goreportcard.com/report/github.com/TFMV/blink)
[![Docs Website](https://img.shields.io/badge/docs-website-brightgreen)](https://github.com/TFMV/blink)
[![Go Reference](https://pkg.go.dev/badge/github.com/TFMV/blink.svg)](https://pkg.go.dev/github.com/TFMV/blink)
[![Release](https://img.shields.io/github/v/release/TFMV/blink)](https://github.com/TFMV/blink/releases)
[![License](https://img.shields.io/github/license/TFMV/blink)](https://github.com/TFMV/blink/blob/main/LICENSE)

# Blink

Blink is a high-performance file system watcher that monitors directories for changes and provides events through a server-sent events (SSE) stream.

## Features

- Recursive directory watching with symbolic link support
- Server-sent events (SSE) for real-time notifications
- Configurable refresh duration
- Cross-origin resource sharing (CORS) support
- Verbose logging option
- High-performance design with:
  - Parallel directory scanning
  - Event debouncing to reduce duplicate events
  - Worker pools for handling large directory structures
  - Non-blocking channel operations
  - Efficient memory usage with periodic cleanup
- Robust CLI with configuration management

## Installation

```bash
go install github.com/TFMV/blink/cmd/blink@latest
```

Or clone the repository and build manually:

```bash
git clone https://github.com/TFMV/blink.git
cd blink
go build -o blink ./cmd/blink
```

## Usage

```bash
blink -path /path/to/watch -event-addr :12345 -event-path /events
```

### Command-line Options

- `-path`: Directory path to watch for changes (must be a valid directory) (default: ".")
- `-allowed-origin`: Value for Access-Control-Allow-Origin header (default: "*")
- `-event-addr`: Address to serve events on (\[host\][:port]) (default: ":12345")
- `-event-path`: URL path for the event stream (default: "/events")
- `-refresh`: Refresh duration for events (default: 100ms)
- `-verbose`: Enable verbose logging (default: false)
- `-max-procs`: Maximum number of CPUs to use (default: all available)
- `-include`: Include patterns for files (e.g., "*.js,*.css,*.html")
- `-exclude`: Exclude patterns for files (e.g., "node_modules,*.tmp")
- `-events`: Include event types (e.g., "write,create")
- `-ignore`: Ignore event types (e.g., "chmod")
- `-webhook-url`: URL for the webhook
- `-webhook-method`: HTTP method for the webhook (default: "POST")
- `-webhook-headers`: Headers for the webhook (format: "key1:value1,key2:value2")
- `-webhook-timeout`: Timeout for the webhook (default: 5s)
- `-webhook-debounce-duration`: Debounce duration for the webhook (default: 0s)
- `-webhook-max-retries`: Maximum number of retries for the webhook (default: 3)
- `-help`: Show help

### Event Filtering

Blink supports filtering capabilities to focus on specific files or event types:

```bash
# Only watch for changes to JavaScript, CSS, and HTML files
blink --include "*.js,*.css,*.html"

# Ignore node_modules directory and temporary files
blink --exclude "node_modules,*.tmp"

# Only trigger on write and create events
blink --events "write,create"

# Ignore chmod events
blink --ignore "chmod"

# Combine multiple filters
blink --include "*.js,*.css" --exclude "node_modules" --events "write,create"
```

Available event types:

- `create`: File or directory creation
- `write`: File modification
- `remove`: File or directory removal
- `rename`: File or directory renaming
- `chmod`: Permission changes

### Webhooks

Blink can send webhooks when file changes occur, allowing integration with other systems:

```bash
# Send webhooks to a URL
blink --webhook-url "https://example.com/webhook"

# Use a specific HTTP method
blink --webhook-method "POST"

# Add custom headers
blink --webhook-headers "Authorization:Bearer token,Content-Type:application/json"

# Set timeout and retry options
blink --webhook-timeout 10s --webhook-max-retries 5

# Debounce webhooks to reduce the number of requests
blink --webhook-debounce-duration 500ms

# Combine with filters to only send webhooks for specific events
blink --include "*.js" --events "write" --webhook-url "https://example.com/webhook"
```

Webhook payload format:

```json
{
  "path": "/path/to/changed/file.js",
  "event_type": "write",
  "time": "2023-03-08T12:34:56.789Z"
}
```

### Configuration Management

Blink supports configuration through:

1. Command-line flags
2. Environment variables
3. Configuration file (YAML)

You can manage configuration using the `config` subcommand:

```bash
# List all configuration values
blink config list

# Get a specific configuration value
blink config get path

# Set a configuration value
blink config set path /path/to/watch
```

Configuration is stored in `$HOME/.blink.yaml` by default, but you can specify a different file with the `--config` flag.

## Example

Start watching the current directory:

```bash
blink
```

Connect to the event stream:

```javascript
// In your web application
const eventSource = new EventSource('http://localhost:12345/events');
eventSource.onmessage = function(event) {
  console.log('File changed:', event.data);
};
```

## Using as a Library

You can also use Blink as a library in your Go projects:

```go
import (
    "time"
    "github.com/TFMV/blink/pkg/blink"
)

func main() {
    // Set verbose mode
    blink.SetVerbose(true)
    
    // Create a filter
    filter := blink.NewEventFilter()
    filter.SetIncludePatterns("*.js,*.css,*.html")
    filter.SetExcludePatterns("node_modules,*.tmp")
    filter.SetIncludeEvents("write,create")
    filter.SetIgnoreEvents("chmod")
    
    // Start the event server with filters and webhooks
    blink.EventServer(
        ".",                  // Directory to watch
        "*",                  // Allow all origins
        ":12345",             // Listen on port 12345
        "/events",            // Event path
        100*time.Millisecond, // Refresh duration
        // Options
        blink.WithFilter(filter),
        blink.WithWebhook("https://example.com/webhook", "POST"),
        blink.WithWebhookHeaders(map[string]string{
            "Authorization": "Bearer token",
            "Content-Type": "application/json",
        }),
        blink.WithWebhookTimeout(10*time.Second),
        blink.WithWebhookDebounce(500*time.Millisecond),
        blink.WithWebhookRetries(5),
    )
    
    // Wait for events
    select {}
}
```

## Testing and Benchmarking

Blink includes a comprehensive test suite and benchmarks to ensure reliability and performance.

### Running Tests

```bash
# Run all tests
make test

# Run tests without integration tests
make test-short
```

The test suite includes:

- Unit tests for core functionality
- Integration tests for the event server
- Tests for file system operations

Current test coverage: ~67% of statements in the core package.

### Running Benchmarks

```bash
# Run benchmarks
make benchmark
```

Benchmark results show excellent performance:

- `ShouldIgnoreFile`: ~30.57 ns/op, 0 B/op, 0 allocs/op
- `RemoveOldEvents`: ~121779 ns/op for 1000 events
- `Subfolders`: Fast directory scanning with optimized memory usage

### Generating Coverage Reports

```bash
# Generate test coverage report
make coverage
```

This will create a coverage report and open it in your browser.

## Examples

Check the `examples/` directory for usage examples:

- `examples/simple/`: A simple Go program using the Blink library
- `examples/web/`: A web application that connects to the Blink event stream

## License

See the [LICENSE](LICENSE) file for details.
