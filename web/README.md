# Claude Squad Web Monitoring

This package provides a web server for monitoring Claude Squad instances remotely. It allows you to:

- View a list of all Claude instances
- Monitor terminal output in real-time
- View git diff information
- Get instance metadata and status
- Interact with terminals through bidirectional communication
- View and track task progress
- Analyze instance performance metrics

## Features

- **RESTful API**: Access instance data programmatically
- **WebSocket Support**: Real-time bidirectional terminal communication
- **Git Diff Visualization**: View changes made by Claude with syntax highlighting
- **Task Tracking**: Monitor Claude's task progress in structured format
- **Advanced Web UI**: Rich interface with customizable dashboard
- **Security**: Authentication, permissions, and rate limiting for remote access
- **Cross-Platform**: Works on both Unix and Windows systems

## Usage

To enable the web monitoring server, use the `--web` flag:

```bash
cs --web
```

By default, the server listens on `127.0.0.1:8099`. You can change the port with the `--web-port` flag:

```bash
cs --web --web-port=9000
```

## Configuration

Configuration is stored in `~/.claude-squad/config.json`. The following settings control the web server:

```json
{
  "web_server_enabled": true,
  "web_server_port": 8099,
  "web_server_host": "127.0.0.1",
  "web_server_auth_token": "your-auth-token",
  "web_server_allow_localhost": true,
  "web_server_use_tls": false,
  "web_server_tls_cert": "",
  "web_server_tls_key": "",
  "web_server_cors_origin": "*"
}
```

## API Endpoints

### Instance Management

- `GET /api/instances`: List all instances
- `GET /api/instances/{name}`: Get instance details
- `GET /api/instances/{name}/output`: Get terminal output
- `GET /api/instances/{name}/diff`: Get git diff information
- `GET /api/instances/{name}/tasks`: Get structured task information

### Terminal Streaming

- `WebSocket /ws/terminal/{name}`: Bidirectional terminal communication
  - Query parameters:
    - `format`: Output format (ansi, html, text)
    - `privileges`: Access level (read-only, read-write)

### System Information

- `GET /api/status`: Get server status information
- `GET /api/metrics`: Get system performance metrics

## Security

- **Authentication**: Bearer token authentication for remote access
- **Privileges**: Read-only vs. read-write access control
- **CORS**: Configurable CORS policy for web clients
- **Rate Limiting**: Protection against excessive requests
- **TLS**: Optional TLS encryption for secure communication

By default, localhost connections are allowed without authentication. For remote access, you'll need to provide an authentication token in the `Authorization` header:

```
Authorization: Bearer your-auth-token
```

## Web UI

The web server includes an advanced web UI for monitoring and interacting with instances. Access it by visiting:

```
http://localhost:8099/
```

The UI provides:

- List of all Claude instances with status indicators
- Real-time terminal output display with bidirectional input
- Enhanced terminal experience with themes and configuration options
- Git diff visualization with syntax highlighting
- Task tracking with progress indicators
- Instance details and performance metrics
- Customizable dashboard with drag-and-drop widgets

## Implementation Status

For detailed information about the implementation status, current tasks, and design specifications, see the [Implementation Status](./IMPLEMENTATION_STATUS.md) document.

## Testing

Test the web server functionality with the included test scripts:

### Basic API and UI Tests

```bash
./web/test-direct.sh
```

This script starts Claude Squad in Simple Mode with web monitoring enabled, then tests the API endpoints and web UI.

### WebSocket Terminal Tests

To test the WebSocket terminal streaming functionality:

```bash
./web/test-websocket.sh
```

This runs a Go test suite that validates the bidirectional terminal communication via WebSockets.

### End-to-End Terminal Tests 

For a more comprehensive E2E test of the terminal WebSocket functionality:

```bash
./web/test-e2e-websocket.sh
```

This script:
1. Starts Claude Squad with web monitoring
2. Connects to the terminal WebSocket
3. Sends input to the terminal
4. Verifies that terminal updates are received

### Terminal Visibility Test

To troubleshoot any issues with terminal display in the web UI:

1. Start Claude Squad with web monitoring
2. Open your browser to http://localhost:8099/
3. Open the browser console (F12)
4. Copy and paste the contents of `web/terminal-test.js` into the console
5. The script will diagnose terminal visibility issues and provide real-time monitoring

### Standalone Terminal Rendering Test

To test terminal rendering outside of the main application:

1. Open the `web/terminal-standalone-test.html` file in your browser
2. This standalone page allows testing various terminal rendering methods:
   - Plain content display
   - Content appending
   - Different DOM manipulation techniques
   - Style and formatting tests

This helps isolate DOM rendering issues from WebSocket communication problems.