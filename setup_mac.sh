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

# Check if Go is installed
if ! command -v go &> /dev/null; then
  echo "Go is not installed. Please install Go first."
  echo "You can install it with: brew install go"
  exit 1
fi

# Determine installation directory
INSTALL_DIR="$HOME/.local/bin"
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
  print_warning "Warning: $INSTALL_DIR is not in your PATH. You may need to add it."
  print_warning "Add the following to your ~/.zshrc or ~/.bash_profile:"
  print_warning "    export PATH=\"\$PATH:$INSTALL_DIR\""
  
  # Create directory if it doesn't exist
  if [ ! -d "$INSTALL_DIR" ]; then
    print_colored "Creating directory $INSTALL_DIR..."
    mkdir -p "$INSTALL_DIR"
  fi
else
  print_colored "Installation directory $INSTALL_DIR is in your PATH."
fi

# Build the application
print_colored "Building claude-squad application..."
go build -o cs main.go

# Install the binary
print_colored "Installing binary to $INSTALL_DIR/cs..."
cp cs "$INSTALL_DIR/"
chmod +x "$INSTALL_DIR/cs"

# Clean up
print_colored "Cleaning up..."
rm -f cs

print_colored "✅ Installation complete!"
print_colored "You can now run 'cs' from anywhere in your terminal."
print_colored "For Simple Mode (recommended): cs -s"
print_colored "For debugging with file logging: cs --log-to-file"