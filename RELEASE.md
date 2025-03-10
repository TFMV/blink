# Blink v0.3.0 Release Notes

## Overview

Blink continues to evolve as a high-performance file system watcher, offering enhanced features and optimizations in this release. Version 0.3.0 introduces significant improvements in event filtering, webhook management, and CLI usability, making it even more versatile and efficient for real-time file monitoring.

## New Features

- **Enhanced Event Filtering**: More precise control over event processing with improved filtering capabilities.
- **Webhook Enhancements**: Support for custom headers and improved retry logic for webhooks.
- **CLI Improvements**: New subcommands for easier configuration management and enhanced command-line options.

## Improvements

- **Performance Optimizations**:
  - Enhanced parallel directory scanning for faster event processing.
  - Improved memory management with periodic cleanup to reduce memory usage.
- **Documentation Updates**: Comprehensive updates to the documentation, including new examples and detailed configuration guides.

## Bug Fixes

- **Fixed SSE Stream Issues**: Resolved issues with SSE stream stability and improved event delivery reliability.
- **Resolved CLI Bugs**: Fixed various bugs related to command-line argument parsing and configuration file handling.

## Known Issues

- **Limited Support for Large Directories**: Performance may degrade when monitoring very large directory structures. Further optimizations are planned for future releases.

## Upgrade Notes

- **Configuration Changes**: Ensure to review the updated configuration options and adjust your settings accordingly.
- **Dependency Updates**: Updated dependencies to the latest versions for improved security and performance.

## Installation

```bash
go install github.com/TFMV/blink/cmd/blink@latest
```

## Usage

### Command Line

The CLI provides enhanced usability with new subcommands and options:

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

## Examples

The release includes updated examples:

- A simple command-line application that prints file events to the console
- A web application that displays file events in real-time using JavaScript

## Future Plans

- Further improvements in event filtering and handling of large directory structures
- Additional event types and metadata
- Performance optimizations for specific file systems
- More examples for different programming languages and frameworks

## Contributors

- TFMV

## License

See the [LICENSE](LICENSE) file for details.
