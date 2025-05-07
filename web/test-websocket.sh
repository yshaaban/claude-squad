#!/bin/bash
set -e

# Terminal WebSocket integration test
echo "========================================"
echo "Claude Squad Terminal WebSocket Integration Test"
echo "========================================"

# Colors for better readability
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Note about test status
echo -e "${YELLOW}The unit test for WebSocket terminal streaming is currently skipped${NC}"
echo -e "${YELLOW}due to issues with the mock package. Use the E2E test instead:${NC}"
echo -e "${BLUE}./web/test-e2e-websocket.sh${NC}"
echo ""

# Show test availability
echo -e "${BLUE}Available Terminal WebSocket tests:${NC}"
echo -e "1. ${GREEN}E2E Test:${NC} ./web/test-e2e-websocket.sh"
echo -e "   - Runs a full end-to-end test with a live WebSocket connection"
echo -e "   - Verifies terminal content streaming and bidirectional communication"
echo -e ""
echo -e "2. ${GREEN}Terminal Visibility Test:${NC} web/test-terminal-visibility.js"
echo -e "   - Browser-based test for diagnosing terminal display issues"
echo -e "   - See web/test-terminal-visibility.txt for instructions"
echo ""

# Show whether to continue
echo -e "${YELLOW}Would you like to run the E2E test now? (y/n)${NC}"
read -n 1 -r REPLY
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
  exec ./web/test-e2e-websocket.sh
else
  echo -e "${BLUE}Skipping E2E test. Run it manually with:${NC}"
  echo -e "${GREEN}./web/test-e2e-websocket.sh${NC}"
fi