#!/bin/bash

# Color output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color
BLUE='\033[0;34m'

# Print a colorized message
print_colored() {
  echo -e "${GREEN}$1${NC}"
}

# Print error message
print_error() {
  echo -e "${RED}$1${NC}" 
}

# Print info message
print_info() {
  echo -e "${BLUE}$1${NC}"
}

# Make this script executable
chmod +x "$0"

print_colored "Rebuilding Claude Squad React frontend and Go application..."

# Check if npm is installed
if ! command -v npm &> /dev/null; then
  print_error "npm not found. Please install Node.js/npm to build the frontend."
  exit 1
fi

# Navigate to frontend directory and build
print_colored "Building React frontend..."
cd frontend || { print_error "Frontend directory not found"; exit 1; }

# Install dependencies if node_modules doesn't exist
if [ ! -d "node_modules" ]; then
  print_info "Installing frontend dependencies..."
  npm install || { print_error "Failed to install dependencies"; exit 1; }
fi

# Build the React app
npm run build || { print_error "Failed to build React app"; exit 1; }

# Go back to root directory
cd ..

# Check where the build directory is
BUILD_DIR=""
if [ -d "frontend/dist" ]; then
  BUILD_DIR="frontend/dist"
elif [ -d "frontend/build" ]; then
  BUILD_DIR="frontend/build"
else
  print_error "Could not find build directory (checked frontend/dist and frontend/build)"
  exit 1
fi

# Copy the build to the web/static/dist directory
print_colored "Copying build from ${BUILD_DIR} to web/static/dist..."
mkdir -p web/static/dist
rm -rf web/static/dist/*
cp -r "$BUILD_DIR"/* web/static/dist/

# Fix asset paths - always ensure they use relative paths starting with ./
print_colored "Fixing asset paths in index.html..."
if grep -q '/assets/' web/static/dist/index.html; then
  # Convert absolute paths to relative
  sed -i '' 's|/assets/|./assets/|g' web/static/dist/index.html
fi
if grep -q '\.\./assets/' web/static/dist/index.html; then
  # Fix parent directory paths
  sed -i '' 's|../assets/|./assets/|g' web/static/dist/index.html
fi

# Build the Go app
print_colored "Building Go application..."
go build -o cs || { print_error "Failed to build Go app"; exit 1; }

print_colored "Build completed successfully!"
print_info "Try running: ./standalone_react_test.sh to test the React frontend"
print_info "Or: ./cs --web --react --web-port=8086 to run the full application"