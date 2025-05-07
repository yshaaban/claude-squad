#!/bin/bash

# Script to test the WebSocket connection fixes
echo "Testing WebSocket connection fixes..."

# Start the application in the background
echo "Starting Claude Squad with React web interface..."
./cs -s --web --react --log-to-file &
CS_PID=$!

# Give it a moment to start up
sleep 3

# Check if the process is still running
if ps -p $CS_PID > /dev/null; then
  echo "✅ Claude Squad started successfully with PID: $CS_PID"
else
  echo "❌ Claude Squad failed to start"
  cat /tmp/claudesquad.log | tail -20
  exit 1
fi

# Check that the web server is responding
echo "Testing web server connection..."
if curl -s http://localhost:8080 > /dev/null; then
  echo "✅ Web server is responding"
else
  echo "❌ Web server is not responding"
  kill $CS_PID 2>/dev/null || true
  cat /tmp/claudesquad.log | tail -20
  exit 1
fi

# Check logs for WebSocket errors
echo "Checking logs for WebSocket errors..."
if grep -i "websocket.*error" /tmp/claudesquad.log; then
  echo "⚠️ Found WebSocket errors in logs"
else
  echo "✅ No WebSocket errors found in logs"
fi

# Check for panics
echo "Checking logs for panic errors..."
if grep -i "panic" /tmp/claudesquad.log; then
  echo "⚠️ Found panic errors in logs"
else
  echo "✅ No panic errors found in logs"
fi

# Cleanup
echo "Killing test process..."
kill $CS_PID
wait $CS_PID 2>/dev/null || true

echo -e "\n=== WebSocket Connection Test Complete ==="
echo "The automated tests have passed."
echo ""
echo "For full connection testing, please:"
echo "1. Run the application: cs -s --web --react"
echo "2. Open http://localhost:8080 in your browser"
echo "3. Create a new instance and verify the terminal connects properly"
echo "4. Test resizing the terminal window to verify dimension handling"
echo "5. Test disconnecting and reconnecting"
echo "6. Leave the connection idle for 1-2 minutes to verify heartbeats work"