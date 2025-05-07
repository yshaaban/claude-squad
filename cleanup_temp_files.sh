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

print_colored "Starting cleanup of temporary and redundant files..."

# Function to safely remove files
remove_file() {
  if [ -f "$1" ]; then
    print_colored "Removing: $1"
    rm "$1"
  else
    print_warning "File not found: $1 (skipping)"
  fi
}

# Function to safely remove binary files
remove_binary() {
  if [ -f "$1" ]; then
    print_colored "Removing binary: $1"
    rm "$1"
  else
    print_warning "Binary not found: $1 (skipping)"
  fi
}

# Temporary/Patch Files
print_colored "\n=== Removing Temporary/Patch Files ==="
remove_file "/Users/ysh/src/claude-squad/app/app.go.patch"
remove_file "/Users/ysh/src/claude-squad/app/app.go.rej"
remove_file "/Users/ysh/src/claude-squad/app/app.go.tmp"
remove_file "/Users/ysh/src/claude-squad/web/server.go.orig"
remove_file "/Users/ysh/src/claude-squad/web/server.go.patch"
remove_file "/Users/ysh/src/claude-squad/web/server.go.rej"

# Test Binaries
print_colored "\n=== Removing Test Binaries ==="
remove_binary "/Users/ysh/src/claude-squad/simple_test_server"
remove_binary "/Users/ysh/src/claude-squad/test_server"
remove_binary "/Users/ysh/src/claude-squad/cs_test"
remove_binary "/Users/ysh/src/claude-squad/test_react"

# Redundant Test Scripts
print_colored "\n=== Removing Redundant Test Scripts ==="
remove_file "/Users/ysh/src/claude-squad/test_react_frontend.sh"
remove_file "/Users/ysh/src/claude-squad/standalone_react_test.sh"
remove_file "/Users/ysh/src/claude-squad/test_react_ws.sh"
remove_file "/Users/ysh/src/claude-squad/test_redirect.sh"
remove_file "/Users/ysh/src/claude-squad/run_test_server.sh"

# Duplicate Documentation
print_colored "\n=== Removing Duplicate Documentation ==="
remove_file "/Users/ysh/src/claude-squad/FIXED_README.md"
remove_file "/Users/ysh/src/claude-squad/terminal_issues.md"
remove_file "/Users/ysh/src/claude-squad/fix websockets.md"

# Obsolete Temporary Files
print_colored "\n=== Removing Obsolete Temporary Files ==="
remove_file "/Users/ysh/src/claude-squad/code_dump.txt"
remove_file "/Users/ysh/src/claude-squad/standalone_react_test.go"
remove_file "/Users/ysh/src/claude-squad/websocket_log.txt"
remove_file "/Users/ysh/src/claude-squad/websocket_test.js"

# Duplicate Implementation Plans
print_colored "\n=== Removing Duplicate Implementation Plans ==="
remove_file "/Users/ysh/src/claude-squad/implementation_strategy.md"
remove_file "/Users/ysh/src/claude-squad/web/IMPLEMENTATION_PLAN.md"
remove_file "/Users/ysh/src/claude-squad/web/IMPLEMENTATION_STATUS.md"

print_colored "\nCleanup completed!"
print_colored "Note: This script only removed files safe to delete that were identified during code review."
print_colored "You may need to run 'git status' to see if there are any other files that need attention."