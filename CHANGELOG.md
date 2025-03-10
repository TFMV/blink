# Changelog

All notable changes to the Blink project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2025-03-10

### Added

- WebSocket support for real-time event streaming
- New `--stream-method` flag to select between SSE, WebSocket, or both
- WebSocket client example in `examples/websocket-client`
- SSE client example in `examples/sse-client`
- Comprehensive documentation for WebSocket usage
- New `EventStreamer` interface for unified event delivery

### Changed

- Refactored event delivery system to support multiple streaming methods
- Improved error handling in event streaming
- Enhanced WebSocket connection management with automatic reconnection
- Optimized event serialization for WebSocket transport

## [0.2.0] - 2025-03-10

### Added

- Comprehensive benchmarking suite with detailed performance metrics
- New `BENCHMARKS.md` file documenting performance characteristics
- Enhanced file descriptor management for high-volume operations
- Improved resource cleanup in watcher Close() method

### Changed

- Optimized watcher implementation, resulting in:
  - 22% faster execution time
  - 5.5% less memory usage
  - 16.6% fewer allocations
- Improved batch processing for high-volume file system events
- Enhanced error handling in file system operations

### Fixed

- Fixed potential file descriptor leaks in high-load scenarios
- Resolved race conditions in watcher cleanup
- Improved handling of watcher close operations

## [0.1.0] - 2025-03-08

### Added

- Initial release of Blink
- Recursive directory watching with symbolic link support
- Server-sent events (SSE) for real-time notifications
- Cross-origin resource sharing (CORS) support
- High-performance design with worker pools and event debouncing
- Command-line interface with configuration management
- Configuration via YAML files, environment variables, and CLI flags
- Comprehensive documentation
- Example applications (CLI and web)
- Test suite with unit and integration tests
- Benchmarks for performance-critical functions

### Known Issues

- File deletion events may be reported as RENAME events on some platforms
- Large directory structures may require additional resources
