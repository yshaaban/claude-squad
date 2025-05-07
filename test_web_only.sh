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

PORT=8085
USE_REACT=true

# Process arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --legacy)
      USE_REACT=false
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
      echo "Usage: $0 [--legacy] [--port PORT] [--build-frontend]"
      exit 1
      ;;
  esac
done

# Build the test binary
print_colored "Building test binary..."
go build -o cs_test

# Check if we need to build the frontend
if [ "$USE_REACT" = true ] && [ ! -f "web/static/dist/index.html" ]; then
  print_warning "React frontend not found. Building it first..."
  ./build_frontend.sh
fi

# Run with appropriate options
if [ "$USE_REACT" = true ]; then
  print_colored "Starting web server with React UI on port $PORT..."
  print_colored "Access at http://localhost:$PORT/"
  ./cs_test --web --web-port $PORT --react --log-to-file
else
  print_colored "Starting web server with legacy UI on port $PORT..."
  print_colored "Access at http://localhost:$PORT/"
  ./cs_test --web --web-port $PORT --log-to-file
fi