# Blink v0.1.0 Release Notes

## Overview

Blink is a high-performance file system watcher that monitors directories for changes and provides events through a server-sent events (SSE) stream. This initial release provides a solid foundation for real-time file monitoring that can be integrated with any system or language that supports SSE. While implemented in Go, Blink is designed as a language-agnostic tool that can be used with JavaScript, Python, Ruby, PHP, or any other environment capable of consuming SSE streams.

## Features

- **Recursive Directory Watching**: Monitor entire directory trees for changes
- **Symbolic Link Support**: Properly follows symbolic links for comprehensive monitoring
- **Server-Sent Events (SSE)**: Real-time notifications via standard SSE protocol
- **Cross-Origin Resource Sharing**: Configurable CORS support for web applications
- **High-Performance Design**:
  - Parallel directory scanning with worker pools
  - Event debouncing to reduce duplicate events
  - Non-blocking channel operations
  - Efficient memory usage with periodic cleanup
- **Robust CLI**: Command-line interface with configuration management (provided as a convenience)
- **Configuration Options**: Support for YAML config files, environment variables, and CLI flags
- **Language Agnostic**: Can be used with any programming language or framework that supports SSE

## Installation

```bash
go install github.com/TFMV/blink/cmd/blink@latest
```

## Usage

### Command Line

The CLI is provided as a convenience, but the core functionality is accessible from any language via SSE:

```bash
# Watch the current directory
blink

# Watch a specific directory
blink --path /path/to/watch

# Customize the server address and port
blink --event-addr localhost:12345

# Enable verbose logging
blink --verbose
```

### As a Go Library

```go
import "github.com/TFMV/blink/pkg/blink"

func main() {
    // Start the event server
    blink.EventServer(
        ".",                  // Directory to watch
        "*",                  // Allow all origins
        "localhost:12345",    // Listen on localhost:12345
        "/events",            // Event path
        100*time.Millisecond, // Refresh duration
    )
    
    // Wait for events
    select {}
}
```

### From Other Languages

Blink can be used from any language that supports SSE. Here are some examples:

#### JavaScript

```javascript
const eventSource = new EventSource('http://localhost:12345/events');
eventSource.onmessage = function(event) {
  console.log('File changed:', event.data);
};
```

#### Python

```python
import sseclient
import requests

url = 'http://localhost:12345/events'
headers = {'Accept': 'text/event-stream'}
response = requests.get(url, headers=headers, stream=True)
client = sseclient.SSEClient(response)
for event in client.events():
    print(f"File changed: {event.data}")
```

#### Ruby

```ruby
require 'em-eventsource'

EM.run do
  source = EventMachine::EventSource.new("http://localhost:12345/events")
  source.message do |message|
    puts "File changed: #{message}"
  end
  source.start
end
```

## Known Issues

- On some platforms, file deletion events may be reported as RENAME events due to limitations in the underlying file system notification APIs
- Very large directory structures may require additional memory and CPU resources

## Examples

The release includes two examples:

- A simple command-line application that prints file events to the console
- A web application that displays file events in real-time using JavaScript

## Future Plans

- Improved filtering options for events
- Better handling of large directory structures
- Additional event types and metadata
- Performance optimizations for specific file systems
- More examples for different programming languages and frameworks

## Contributors

- TFMV Team

## License

See the [LICENSE](LICENSE) file for details.
