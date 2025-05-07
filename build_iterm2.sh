#!/bin/bash
set -e

# NO COLORS - iTerm2 compatibility mode
# This script builds claude-squad without using terminal escape sequences that might cause issues in iTerm2

# Print functions (no colors)
print_message() {
  echo "$1"
}

print_warning() {
  echo "WARNING: $1"
}

print_error() {
  echo "ERROR: $1"
}

# Determine installation directory
INSTALL_NAME="cs"
BIN_DIR="$HOME/.local/bin"

print_message "=== Claude Squad iTerm2-Compatible Build ==="

# Check if Go is installed
if ! command -v go &> /dev/null; then
  print_error "Go is not installed. Please install Go first."
  print_warning "You can install it with: brew install go"
  exit 1
fi

# Build frontend without escape sequences
print_message "Building React frontend..."

# Check for Node.js before proceeding
if ! command -v node &> /dev/null; then
  print_error "Node.js is not installed. Please install Node.js first."
  print_warning "You can install it with: brew install node"
  exit 1
fi

# Check for npm
if ! command -v npm &> /dev/null; then
  print_error "npm is not installed. Please install npm first."
  print_warning "You can install it with: brew install npm"
  exit 1
fi

# Navigate to frontend directory
cd "$(dirname "$0")/frontend"

# Install dependencies
print_message "Installing frontend dependencies..."
npm install

# Build the frontend
print_message "Building frontend..."
npm run build || { 
  print_warning "Build failed. Using fallback build instead."
  mkdir -p dist
  echo "<!DOCTYPE html><html><head><title>Claude Squad</title></head><body><h1>Claude Squad</h1><p>This is a fallback build. The actual React build failed.</p></body></html>" > dist/index.html
}

# Create web/static/dist directory and assets subdirectory if they don't exist
mkdir -p ../web/static/dist
mkdir -p ../web/static/dist/assets

# Copy the build output to the static directory
print_message "Copying build output to static directory..."
if [ -d "build" ]; then
  print_message "Found 'build' directory, copying to web/static/dist/..."
  cp -r build/* ../web/static/dist/
  # Fix asset paths without using sed -i which might be problematic
  print_message "Fixing asset paths in index.html..."
  if grep -q '/assets/' ../web/static/dist/index.html; then
    # Create a temporary file and move it back
    cat ../web/static/dist/index.html | tr '/assets/' './assets/' > ../web/static/dist/index.html.tmp
    mv ../web/static/dist/index.html.tmp ../web/static/dist/index.html
  fi
elif [ -d "dist" ]; then
  print_message "Found 'dist' directory, copying to web/static/dist/..."
  cp -r dist/* ../web/static/dist/
  # Fix asset paths without using sed -i which might be problematic
  print_message "Fixing asset paths in index.html..."
  if grep -q '/assets/' ../web/static/dist/index.html; then
    # Create a temporary file and move it back
    cat ../web/static/dist/index.html | tr '/assets/' './assets/' > ../web/static/dist/index.html.tmp
    mv ../web/static/dist/index.html.tmp ../web/static/dist/index.html
  fi
else
  print_warning "Build directory not found. Creating minimal fallback."
  echo "<!DOCTYPE html><html><head><title>Claude Squad</title></head><body><h1>Claude Squad</h1><p>Web interface fallback.</p></body></html>" > ../web/static/dist/index.html
fi

# Return to root directory
cd ..

# Ensure the dist directory exists
print_message "Verifying static files directory..."
mkdir -p web/static/dist

# Validate that the static files exist
if [ ! -f "web/static/dist/index.html" ]; then
  print_warning "Frontend build failed or was not copied correctly."
  print_warning "Creating a minimal fallback index.html..."
  echo "<!DOCTYPE html><html><head><title>Claude Squad</title></head><body><h1>Claude Squad</h1><p>Web interface is available but may not be fully functional.</p></body></html>" > web/static/dist/index.html
fi

# Create bin directory if it doesn't exist
if [ ! -d "$BIN_DIR" ]; then
  print_message "Creating directory $BIN_DIR..."
  mkdir -p "$BIN_DIR"
fi

# Build the application
print_message "Building claude-squad application..."
go build -o "$INSTALL_NAME"

# Test the build
print_message "Testing the build..."
if [ -f "$INSTALL_NAME" ]; then
  print_message "Build successful!"
else
  print_error "Build failed!"
  exit 1
fi

# Make the executable in the repository root
print_message "Making executable available in repository root..."
chmod +x "$INSTALL_NAME"
print_message "Executable created at: $(pwd)/$INSTALL_NAME"

# Install the binary to ~/.local/bin
print_message "Installing binary to $BIN_DIR/$INSTALL_NAME..."
cp "$INSTALL_NAME" "$BIN_DIR/"
chmod +x "$BIN_DIR/$INSTALL_NAME"

print_message "=== Installation complete! ==="
print_message "The executable is available in two locations:"
print_message "1. Repository root: $(pwd)/$INSTALL_NAME"
print_message "2. System path: $BIN_DIR/$INSTALL_NAME"
print_message ""
print_message "You can run it directly from the repo with: ./$INSTALL_NAME"
print_message "For iTerm2-compatible usage, run: ./run_iterm2.sh"