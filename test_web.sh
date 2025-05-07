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

# Default port
PORT=8095
USE_REACT=false

# Process arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --react)
      USE_REACT=true
      shift
      ;;
    --port)
      PORT="$2"
      shift 2
      ;;
    --build-frontend)
      print_colored "Building frontend first..."
      ./build_frontend.sh
      shift
      ;;
    *)
      print_error "Unknown argument: $1"
      echo "Usage: $0 [--react] [--port PORT] [--build-frontend]"
      exit 1
      ;;
  esac
done

# Build the test server
print_colored "Building test server..."
go build -o test_react cmd/test_server/main.go

# Run the test server in the background
if [ "$USE_REACT" = true ]; then
  print_colored "Starting test server with React UI on port $PORT..."
  ./test_react --port $PORT --react &
else
  print_colored "Starting test server on port $PORT..."
  ./test_react --port $PORT &
fi
PID=$!

# Wait for server to start
sleep 1

# Test the server
print_colored "Testing web server..."
curl -s http://localhost:8095 | head -n 20

# Create a trap to kill the server when the script exits
trap "kill $PID" EXIT

if [ "$USE_REACT" = true ]; then
  print_colored "\nReact web UI is running on http://localhost:$PORT"
else
  print_colored "\nServer is running on http://localhost:$PORT"
fi
print_colored "Press Ctrl+C to stop the server"

# Wait for user to stop the server
wait $PID