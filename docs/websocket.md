# WebSocket Support in Blink

Blink now supports WebSocket connections for real-time file system events, in addition to the existing Server-Sent Events (SSE) implementation. This document explains how to use WebSockets with Blink.

## Overview

WebSockets provide a bidirectional communication channel between the client and server, allowing for more interactive applications. Compared to SSE, WebSockets offer:

- Bidirectional communication
- Better support across browsers and platforms
- Lower overhead for high-frequency events
- Support for binary data

## Usage

### Command Line

To use WebSockets with Blink, use the `--stream-method` flag:

```bash
# Use WebSockets only
blink --path /path/to/watch --stream-method websocket

# Use both SSE and WebSockets
blink --path /path/to/watch --stream-method both

# Use SSE only (default)
blink --path /path/to/watch --stream-method sse
```

When using `--stream-method both`, Blink will serve:

- SSE events at the path specified by `--event-path` (default: `/events`)
- WebSocket events at the same path with `/ws` appended (default: `/events/ws`)

### Configuration

You can also configure the stream method in your configuration file:

```yaml
# ~/.blink.yaml
stream-method: websocket  # or "sse" or "both"
```

Or using environment variables:

```bash
export BLINK_STREAM_METHOD=websocket
```

## WebSocket Event Format

WebSocket events are sent as JSON objects with the following structure:

```json
{
  "type": "event",
  "timestamp": 1615123456789,
  "path": "/path/to/file.txt",
  "operation": "write"
}
```

The `operation` field can be one of:

- `create`: File or directory creation
- `write`: File modification
- `remove`: File or directory removal
- `rename`: File or directory renaming
- `chmod`: Permission changes

## Client Examples

### JavaScript

```javascript
const socket = new WebSocket('ws://localhost:12345/events/ws');

socket.onopen = () => {
  console.log('Connected to Blink WebSocket server');
};

socket.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log(`File ${data.operation}: ${data.path}`);
};

socket.onclose = () => {
  console.log('Disconnected from Blink WebSocket server');
};

socket.onerror = (error) => {
  console.error('WebSocket error:', error);
};
```

### HTML Example

A complete HTML example is available in the `examples/websocket-client` directory.

## Programmatic Usage

When using Blink as a library, you can specify the stream method using the `WithStreamMethod` option:

```go
import "github.com/TFMV/blink/pkg/blink"

// Use WebSockets
blink.EventServer(
    watchPath,
    "*",
    ":12345",
    "/events",
    100*time.Millisecond,
    blink.WithStreamMethod(blink.StreamMethodWebSocket),
)

// Use both SSE and WebSockets
blink.EventServer(
    watchPath,
    "*",
    ":12345",
    "/events",
    100*time.Millisecond,
    blink.WithStreamMethod(blink.StreamMethodBoth),
)
```

## Performance Considerations

WebSockets generally have better performance than SSE for high-frequency events, but they also require more resources on the server side. If you're monitoring a large number of files with frequent changes, WebSockets may be the better choice.

For simpler use cases or when compatibility with older browsers is important, SSE may be sufficient.

## Browser Support

WebSockets are supported in all modern browsers:

- Chrome 4+
- Firefox 4+
- Safari 5+
- Edge 12+
- Opera 10.70+
- iOS Safari 4.2+
- Android Browser 4.4+

## Security Considerations

WebSockets use the same-origin policy by default, but you can configure CORS using the `--allowed-origin` flag:

```bash
blink --allowed-origin "https://example.com" --stream-method websocket
```

This will only allow connections from the specified origin.
