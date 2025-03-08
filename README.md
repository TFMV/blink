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
- `-event-addr`: Address to serve events on ([host][:port]) (default: ":12345")
- `-event-path`: URL path for the event stream (default: "/events")
- `-refresh`: Refresh duration for events (default: 100ms)
- `-verbose`: Enable verbose logging (default: false)
- `-max-procs`: Maximum number of CPUs to use (default: all available)
- `-help`: Show help

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
import "github.com/TFMV/blink/pkg/blink"

func main() {
    // Set verbose mode
    blink.SetVerbose(true)
    
    // Start the event server
    blink.EventServer(
        ".",                  // Directory to watch
        "*",                  // Allow all origins
        ":12345",             // Listen on port 12345
        "/events",            // Event path
        100*time.Millisecond, // Refresh duration
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
