# Claude Squad Frontend

This directory contains the React/TypeScript frontend for Claude Squad.

## Development

To start the development environment with hot-reloading:

```bash
# From the project root
./dev.sh
```

This will start both the Go backend server and the React development server in a tmux session.

## Building

To build the frontend and integrate it with the Go backend:

```bash
# From the project root
./build.sh
```

This will:
1. Build the React frontend with `npm run build`
2. Copy the build output to `web/static/dist/`
3. Build the Go application with the embedded frontend
4. Install the binary to `~/.local/bin/cs`

## Architecture

The frontend is built with:
- React for UI components
- TypeScript for type safety
- React Router for client-side routing
- xterm.js for terminal emulation

The backend serves the frontend as a Single Page Application (SPA) and provides:
- RESTful API for instance management
- WebSocket connections for real-time terminal updates
- Static file serving for the React application

## Development vs Production

In development mode:
- Frontend code is served by Vite's development server with hot reloading
- API requests are proxied to the Go backend
- Changes to React code appear immediately without rebuilding

In production mode:
- Frontend code is embedded in the Go binary
- All requests are handled by a single executable
- The application works offline with no external dependencies