# Claude Squad Web Monitoring

This package provides a web server for monitoring Claude Squad instances remotely. It allows you to:

- View a list of all Claude instances
- Monitor terminal output in real-time
- View git diff information
- Get instance metadata and status

## Features

- **RESTful API**: Access instance data programmatically
- **WebSocket Support**: Real-time terminal output streaming
- **Git Diff Visualization**: View changes made by Claude
- **Web UI**: A simple interface for monitoring instances
- **Security**: Authentication and rate limiting for remote access
- **Cross-Platform**: Works on both Unix and Windows systems

## Usage

To enable the web monitoring server, use the `--web` flag:

```bash
cs --web
```

By default, the server listens on `127.0.0.1:8080`. You can change the port with the `--web-port` flag:

```bash
cs --web --web-port=9000
```

## Configuration

Configuration is stored in `~/.claude-squad/config.json`. The following settings control the web server:

```json
{
  "web_server_enabled": true,
  "web_server_port": 8080,
  "web_server_host": "127.0.0.1",
  "web_server_auth_token": "your-auth-token",
  "web_server_allow_localhost": true,
  "web_server_use_tls": false,
  "web_server_tls_cert": "",
  "web_server_tls_key": "",
  "web_server_cors_origin": "http://localhost:3000"
}
```

## API Endpoints

### Instance Management

- `GET /api/instances`: List all instances
- `GET /api/instances/{name}`: Get instance details
- `GET /api/instances/{name}/output`: Get terminal output
- `GET /api/instances/{name}/diff`: Get git diff information

### Terminal Streaming

- `WebSocket /ws/terminal/{name}`: Stream terminal output in real-time

### System Information

- `GET /api/status`: Get server status information

## Security

- **Authentication**: Bearer token authentication for remote access
- **CORS**: Configurable CORS policy for web clients
- **Rate Limiting**: Protection against excessive requests
- **TLS**: Optional TLS encryption for secure communication

By default, localhost connections are allowed without authentication. For remote access, you'll need to provide an authentication token in the `Authorization` header:

```
Authorization: Bearer your-auth-token
```

## Web UI

The web server includes a simple web UI for monitoring instances. Access it by visiting:

```
http://localhost:8080/
```

The UI provides:

- List of all Claude instances
- Real-time terminal output display
- Git diff visualization
- Instance details and status information