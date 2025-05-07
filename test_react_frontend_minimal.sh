#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Print with color
print_colored() {
  echo -e "${GREEN}$1${NC}"
}

print_warning() {
  echo -e "${YELLOW}$1${NC}"
}

print_error() {
  echo -e "${RED}$1${NC}"
}

# Make sure the static directory exists
if [ ! -d "web/static/dist" ]; then
  print_error "Error: web/static/dist directory does not exist!"
  exit 1
fi

# Check if test.html exists
if [ ! -f "web/static/dist/test.html" ]; then
  print_warning "Warning: test.html not found, creating it..."
  cat > web/static/dist/test.html << 'EOL'
<!DOCTYPE html>
<html>
<head>
  <title>Claude Squad Test Page</title>
  <style>
    body { font-family: sans-serif; margin: 20px; }
  </style>
</head>
<body>
  <h1>Claude Squad Test Page</h1>
  <p>If you can see this page, static file serving is working correctly!</p>
</body>
</html>
EOL
fi

# Run the CLI with minimal options just to test the web server
print_colored "Starting Claude Squad with minimal React web UI..."
print_colored "This skips most features and only tests the web server!"
print_colored ""
print_colored "Access these test URLs:"
print_colored "  http://localhost:8086/test.html     - Simple test page"
print_colored "  http://localhost:8086/asset-test.html - Test asset loading"
print_colored "  http://localhost:8086/              - Minimal React page"
print_colored ""
print_colored "Press Ctrl+C to stop"
print_colored ""

# Use --web-only flag to skip tmux-related features for testing
if [ -f "./cs_test" ]; then
  # Run with --web and --react flags, using a different port to avoid conflicts
  ./cs_test --web --web-port 8086 --react --log-to-file
else
  print_error "Error: cs_test binary not found. Building it first..."
  go build -o cs_test
  ./cs_test --web --web-port 8086 --react --log-to-file
fi