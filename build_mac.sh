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

# Check for tmux
if ! command -v tmux &> /dev/null; then
  print_warning "tmux is not installed. Installing tmux..."
  if command -v brew &> /dev/null; then
    brew install tmux
  else
    print_warning "Homebrew is not installed. Please install Homebrew first to install tmux."
    print_warning "Visit https://brew.sh for installation instructions."
    exit 1
  fi
  print_colored "tmux installed successfully."
else
  print_colored "tmux is already installed."
fi

# Check for GitHub CLI (gh)
if ! command -v gh &> /dev/null; then
  print_warning "GitHub CLI (gh) is not installed. Installing GitHub CLI..."
  if command -v brew &> /dev/null; then
    brew install gh
  else
    print_warning "Homebrew is not installed. Please install Homebrew first to install GitHub CLI."
    print_warning "Visit https://brew.sh for installation instructions."
    exit 1
  fi
  print_colored "GitHub CLI (gh) installed successfully."
else
  print_colored "GitHub CLI (gh) is already installed."
fi

# Determine installation directory
INSTALL_NAME="cs"
BIN_DIR="$HOME/.local/bin"

# Set up shell profile if needed
setup_shell_path() {
  case $SHELL in
    */zsh)
      PROFILE=$HOME/.zshrc
      ;;
    */bash)
      PROFILE=$HOME/.bashrc
      ;;
    */fish)
      PROFILE=$HOME/.config/fish/config.fish
      ;;
    */ash)
      PROFILE=$HOME/.profile
      ;;
    *)
      print_warning "Could not detect shell, you may need to manually add ${BIN_DIR} to your PATH."
      return
  esac

  if [[ ":$PATH:" != *":${BIN_DIR}:"* ]]; then
    print_colored "Adding $BIN_DIR to your PATH in $PROFILE"
    echo >> "$PROFILE" && echo "export PATH=\"\$PATH:$BIN_DIR\"" >> "$PROFILE"
    print_warning "You'll need to restart your terminal or run 'source $PROFILE' for the changes to take effect."
  fi
}

# Create bin directory if it doesn't exist
if [ ! -d "$BIN_DIR" ]; then
  print_colored "Creating directory $BIN_DIR..."
  mkdir -p "$BIN_DIR"
fi

# Build the application
print_colored "Building claude-squad application..."
go build -o "$INSTALL_NAME"

# Install the binary
print_colored "Installing binary to $BIN_DIR/$INSTALL_NAME..."
cp "$INSTALL_NAME" "$BIN_DIR/"
chmod +x "$BIN_DIR/$INSTALL_NAME"

# Clean up
print_colored "Cleaning up..."
rm -f "$INSTALL_NAME"

# Set up PATH if needed
setup_shell_path

print_colored "✅ Installation complete!"
print_colored "You can now run '$INSTALL_NAME' from anywhere in your terminal."
print_colored "For Simple Mode (recommended): $INSTALL_NAME -s"
print_colored "For debugging with file logging: $INSTALL_NAME --log-to-file"