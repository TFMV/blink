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

## Project Structure

This project follows the standard Go project layout:

- `cmd/blink/`: Contains the CLI application (package `main`)
- `pkg/blink/`: Contains the library code (package `blink`)
- `examples/`: Contains example applications

This separation of concerns allows:

- Using Blink as a library in other Go projects
- Installing the CLI tool independently
- Clear distinction between public API and implementation details

## Installation

```bash
go install github.com/TFMV/blink/cmd/blink@latest
```

> **Note:** The installation path includes `/cmd/blink` because that's where the main executable is located, following Go's standard project layout. This structure separates the reusable library code (in `pkg/blink`) from the command-line interface (in `cmd/blink`).

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

- `-path`: Directory path to watch for changes (default: ".")
- `-allowed-origin`: Value for Access-Control-Allow-Origin header (default: "*")
- `-event-addr`: Address to serve events on ([host][:port]) (default: ":12345")
- `-event-path`: URL path for the event stream (default: "/events")
- `-refresh`: Refresh duration for events (default: 100ms)
- `-verbose`: Enable verbose logging (default: false)
- `-max-procs`: Maximum number of CPUs to use (default: all available)
- `-help`: Show help

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

## Performance Considerations

Blink is designed to be efficient even when watching large directory structures:

- Uses worker pools for parallel directory scanning
- Implements event debouncing to reduce duplicate events
- Uses non-blocking channel operations to prevent goroutine leaks
- Periodically cleans up old events to prevent memory leaks
- Provides configurable CPU usage control

## Examples

Check the `examples/` directory for usage examples:

- `examples/simple/`: A simple Go program using the Blink library
- `examples/web/`: A web application that connects to the Blink event stream

## License

See the [LICENSE](LICENSE) file for details.
