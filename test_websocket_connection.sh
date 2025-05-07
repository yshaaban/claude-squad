#!/bin/bash

# Script to test WebSocket connections in Claude Squad
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Running WebSocket connection test..."

# Kill any existing Claude Squad processes
echo "Cleaning up any existing Claude Squad processes..."
pkill -f "claude-squad" || true
sleep 2

# Start Claude Squad in simple mode with web and React enabled
echo "Starting Claude Squad in simple mode with web and React enabled..."
./cs -s --web --react --log-to-file &
CS_PID=$!

# Give it time to start
echo "Waiting for server to start..."
sleep 5

# Verify the server is running
if ! ps -p $CS_PID > /dev/null; then
  echo "❌ ERROR: Claude Squad process died unexpectedly"
  cat /tmp/claudesquad.log | tail -50
  exit 1
fi

echo "✅ Claude Squad started successfully with PID $CS_PID"

# Test with curl to see if the server is responding
echo "Testing web server response..."
if curl -s http://localhost:8080 > /dev/null; then
  echo "✅ Web server is responding"
else
  echo "❌ ERROR: Web server is not responding"
  kill $CS_PID
  cat /tmp/claudesquad.log | tail -50
  exit 1
fi

# Check for WebSocket errors in the log file
echo "Checking for WebSocket errors in log..."
if grep -i "websocket.*error" /tmp/claudesquad.log; then
  echo "⚠️ WARNING: Found WebSocket errors in log file"
else
  echo "✅ No WebSocket errors found in logs"
fi

# Check for panic errors in the log file
echo "Checking for panic errors in log..."
if grep -i "panic" /tmp/claudesquad.log; then
  echo "⚠️ WARNING: Found panic errors in log file"
else
  echo "✅ No panic errors found in logs"
fi

# Kill the process
echo "Killing Claude Squad process..."
kill $CS_PID
wait $CS_PID 2>/dev/null || true

echo -e "\n=== WebSocket Connection Test Complete ==="
echo "The server started successfully without immediately crashing."
echo "For a complete test, please:"
echo "1. Run 'cs -s --web --react' manually"
echo "2. Open http://localhost:8080 in your browser"
echo "3. Create a new instance and test terminal functionality"
echo "4. Test resizing the terminal window"
echo "5. Try disconnecting and reconnecting"
echo "6. Leave the connection idle for 1-2 minutes to test heartbeats"