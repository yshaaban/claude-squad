#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Claude Squad Improved Terminal Launcher ===${NC}"
echo -e "${YELLOW}This script runs Claude Squad with WebSocket optimizations${NC}"

# Kill any existing instances first to avoid conflicts
echo -e "${YELLOW}Stopping any existing instances...${NC}"
pkill -f "cs.*--web" > /dev/null 2>&1 || true
sleep 1

# Clean up temporary sockets and other potential resource issues
echo -e "${YELLOW}Cleaning up any stale resources...${NC}"
rm -f /tmp/claudesquad*.sock > /dev/null 2>&1 || true
rm -f /tmp/cs_*.sock > /dev/null 2>&1 || true

# Set environment variables to ensure proper terminal handling
export TERM=xterm-256color
export NODE_ENV=production  # Using production mode for stability
export FORCE_COLOR=1
export MAX_WS_CONNECTIONS=10  # Limit WebSocket connections

# Ask user which mode to use but default to standard web UI
echo -e "${BLUE}Select terminal interface to use:${NC}"
echo -e "${YELLOW}1) Standard Web UI (recommended, more stable)${NC}"
echo -e "${YELLOW}2) React UI with improved WebSocket handling (experimental)${NC}"
read -t 10 -p "Enter choice [1-2, default=1]: " ui_choice || ui_choice=1

# Default to option 1 (standard UI) if no input
ui_choice=${ui_choice:-1}

case $ui_choice in
  2)
    echo -e "${YELLOW}Starting Claude Squad with experimental React UI...${NC}"
    echo -e "${YELLOW}Warning: React UI is less stable than standard web UI${NC}"
    
    # Run with React and optimized WebSocket settings
    ./cs -s --web --web-port 8086 --react --log-to-file &
    ;;
    
  1|*)
    echo -e "${GREEN}Starting Claude Squad with standard web UI...${NC}"
    echo -e "${GREEN}Using standard web interface for better compatibility${NC}"
    
    # Run the command directly without React flag to use the simpler web UI
    ./cs -s --web --web-port 8086 --log-to-file > /dev/null 2>&1 &
    ;;
esac

# Store the PID
PID=$!
echo $PID > /tmp/claudesquad_pid.txt

echo -e "${GREEN}Claude Squad started with PID ${PID}${NC}"
echo -e "${GREEN}Web UI available at: http://localhost:8086/${NC}"
echo -e "${GREEN}Monitor session with: tail -f /var/folders/*/T/claudesquad.log${NC}"
echo -e "${GREEN}Stop with: kill ${PID}${NC}"
echo ""
echo -e "${YELLOW}=== Connection Troubleshooting ===${NC}"
echo -e "If you encounter connection issues:"
echo -e "1. Try using Chrome or Edge instead of Safari"
echo -e "2. Only open one terminal tab at a time"
echo -e "3. Refresh page if terminal becomes unresponsive" 
echo -e "4. Close other browser tabs to free up WebSocket connections"
echo -e ""

if [ "$ui_choice" == "2" ]; then
  echo -e "${BLUE}The React UI includes these improvements:${NC}"
  echo -e "- Better connection handling and resource cleanup"
  echo -e "- Enhanced terminal dimensions initialization"
  echo -e "- Improved reconnection with exponential backoff"
  echo -e "- More robust WebSocket resource management"
  echo -e ""
  echo -e "${YELLOW}But may still encounter occasional stability issues${NC}"
else
  echo -e "${GREEN}The standard web UI provides better stability and compatibility${NC}"
  echo -e "- Simple and reliable terminal connection"
  echo -e "- More efficient resource usage"
  echo -e "- Better compatibility with all browsers"
  echo -e ""
  echo -e "${BLUE}Use this as your default unless you need React UI features${NC}"
fi
