# Blink Configuration Guide

Blink supports configuration through multiple methods, with the following precedence (highest to lowest):

1. Command-line flags
2. Environment variables
3. Configuration file
4. Default values

## Configuration File

The default configuration file location is `$HOME/.blink.yaml`. You can specify a different file using the `--config` flag:

```bash
blink --config /path/to/config.yaml
```

## Configuration Options

### Basic Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `path` | string | `.` | Directory path to watch for changes |
| `allowed-origin` | string | `*` | Value for Access-Control-Allow-Origin header |
| `event-addr` | string | `:12345` | Address to serve events on ([host][:port]) |
| `event-path` | string | `/events` | URL path for the event stream |
| `refresh` | duration | `100ms` | Refresh duration for events |
| `verbose` | boolean | `false` | Enable verbose logging |
| `max-procs` | integer | `0` (all CPUs) | Maximum number of CPUs to use |

### Advanced Options

These options are available in the configuration file but not as command-line flags:

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `ignore-patterns` | string[] | `[]` | Additional file patterns to ignore |
| `shutdown-timeout` | duration | `5s` | Custom timeout for event server shutdown |
| `debug` | boolean | `false` | Enable debug mode for more detailed logging |

## Environment Variables

All configuration options can be set using environment variables with the prefix `BLINK_` followed by the option name in uppercase with dashes replaced by underscores:

```bash
# Examples
export BLINK_PATH="/path/to/watch"
export BLINK_ALLOWED_ORIGIN="https://example.com"
export BLINK_EVENT_ADDR=":8080"
export BLINK_VERBOSE="true"
```

## Configuration Management

You can manage configuration using the `config` subcommand:

```bash
# List all configuration values
blink config list

# Get a specific configuration value
blink config get path

# Set a configuration value
blink config set path /path/to/watch
```

## Sample Configuration File

```yaml
# Blink Configuration File
# This file can be placed at $HOME/.blink.yaml or specified with --config flag

# Directory path to watch for changes
path: "."

# Value for Access-Control-Allow-Origin header
allowed-origin: "*"

# Address to serve events on ([host][:port])
event-addr: ":12345"

# URL path for the event stream
event-path: "/events"

# Refresh duration for events (in milliseconds)
# This controls how frequently old events are cleaned up
refresh: 100ms

# Enable verbose logging
verbose: false

# Maximum number of CPUs to use
# Set to 0 to use all available CPUs
max-procs: 0

# Advanced configuration options

# List of file patterns to ignore (in addition to default patterns)
# ignore-patterns:
#   - "*.tmp"
#   - "node_modules"
#   - "*.log"

# Custom timeout for event server shutdown (in seconds)
# shutdown-timeout: 5s

# Enable debug mode for more detailed logging
# debug: false
```
