#!/bin/bash
set -e

# E2E test specifically for NoTTY mode terminal WebSocket compatibility
echo "========================================"
echo "Claude Squad NoTTY WebSocket Debug Test"
echo "========================================"

# Colors for better readability
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Build the application with debug logging
echo -e "${BLUE}Building Claude Squad with debugging...${NC}"
cd "$(dirname "$0")/.."

# Ensure debug logging is enabled
if grep -q "const debugLogging = false" web/monitor.go; then
  echo -e "${YELLOW}Enabling debug logging in monitor.go...${NC}"
  sed -i.bak 's/const debugLogging = false/const debugLogging = true/g' web/monitor.go
  RESTORE_DEBUG=true
else
  echo -e "${GREEN}Debug logging already enabled${NC}"
  RESTORE_DEBUG=false
fi

go build -o cs
echo -e "${GREEN}✓ Build successful${NC}"

# Create a function to clean up on exit
cleanup() {
  echo -e "\n${BLUE}Cleaning up...${NC}"
  kill $APP_PID 2>/dev/null || true
  
  # Restore debug flag if needed
  if [ "$RESTORE_DEBUG" = true ]; then
    echo -e "${GREEN}Restoring debug flag${NC}"
    mv web/monitor.go.bak web/monitor.go
  fi
  
  echo -e "${GREEN}Done!${NC}"
}

# Set up cleanup on script exit
trap cleanup EXIT INT

# Start the application with web server and NoTTY mode
echo -e "${BLUE}Starting Claude Squad with NoTTY mode + web server...${NC}"
./cs -s --web --log-to-file --no-tty &
APP_PID=$!

# Wait for server to start
echo -e "${YELLOW}Waiting for server to start...${NC}"
for i in {1..10}; do
  sleep 1
  echo -n "."
done
echo ""

# Check if server is responding
echo -e "${BLUE}Testing web server...${NC}"
if curl -s http://localhost:8099 > /dev/null 2>&1; then
  echo -e "${GREEN}✓ Web UI is responding${NC}"
else
  echo -e "${RED}✗ Web UI is not responding${NC}"
  exit 1
fi

# Check for tmux sessions
echo -e "${BLUE}Checking for tmux sessions...${NC}"
TMUX_SESSIONS=$(tmux ls 2>/dev/null || echo "No tmux sessions found")
echo -e "Tmux sessions:\n${TMUX_SESSIONS}"

# Check API for instances
echo -e "${BLUE}Checking API for instances...${NC}"
INSTANCES=$(curl -s http://localhost:8099/api/instances)
echo "$INSTANCES" | jq .

# Extract instance name if available
INSTANCE_TITLE=$(echo "$INSTANCES" | jq -r '.instances[0].title')
if [ "$INSTANCE_TITLE" != "null" ] && [ -n "$INSTANCE_TITLE" ]; then
  echo -e "${GREEN}Found instance: $INSTANCE_TITLE${NC}"
  
  # Get instance details
  echo -e "${BLUE}Getting instance details...${NC}"
  curl -s "http://localhost:8099/api/instances/$INSTANCE_TITLE" | jq .
  
  # Try to get terminal content
  echo -e "${BLUE}Getting terminal content...${NC}"
  curl -s "http://localhost:8099/api/instances/$INSTANCE_TITLE/output" | jq .
else
  echo -e "${RED}No instances found${NC}"
fi

# Check session state in storage
echo -e "${BLUE}Checking instance storage file...${NC}"
STORAGE_FILE="$HOME/.claude-squad/instances.json"
if [ -f "$STORAGE_FILE" ]; then
  echo -e "Instance storage file contents:"
  cat "$STORAGE_FILE" | jq .
else
  echo -e "${RED}Instance storage file not found${NC}"
fi

# Check log file
echo -e "${BLUE}Checking log file for errors...${NC}"
LOG_FILE="/tmp/claudesquad.log"
if [ -f "$LOG_FILE" ]; then
  echo -e "${YELLOW}Last 30 log entries:${NC}"
  tail -n 30 "$LOG_FILE"
  
  echo -e "${YELLOW}WebSocket connection related logs:${NC}"
  grep -i websocket "$LOG_FILE" | tail -n 10
  
  echo -e "${YELLOW}NoTTY mode related logs:${NC}"
  grep -i "notty" "$LOG_FILE" | tail -n 10
  
  echo -e "${YELLOW}Monitor related logs:${NC}"
  grep -i "monitor" "$LOG_FILE" | tail -n 10
  
  echo -e "${YELLOW}Tmux related logs:${NC}"
  grep -i "tmux" "$LOG_FILE" | tail -n 10
else
  echo -e "${RED}Log file not found at $LOG_FILE${NC}"
fi

echo -e "\n${GREEN}=======================================${NC}"
echo -e "${GREEN}Debug information collected${NC}"
echo -e "${GREEN}=======================================${NC}"

# Keep running for interactive testing
echo -e "${YELLOW}Press Ctrl+C to stop and exit${NC}"
wait $APP_PID