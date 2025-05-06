#!/bin/bash
set -e

# Direct test for the web server implementation in Simple Mode
echo "========================================"
echo "Claude Squad Web Server Test (Simple Mode)"
echo "========================================"

# Test parameters
PORT=8099
TEST_DURATION=30

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

# Start the application with web server in Simple Mode
echo -e "${BLUE}Starting Claude Squad in Simple Mode with web monitoring...${NC}"
./cs -s --web --web-port=$PORT &
APP_PID=$!

# Create a function to clean up on exit
cleanup() {
  echo -e "\n${BLUE}Cleaning up...${NC}"
  kill $APP_PID 2>/dev/null || true
  echo -e "${GREEN}Done!${NC}"
}

# Set up cleanup on script exit
trap cleanup EXIT INT

# Wait for server to start and for a Claude session to be created automatically
echo -e "${YELLOW}Waiting for server to start and Claude session to initialize...${NC}"
for i in {1..60}; do
  sleep 1
  echo -n "."
done
echo ""

# Check if server is responding
echo -e "${BLUE}Testing web server...${NC}"
if curl -s -v http://localhost:$PORT > /dev/null 2>&1; then
  echo -e "${GREEN}✓ Web UI is responding${NC}"
else
  echo -e "${RED}✗ Web UI is not responding${NC}"
  echo -e "Debug: Attempting direct curl to see response:"
  curl -v http://localhost:$PORT
  exit 1
fi

# Check API endpoint
echo -e "${BLUE}Testing API endpoint...${NC}"
if curl -s http://localhost:$PORT/api/instances > api_response.txt; then
  echo -e "${GREEN}✓ API endpoint is responding${NC}"
  echo -e "API Response: $(cat api_response.txt)"
  rm api_response.txt
else
  echo -e "${RED}✗ API endpoint is not responding${NC}"
  exit 1
fi

# Success message - server is ready for manual testing
echo ""
echo -e "${GREEN}=======================================${NC}"
echo -e "${GREEN}Web monitoring server is running!${NC}"
echo -e "${GREEN}=======================================${NC}"
echo ""
echo -e "Open your browser and navigate to:"
echo -e "${BLUE}http://localhost:$PORT${NC}"
echo ""
echo -e "You can interact with Claude in the terminal."
echo -e "The web interface will show terminal output and changes in real-time."
echo ""
echo -e "${YELLOW}Server will remain active for $TEST_DURATION seconds for testing.${NC}"
echo -e "${YELLOW}Press Ctrl+C to stop the test at any time.${NC}"
echo ""

# Keep server running for the specified time to allow manual testing
for i in $(seq 1 $TEST_DURATION); do
  echo -ne "\rTime remaining: ${YELLOW}$(($TEST_DURATION - $i))s${NC} "
  sleep 1
done

echo -e "\n\n${GREEN}Test completed successfully!${NC}"
