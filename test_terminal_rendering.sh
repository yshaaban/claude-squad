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

print_colored "Building frontend with improved terminal component..."
./build_frontend.sh

print_colored "Building application with improved connection handling..."
go build -o cs_test

# Run the application
print_colored "Starting application with fixed terminal rendering and connection handling..."
print_colored "The terminal should now show:"
print_colored "1. A status bar at the top instead of text mixed with terminal output"
print_colored "2. No duplicate content when reconnecting"
print_colored "3. Cleaner terminal output without 'Connecting...' messages in output"
print_colored "4. More stable WebSocket connections with less reconnection issues"
print_colored ""
print_colored "Press Ctrl+C to stop the server"

./cs_test -s --web --web-port 8085 --react --log-to-file