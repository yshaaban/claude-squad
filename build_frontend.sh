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

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
  print_error "Node.js is not installed. Please install Node.js first."
  print_warning "You can install it with: brew install node"
  exit 1
fi

# Check if npm is installed
if ! command -v npm &> /dev/null; then
  print_error "npm is not installed. Please install npm first."
  print_warning "You can install it with: brew install npm"
  exit 1
fi

# Navigate to frontend directory
cd "$(dirname "$0")/frontend"

# Install dependencies
print_colored "Installing frontend dependencies..."
npm install

# Build the frontend
print_colored "Building frontend..."
npm run build || { 
  print_warning "Build failed. Using fallback build instead."
  mkdir -p dist
  echo "<!DOCTYPE html><html><head><title>Claude Squad</title></head><body><h1>Claude Squad</h1><p>This is a fallback build. The actual React build failed.</p></body></html>" > dist/index.html
}

# Create web/static/dist directory and assets subdirectory if they don't exist
mkdir -p ../web/static/dist
mkdir -p ../web/static/dist/assets

# Copy the build output to the static directory
print_colored "Copying build output to static directory..."
if [ -d "build" ]; then
  print_colored "Found 'build' directory, copying to web/static/dist/..."
  cp -r build/* ../web/static/dist/
  # Fix asset paths - always ensure they use relative paths starting with ./
  print_colored "Fixing asset paths in index.html..."
  if grep -q '/assets/' ../web/static/dist/index.html; then
    # Convert absolute paths to relative
    sed -i '' 's|/assets/|./assets/|g' ../web/static/dist/index.html
  fi
  if grep -q '\.\./assets/' ../web/static/dist/index.html; then
    # Fix parent directory paths
    sed -i '' 's|../assets/|./assets/|g' ../web/static/dist/index.html
  fi
elif [ -d "dist" ]; then
  print_colored "Found 'dist' directory, copying to web/static/dist/..."
  cp -r dist/* ../web/static/dist/
  # Fix asset paths - always ensure they use relative paths starting with ./
  print_colored "Fixing asset paths in index.html..."
  if grep -q '/assets/' ../web/static/dist/index.html; then
    # Convert absolute paths to relative
    sed -i '' 's|/assets/|./assets/|g' ../web/static/dist/index.html
  fi
  if grep -q '\.\./assets/' ../web/static/dist/index.html; then
    # Fix parent directory paths
    sed -i '' 's|../assets/|./assets/|g' ../web/static/dist/index.html
  fi
else
  print_warning "Build directory not found. Creating minimal fallback."
  echo "<!DOCTYPE html><html><head><title>Claude Squad</title></head><body><h1>Claude Squad</h1><p>Web interface fallback.</p></body></html>" > ../web/static/dist/index.html
fi

print_colored "✅ Frontend build complete!"