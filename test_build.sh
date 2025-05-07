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

# Build the application with React frontend
print_colored "Building the application with React frontend..."

# Build the React frontend
print_colored "Step 1: Building React frontend..."
./build_frontend.sh

# Build the Go application
print_colored "Step 2: Building Go application..."
go build -o cs_test

# Verify the build was successful
if [ -f "cs_test" ]; then
    print_colored "✅ Build successful! The application is ready to run."
    print_colored "You can run it with: ./cs_test -s --web --react"
    print_colored "Or test it with: ./test_web_redirect.sh"
else
    print_error "❌ Build failed! The application binary was not created."
    exit 1
fi