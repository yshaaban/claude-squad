#!/bin/bash
set -e

# Direct test for the terminal websocket with the simple-terminal.html interface
echo "========================================"
echo "Claude Squad WebSocket Terminal Test"
echo "========================================"

# Test parameters
PORT=8099
TEST_DURATION=60

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

# Start the application with web server only (no tmux/pty), with detailed logging
echo -e "${BLUE}Starting Claude Squad with web server only and detailed logging...${NC}"
./cs --web --web-port=$PORT --log-to-file &
APP_PID=$!

# Create a function to clean up on exit
cleanup() {
  echo -e "\n${BLUE}Cleaning up...${NC}"
  kill $APP_PID 2>/dev/null || true
  echo -e "${GREEN}Done!${NC}"
}

# Set up cleanup on script exit
trap cleanup EXIT INT

# Wait for server to start
echo -e "${YELLOW}Waiting for server to start...${NC}"
for i in {1..20}; do
  sleep 1
  echo -n "."
done
echo ""

# Check if server is responding
echo -e "${BLUE}Testing web server...${NC}"
if curl -s http://localhost:$PORT > /dev/null 2>&1; then
  echo -e "${GREEN}✓ Web UI is responding${NC}"
else
  echo -e "${RED}✗ Web UI is not responding${NC}"
  echo -e "Debug: Attempting direct curl to see response:"
  curl -v http://localhost:$PORT
  exit 1
fi

# Manually create a test instance for websocket testing
echo -e "${BLUE}Creating a test instance...${NC}"
INSTANCE_NAME="test-instance"

# Show instructions for testing terminal
echo ""
echo -e "${GREEN}=======================================${NC}"
echo -e "${GREEN}Simple Terminal Test is running!${NC}"
echo -e "${GREEN}=======================================${NC}"
echo ""
echo -e "Open your browser and navigate to:"
echo -e "${BLUE}http://localhost:$PORT/simple-terminal.html?instance=$INSTANCE_NAME${NC}"
echo ""
echo -e "Click 'Connect' to establish WebSocket connection."
echo -e "You should see terminal output appear in the terminal window."
echo -e "Check the debug log for detailed messaging information."
echo ""
echo -e "${YELLOW}Server will remain active for $TEST_DURATION seconds for testing.${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop the test at any time.${NC}"
echo ""

# Keep server running for the specified time to allow manual testing
for i in $(seq 1 $TEST_DURATION); do
  echo -ne "\rTime remaining: ${YELLOW}$(($TEST_DURATION - $i))s${NC} "
  sleep 1
done

echo -e "\n\n${GREEN}Test completed!${NC}"
echo -e "You can check the logs at /tmp/claudesquad.log for detailed information."