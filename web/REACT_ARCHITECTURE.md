# Claude Squad React Frontend Architecture

This document outlines the architecture and implementation details of the React frontend for Claude Squad.

## Overview

The Claude Squad web interface has been restructured using React and TypeScript to provide a more modern, maintainable, and feature-rich user experience. The frontend is built with:

- React for UI components
- TypeScript for type safety
- React Router for client-side routing
- xterm.js for terminal emulation

## Directory Structure

```
frontend/
├── public/             # Static assets
├── src/
│   ├── api/            # API client and WebSocket code
│   ├── components/     # Reusable UI components
│   │   ├── common/     # Shared components
│   │   └── terminal/   # Terminal-specific components
│   ├── context/        # React context providers
│   ├── hooks/          # Custom React hooks
│   ├── pages/          # Page components
│   ├── types/          # TypeScript type definitions
│   ├── utils/          # Utility functions
│   ├── App.tsx         # Main application component
│   └── main.tsx        # Application entry point
└── package.json        # Dependencies and scripts
```

## Key Components

### Terminal Component

The Terminal component (`components/terminal/Terminal.tsx`) is the core of the application's functionality. It:

1. Establishes a WebSocket connection to the backend server
2. Renders terminal output using xterm.js
3. Handles user input and sends it to the backend
4. Manages connection state and reconnection logic
5. Processes both binary and JSON message formats for backward compatibility

```typescript
// Terminal component supports both protocols:
// 1. JSON protocol for newer clients
// 2. Binary protocol for backward compatibility

// Key features:
// - Automatic reconnection if connection is lost
// - Heartbeat mechanism to detect stale connections
// - Input handling with proper character encoding
// - ANSI escape sequence rendering via xterm.js
// - Terminal resize events
```

### Instances Page

The Instances Page (`pages/InstancesPage.tsx`) provides a listing of all available Claude instances with:

1. Real-time status indicators
2. Auto-refreshing instance data
3. Details about each instance (creation time, status, path)
4. Direct links to open terminal for any instance

## WebSocket Protocol

The WebSocket implementation supports two message formats:

### JSON Format (Newer Protocol)

```json
// Output message
{
  "InstanceTitle": "instance-name",
  "Content": "Terminal output content with ANSI codes",
  "Timestamp": "2023-05-07T08:35:51Z",
  "Status": "running",
  "HasPrompt": true
}

// Input message
{
  "content": "User input text",
  "isCommand": false
}

// Command message
{
  "content": "resize",
  "isCommand": true,
  "cols": 80,
  "rows": 24
}
```

### Binary Format (Legacy Protocol)

Binary messages use a prefix byte to indicate message type:

- 'o' (111) - Output message
- 'i' (105) - Input message
- 'r' (114) - Resize message
- 'p' (112) - Ping message
- 'P' (80) - Pong message
- 'c' (99) - Close message

## Server Integration

The React frontend integrates with the Go backend through:

1. RESTful API endpoints for instance data
2. WebSocket connections for real-time terminal I/O
3. Static file serving with automatic fallback

The server (`web/static/serve.go`) prioritizes serving the React frontend if available, with fallback to the legacy HTML interface:

```go
// Priority order for serving:
// 1. React SPA routes (for non-asset, non-API routes)
// 2. React static assets
// 3. Embedded legacy HTML files
```

## Build Process

The frontend is built and integrated with the backend using:

1. `build_frontend.sh` - Builds the React app with npm
2. `build.sh` - Builds the entire application, including embedding the frontend

## Future Improvements

- Add authentication to the web interface
- Implement instance creation and management from the web UI
- Add theme customization and preferences
- Implement Git operations in the web interface
- Add mobile-specific optimizations for touch interactions