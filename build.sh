#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print with color
print_colored() {
  echo -e "${GREEN}$1${NC}"
}

print_warning() {
  echo -e "${YELLOW}$1${NC}"
}

# Determine installation directory
INSTALL_NAME="cs"
BIN_DIR="$HOME/.local/bin"

# Check if Go is installed
if ! command -v go &> /dev/null; then
  print_warning "Go is not installed. Please install Go first."
  print_warning "You can install it with: brew install go"
  exit 1
fi

# Make build_frontend.sh executable
chmod +x ./build_frontend.sh

# Build the frontend
print_colored "Building the React frontend..."
./build_frontend.sh

# Ensure the dist directory exists
print_colored "Verifying static files directory..."
mkdir -p web/static/dist

# Validate that the static files exist
if [ ! -f "web/static/dist/index.html" ]; then
  print_warning "Frontend build failed or was not copied correctly."
  print_warning "Creating a minimal fallback index.html..."
  echo "<!DOCTYPE html><html><head><title>Claude Squad</title></head><body><h1>Claude Squad</h1><p>Web interface is available but may not be fully functional.</p></body></html>" > web/static/dist/index.html
fi

# Create bin directory if it doesn't exist
if [ ! -d "$BIN_DIR" ]; then
  print_colored "Creating directory $BIN_DIR..."
  mkdir -p "$BIN_DIR"
fi

# Build the application
print_colored "Building claude-squad application..."
go build -o "$INSTALL_NAME"

# Test the build
print_colored "Testing the build..."
if [ -f "$INSTALL_NAME" ]; then
  print_colored "Build successful!"
else
  print_warning "Build failed!"
  exit 1
fi

# Make the executable in the repository root
print_colored "Making executable available in repository root..."
chmod +x "$INSTALL_NAME"
print_colored "Executable created at: $(pwd)/$INSTALL_NAME"

# Install the binary to ~/.local/bin
print_colored "Installing binary to $BIN_DIR/$INSTALL_NAME..."
cp "$INSTALL_NAME" "$BIN_DIR/"
chmod +x "$BIN_DIR/$INSTALL_NAME"

# No cleanup - keep the binary in the repo root
print_colored "The binary is available both in the repository root and in $BIN_DIR/"

print_colored "✅ Installation complete!"
print_colored "The executable is available in two locations:"
print_colored "1. Repository root: $(pwd)/$INSTALL_NAME"
print_colored "2. System path: $BIN_DIR/$INSTALL_NAME"
print_colored ""
print_colored "You can run it directly from the repo with: ./$INSTALL_NAME"
print_colored "Or from anywhere using: $INSTALL_NAME"
print_colored ""
print_colored "Common usage:"
print_colored "For Simple Mode (recommended): ./$INSTALL_NAME -s"
print_colored "For debugging with file logging: ./$INSTALL_NAME --log-to-file"
print_colored "To enable the web interface: ./$INSTALL_NAME -s --web"
print_colored "To enable the React web interface: ./$INSTALL_NAME -s --web --react"