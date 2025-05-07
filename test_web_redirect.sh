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

print_colored "Testing web UI redirect behavior..."

# Build the application if needed
if [ ! -f "cs_test" ]; then
  print_colored "Building application..."
  go build -o cs_test
fi

# Test legacy mode redirect
print_colored "Testing legacy mode redirect..."
./cs_test -s --web --web-port 8085 &
SERVER_PID=$!

# Give it a moment to start
sleep 2

# Test the redirect with curl
print_colored "Checking redirect from root path (legacy)..."
HEADERS=$(curl -s -I http://localhost:8085/)
REDIRECT_URL=$(echo "$HEADERS" | grep -i "location" | awk '{print $2}' | tr -d '\r')

if [[ "$HEADERS" == *"302 Found"* && "$REDIRECT_URL" == "/easy-terminal.html" ]]; then
    print_colored "SUCCESS: Legacy mode - Root path redirects to /easy-terminal.html"
else
    print_warning "UNEXPECTED: Legacy mode redirect behavior:"
    print_warning "$HEADERS"
fi

# Clean up
print_colored "Stopping legacy web server..."
kill $SERVER_PID
sleep 1

# Make sure React frontend is built
if [ ! -d "web/static/dist" ] || [ ! -f "web/static/dist/index.html" ]; then
  print_colored "Building React frontend..."
  ./build_frontend.sh
fi

# Test with React mode
print_colored "Testing React mode behavior..."
./cs_test -s --web --web-port 8085 --react &
SERVER_PID=$!

# Give it a moment to start
sleep 2

# Test the direct HTML access with curl
print_colored "Checking root path (React)..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8085/)
HTTP_CONTENT_TYPE=$(curl -s -I http://localhost:8085/ | grep -i "content-type" | awk '{print $2}' | tr -d '\r')

if [[ "$HTTP_CODE" == "200" ]]; then
    print_colored "SUCCESS: React mode - Root path returns 200 OK (React SPA routing working)"
    if [[ "$HTTP_CONTENT_TYPE" == "text/html"* ]]; then
        print_colored "SUCCESS: React mode - Content type is text/html"
    else
        print_warning "WARNING: React mode - Content type is $HTTP_CONTENT_TYPE, expected text/html"
    fi
else
    print_warning "UNEXPECTED: React mode - Root path returns $HTTP_CODE instead of 200"
fi

# Test SPA routing
print_colored "Testing SPA routing for non-existent path (React)..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8085/does-not-exist)

if [[ "$HTTP_CODE" == "200" ]]; then
    print_colored "SUCCESS: React SPA routing - Non-existent path returns 200 OK (handled by React router)"
else
    print_warning "UNEXPECTED: React SPA routing - Non-existent path returns $HTTP_CODE instead of 200"
fi

# Check asset loading
print_colored "Testing asset loading (React)..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8085/assets/)

if [[ "$HTTP_CODE" == "404" ]]; then
    print_colored "SUCCESS: Asset directory listing not allowed"
else
    print_warning "UNEXPECTED: Asset directory listing returns $HTTP_CODE instead of 404"
fi

# Clean up
print_colored "Stopping React web server..."
kill $SERVER_PID

print_colored "Redirect tests completed."