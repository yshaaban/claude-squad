#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Starting Claude Squad with minimal WebSocket test...${NC}"
echo -e "${GREEN}URL: http://localhost:8086/simple-test.html${NC}"
echo -e "${YELLOW}This is a minimal test to verify WebSocket functionality${NC}"
echo -e "${GREEN}Press Ctrl+C to stop${NC}"

# Kill any existing instances first to avoid conflicts
pkill -f "cs.*--web" || true

# Run with the web flag and specific port
./cs --web --web-port 8086 --log-to-file