[![Build](https://github.com/TFMV/blink/actions/workflows/ci.yml/badge.svg)](https://github.com/TFMV/blink/actions/workflows/ci.yml)
[![Go 1.24](https://img.shields.io/badge/Go-1.24-blue)](https://golang.org/doc/go1.24)
[![Go Report Card](https://goreportcard.com/badge/github.com/TFMV/blink)](https://goreportcard.com/report/github.com/TFMV/blink)
[![Docs Website](https://img.shields.io/badge/docs-website-brightgreen)](https://tfmv.github.io/blink/)
[![Go Reference](https://pkg.go.dev/badge/github.com/TFMV/blink.svg)](https://pkg.go.dev/github.com/TFMV/blink)
[![Release](https://img.shields.io/github/v/release/TFMV/blink)](https://github.com/TFMV/blink/releases)
[![License](https://img.shields.io/github/license/TFMV/blink)](https://github.com/TFMV/blink/blob/main/LICENSE)

# Blink

Blink is a high-performance file system watcher that monitors directories for changes and provides events through a server-sent events (SSE) stream.

## Features

### Core Capabilities

- 🔍 **File System Monitoring**
  - Recursive directory watching
  - Symbolic link support
  - Real-time change detection

- 📡 **Event Delivery**
  - Server-sent events (SSE) for real-time notifications
  - Configurable refresh duration
  - Cross-origin resource sharing (CORS) support

### Performance Optimizations

- ⚡ **High-Performance Design**
  - Parallel directory scanning with worker pools
  - Non-blocking channel operations
  - Efficient memory usage with periodic cleanup
  - Event debouncing to reduce duplicate events
  - **Watcher**: Advanced file system monitoring with batched events
  - **Smart Event Handling**: Separate processing for file and directory events
  - **Configurable Batching**: Adjustable delay for grouping related events

### Integration & Configuration

- 🔧 **Flexible Configuration**
  - Command-line interface
  - Environment variables
  - YAML configuration files
  - Dynamic configuration management
  - **Polling Support**: Periodic scanning for new files
  - **Customizable Delays**: Fine-tune event batching for your workflow

- 🔌 **Integration Options**
  - Webhook support with retry logic
  - Custom HTTP headers
  - Configurable timeouts and debouncing
  - Filterable events and patterns
  - **Improved Pattern Matching**: Better file filtering with glob patterns

### Monitoring & Debugging

- 📊 **Observability**
  - Prometheus metrics integration
  - Health check endpoints
  - Verbose logging option
  - Kubernetes-ready deployment

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

### Kubernetes Deployment

Please see the [Kubernetes deployment guide](kubernetes/README.md) for detailed instructions on deploying Blink to Kubernetes.

## Benchmarks

Blink is optimized for high-performance. Here are some benchmark results from our test suite. For detailed performance metrics, memory analysis, and race condition testing, see our [comprehensive benchmarks report](BENCHMARKS.md).

### Filter Performance

| Scenario | Operations/sec | Time/op | Memory/op | Allocations/op |
|----------|---------------|---------|-----------|----------------|
| No Filters | 337,099,838 | 3.51 ns/op | 0 B/op | 0 allocs/op |
| Include Patterns | 18,537,398 | 65.64 ns/op | 0 B/op | 0 allocs/op |
| Exclude Patterns | 5,711,019 | 212.4 ns/op | 0 B/op | 0 allocs/op |
| Include Events | 5,038,678 | 242.2 ns/op | 0 B/op | 0 allocs/op |
| Ignore Events | 4,297,494 | 273.1 ns/op | 0 B/op | 0 allocs/op |
| All Filters | 4,412,713 | 276.1 ns/op | 0 B/op | 0 allocs/op |

## Usage

```bash
blink -path /path/to/watch -event-addr :12345 -event-path /events
```

### Command-line Options

| Option | Description | Default |
|--------|-------------|---------|
| `-path` | Directory path to watch for changes (must be a valid directory) | `"."` |
| `-allowed-origin` | Value for Access-Control-Allow-Origin header | `"*"` |
| `-event-addr` | Address to serve events on ([host][:port]) | `":12345"` |
| `-event-path` | URL path for the event stream | `"/events"` |
| `-refresh` | Refresh duration for events | `100ms` |
| `-verbose` | Enable verbose logging | `false` |
| `-max-procs` | Maximum number of CPUs to use | all available |
| `-include` | Include patterns for files (e.g., "*.js,*.css,*.html") | none |
| `-exclude` | Exclude patterns for files (e.g., "node_modules,*.tmp") | none |
| `-events` | Include event types (e.g., "write,create") | none |
| `-ignore` | Ignore event types (e.g., "chmod") | none |
| `-webhook-url` | URL for the webhook | none |
| `-webhook-method` | HTTP method for the webhook | `"POST"` |
| `-webhook-headers` | Headers for the webhook (format: "key1:value1,key2:value2") | none |
| `-webhook-timeout` | Timeout for the webhook | `5s` |
| `-webhook-debounce-duration` | Debounce duration for the webhook | `0s` |
| `-webhook-max-retries` | Maximum number of retries for the webhook | `3` |
| `-help` | Show help | n/a |

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
