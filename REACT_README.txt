# Claude Squad - React Frontend Integration

The React frontend has been successfully integrated into Claude Squad. This document provides a quick guide to using and testing the React frontend.

## How to Use

1. **Run with React frontend**:
   ```bash
   ./cs -s --web --react
   ```

2. **Access in browser**:
   ```
   http://localhost:8080/
   ```
   (or whatever port you specified with --web-port)

## Features

- Interactive terminal with WebSocket communication
- Instance listing with status indicators
- Responsive design for desktop and mobile
- Proper SPA routing for all paths

## Testing

Several test scripts are available to test the React frontend:

1. **Basic test**:
   ```bash
   ./test_web_only.sh
   ```

2. **Redirect test** (tests both legacy and React modes):
   ```bash
   ./test_web_redirect.sh
   ```

3. **Full React UI test**:
   ```bash
   ./test_react_frontend.sh
   ```

## Build Process

The React frontend is built using the `build_frontend.sh` script, which:
1. Builds the React app with npm
2. Copies the build output to web/static/dist/
3. Fixes asset paths if needed

## Documentation

For more detailed information, see:
- REACT_FRONTEND.md - Overview of the React frontend
- web/REACT_ARCHITECTURE.md - Technical architecture details
- web/IMPLEMENTATION_STATUS.md - Implementation status and details