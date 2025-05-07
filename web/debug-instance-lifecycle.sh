#!/bin/bash
set -e

# Debug script for instance lifecycle in Simple Mode
echo "========================================"
echo "Claude Squad Simple Mode Instance Lifecycle Debug"
echo "========================================"

# Colors for better readability
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Build the application
echo -e "${BLUE}Building Claude Squad...${NC}"
cd "$(dirname "$0")/.."
go build -o cs

# Check if build was successful
if [ ! -f ./cs ]; then
  echo -e "${RED}Failed to build Claude Squad!${NC}"
  exit 1
fi
echo -e "${GREEN}✓ Build successful${NC}"

# Enable debug logging for extra visibility
echo -e "${BLUE}Enabling debug logging in monitor.go...${NC}"
sed -i.bak 's/const debugLogging = false/const debugLogging = true/g' web/monitor.go
echo -e "${GREEN}✓ Debug logging enabled${NC}"

# Rebuild with debug logging
echo -e "${BLUE}Rebuilding with debug logging...${NC}"
go build -o cs
echo -e "${GREEN}✓ Rebuild successful${NC}"

# Start with more aggressive logging to file
echo -e "${BLUE}Starting Claude Squad in Simple Mode with web monitoring and logging...${NC}"
./cs -s --web --log-to-file &
APP_PID=$!

# Create a function to clean up on exit
cleanup() {
  echo -e "\n${BLUE}Cleaning up...${NC}"
  kill $APP_PID 2>/dev/null || true
  echo -e "${GREEN}Restoring debug flag to original state${NC}"
  mv web/monitor.go.bak web/monitor.go
  echo -e "${GREEN}Done!${NC}"
}

# Set up cleanup on script exit
trap cleanup EXIT INT

# Wait a bit for initialization
echo -e "${YELLOW}Waiting for initialization...${NC}"
sleep 10
echo -e "${GREEN}✓ Initial wait complete${NC}"

# Check if tmux session exists
echo -e "${BLUE}Checking for tmux sessions...${NC}"
TMUX_SESSIONS=$(tmux ls 2>/dev/null || echo "No tmux sessions found")
echo -e "Tmux sessions:\n${TMUX_SESSIONS}"

# Check instance status via API
echo -e "${BLUE}Checking instance API...${NC}"
curl -s http://localhost:8099/api/instances | jq . || echo -e "${RED}API request failed${NC}"

# Tail the log file
echo -e "${BLUE}Showing recent logs...${NC}"
LOG_FILE="/var/folders/wr/5pz9z8052jq7_m_q0h4pmcvh0000gn/T/claudesquad.log"
if [ -f "$LOG_FILE" ]; then
  echo -e "Last 50 log entries:"
  tail -n 50 "$LOG_FILE"
else
  echo -e "${RED}Log file not found at $LOG_FILE${NC}"
fi

# Check if instance storage file exists
echo -e "${BLUE}Checking instance storage file...${NC}"
STORAGE_FILE="$HOME/.claude-squad/instances.json"
if [ -f "$STORAGE_FILE" ]; then
  echo -e "Instance storage file found. Contents:"
  cat "$STORAGE_FILE" | jq . || cat "$STORAGE_FILE"
else
  echo -e "${RED}Instance storage file not found${NC}"
fi

echo -e "\n${GREEN}=======================================${NC}"
echo -e "${GREEN}Debug information collected${NC}"
echo -e "${GREEN}=======================================${NC}"

# Keep running for interactive testing
echo -e "${YELLOW}Press Ctrl+C to stop and exit${NC}"
wait $APP_PID