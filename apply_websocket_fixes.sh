#!/bin/bash

# Apply WebSocket fixes to improve terminal connection stability
echo "Applying WebSocket connection fixes..."

# Back up original files
echo "Backing up original files..."
cp -f frontend/src/components/terminal/Terminal.tsx frontend/src/components/terminal/Terminal.tsx.pre_ws_fix
cp -f frontend/src/components/terminal/Terminal.fixed.tsx frontend/src/components/terminal/Terminal.tsx

# Rebuild the application
echo "Rebuilding application with fixes..."
./build.sh

echo "WebSocket fixes have been applied!"
echo
echo "Key improvements:"
echo "1. Fixed terminal initialization and dimension handling"
echo "2. Improved WebSocket connection management"
echo "3. Enhanced reconnection logic"
echo
echo "For more details, see:"
echo "- WEBSOCKET_FIX.md - Original protocol fixes"
echo "- WEBSOCKET_FIX_UPDATE.md - Terminal initialization and dimension fixes"
echo
echo "To test the fixed implementation, run:"
echo "cs -s --web --react"