# Changelog

All notable changes to the Blink project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
