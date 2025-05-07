#!/bin/bash
# iTerm2-compatible script for Claude Squad

# Use non-fancy output to avoid escape sequence issues
echo "Starting Claude Squad for iTerm2..."

# Kill any existing instances first to avoid conflicts
echo "Stopping any existing instances..."
pkill -f "cs.*--web" > /dev/null 2>&1 || true
sleep 1

# Clean up temporary sockets and other potential resource issues
echo "Cleaning up any stale resources..."
rm -f /tmp/claudesquad*.sock > /dev/null 2>&1 || true
rm -f /tmp/cs_*.sock > /dev/null 2>&1 || true

# Set environment variables to ensure proper terminal handling
export TERM=xterm-256color
export NODE_ENV=production
export FORCE_COLOR=1

# Use standard web UI for iTerm2 compatibility
echo "Starting standard web UI (recommended for iTerm2)..."

# Run the command directly without React flag to use the simpler web UI
./cs -s --web --web-port 8086 --log-to-file > /dev/null 2>&1 &

# Store the PID
PID=$!
echo $PID > /tmp/claudesquad_pid.txt

echo "Claude Squad started with PID ${PID}"
echo "Web UI available at: http://localhost:8086/"
echo "Monitor logs with: tail -f /var/folders/*/T/claudesquad.log"
echo "Stop with: kill ${PID}"
echo ""
echo "iTerm2 Compatibility Notes:"
echo "- Using standard web interface for best compatibility"
echo "- Avoid escape sequences that can confuse iTerm2"
echo "- If you encounter errors, try the web browser directly"
echo ""
echo "Troubleshooting Tips:"
echo "1. Use Chrome or Firefox instead of Safari"
echo "2. Only open one terminal tab at a time"
echo "3. Refresh page if terminal becomes unresponsive"
echo "4. Close other browser tabs to free up connections"