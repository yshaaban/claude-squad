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
  print_warning "Go is not installed. Please install Go first."
  print_warning "You can install it with: brew install go"
  exit 1
fi

# Check if Node.js is installed
if ! command -v node &> /dev/null; then
  print_warning "Node.js is not installed. Please install Node.js first."
  print_warning "You can install it with: brew install node"
  exit 1
fi

# Check if tmux is installed
if ! command -v tmux &> /dev/null; then
  print_warning "tmux is not installed. Please install tmux first."
  print_warning "You can install it with: brew install tmux"
  exit 1
fi

print_colored "Starting development environment..."

# Create a tmux session for development
SESSION_NAME="claude-squad-dev"

# Kill existing session if it exists
tmux kill-session -t "$SESSION_NAME" 2>/dev/null || true

# Create new session
tmux new-session -d -s "$SESSION_NAME" -n "Backend"

# Configure the first window for the Go backend
tmux send-keys -t "$SESSION_NAME:Backend" "cd $(pwd) && go run main.go -s --web --web-port 8085 --log-to-file" C-m

# Create a window for the React frontend
tmux new-window -t "$SESSION_NAME" -n "Frontend"
tmux send-keys -t "$SESSION_NAME:Frontend" "cd $(pwd)/frontend && npm install && npm run dev" C-m

# Create a window for git/other commands
tmux new-window -t "$SESSION_NAME" -n "Tools"
tmux send-keys -t "$SESSION_NAME:Tools" "cd $(pwd)" C-m

# Attach to the session
print_colored "Development environment started. Attaching to tmux session..."
print_colored "Use Ctrl+B D to detach from the session without stopping it."
print_colored "Use Ctrl+B 0/1/2 to switch between windows."

# Sleep briefly to allow processes to start
sleep 1

tmux attach-session -t "$SESSION_NAME"