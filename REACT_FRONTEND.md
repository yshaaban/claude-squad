# Claude Squad React Frontend

This document outlines the React/TypeScript frontend integration for Claude Squad.

## Implementation Status

✅ **COMPLETED**: The React frontend is fully implemented and integrated into Claude Squad.

Key features:
- Interactive terminal with WebSocket communication
- Instance listing and management
- Responsive design for desktop and mobile
- Proper asset handling and routing
- Binary and JSON protocol support
- Integrated build system

Access it with: `cs -s --web --react`

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
│   ├── context/        # React context providers
│   ├── hooks/          # Custom React hooks
│   ├── pages/          # Page components
│   ├── types/          # TypeScript type definitions
│   ├── utils/          # Utility functions
│   ├── App.tsx         # Main application component
│   └── main.tsx        # Application entry point
└── package.json        # Dependencies and scripts
```

## Development

To start the development environment with hot-reloading:

```bash
# From the project root
./dev.sh
```

This launches a tmux session with:
1. The Go backend on port 8085
2. The React development server with HMR
3. A utility terminal for commands

## Building

To build and install the complete application:

```bash
# From the project root
./build.sh
```

This will:
1. Build the React frontend with optimizations
2. Copy the build output to web/static/dist/
3. Build the Go application with the embedded frontend
4. Install the binary to ~/.local/bin/cs

## Running the Application

After installation, run the application with:

```bash
# For Simple Mode with the web interface (legacy HTML)
cs -s --web

# For Simple Mode with the React web interface (recommended)
cs -s --web --react

# For web interface with a specific port
cs -s --web --web-port 8090 --react
```

## New Integrated View

The React frontend now features an integrated view that combines the instance list and terminal in a single page for improved usability:

- **Side-by-Side Layout**: Instance list on the left, terminal on the right
- **Click-to-Connect**: Click any instance to connect to its terminal without page navigation
- **Live Status**: Real-time connection status and instance information
- **Improved Performance**: Better state management to prevent infinite loops and excessive API requests

This integrated view is now the default when you access the root URL of the web interface.

## Testing the Web Server

For quick testing of the React frontend without running the full application:

```bash
# Test server with legacy HTML interface
./test_web.sh

# Test server with React frontend
./test_web.sh --react

# Test server with React frontend and custom port
./test_web.sh --react --port 9000

# Build frontend first, then test
./test_web.sh --react --build-frontend

# React frontend with full terminal functionality
./test_react_frontend.sh

# Enhanced diagnostics for the React frontend
./test_robust_react.sh
```

These test scripts provide different options for testing:
- `test_web.sh` provides a lightweight test server for UI validation
- `test_react_frontend.sh` provides a full-featured test environment with terminal functionality
- `test_robust_react.sh` includes diagnostic tools and tests for asset loading and WebSocket connections
- Use the `--build-frontend` flag when you've made changes to the React code

## Architecture

The application architecture follows these principles:

1. **Single Executable**: The Go application embeds the React frontend build, resulting in a single executable with no external dependencies.

2. **Separation of Concerns**:
   - Go backend handles terminal management, git operations, and WebSocket communication
   - React frontend handles UI rendering, routing, and user interaction

3. **API-First Design**:
   - REST API for instance management and static data
   - WebSocket for real-time terminal updates

4. **Development Experience**:
   - Hot Module Replacement during development
   - TypeScript for better type safety and developer tooling
   - Modern React patterns with hooks and context

## Implementation Notes

- The web server implementation uses Chi router for API endpoints
- Static file serving is handled by Go's embed package in production
- In development mode, the React app is served by Vite and proxied to the Go backend
- WebSocket connections are maintained for real-time terminal updates
- The application is responsive and works on mobile devices

## Recent Fixes and Improvements

### Key Issues Fixed

1. **Rate Limiting Problems**
   - Created separate rate limits for API endpoints (1000 requests per minute)
   - Enhanced WebSocket detection to prevent rate limiting for terminal connections
   - Added path-based detection to exempt WebSocket and terminal paths
   - Fixed infinite polling loop in React frontend with proper React patterns:
     - Using useRef for mutable state that shouldn't trigger re-renders
     - Fixed useEffect dependency arrays to prevent re-creation of intervals
     - Implemented proper cleanup of intervals and event listeners
     - Added throttling to prevent excessive API requests
   - Added proper error handling with exponential backoff for API requests

2. **Asset Path Resolution**
   - Fixed path handling in the React build to use `./assets/` instead of absolute paths
   - Enhanced the file server to try multiple path variations for assets
   - Added better logging for asset resolution

3. **SPA Route Handling**
   - Improved the file server to better handle React's Single Page Application routing
   - Enhanced error handling and logging for route resolution

### Path Resolution Strategy

The server now attempts to resolve assets in multiple ways:

1. First tries the exact path requested
2. Then tries variations with different prefixes
3. For asset requests, checks for files in multiple locations

Example for `/assets/main.js`:
- `web/static/dist/assets/main.js`
- `web/static/dist/assets/main.js` (without leading slash)
- `web/static/dist/assets/main.js` (explicit file join)

### WebSocket Exemption Logic

WebSocket connections are now exempted from rate limiting if they match any of:
- Standard WebSocket upgrade headers
- URLs starting with `/ws`
- URLs containing `/terminal/`
- URLs with query params containing `instance`

### Build Process Enhancements

The `build_frontend.sh` script now fixes both absolute and relative paths in the HTML:
- `/assets/` → `./assets/`
- `../assets/` → `./assets/`

## Debugging Tips

1. Check the log file at `/var/folders/wr/5pz9z8052jq7_m_q0h4pmcvh0000gn/T/claudesquad.log` for detailed debug information
2. Use the diagnostic pages at `/test.html` and `/asset-test.html` to verify functionality
3. Monitor the terminal output for "DEBUG:" entries that show asset resolution attempts
4. Use the robust test script to diagnose issues: `./test_robust_react.sh`

## Known Limitations

1. Embedded assets may not work correctly - prefer using file system assets
2. Some browser caching issues may require a hard refresh (Ctrl+F5)
3. WebSocket connections in older browsers might still face rate limiting