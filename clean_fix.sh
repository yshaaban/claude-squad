#!/bin/bash

# Clean fix to restore WebSocket state and apply only the Terminal.tsx changes
echo "Applying clean WebSocket connection fixes..."

# Back up original Terminal.tsx file if it doesn't exist yet
if [ ! -f frontend/src/components/terminal/Terminal.tsx.original ]; then
    echo "Backing up original Terminal.tsx file..."
    cp -f frontend/src/components/terminal/Terminal.tsx frontend/src/components/terminal/Terminal.tsx.original
fi

# Restore original websocket.go but keep the fixed Terminal.tsx
echo "Restoring original websocket.go file..."
cp -f /Users/ysh/src/claude-squad/web/handlers/websocket.go.backup /Users/ysh/src/claude-squad/web/handlers/websocket.go

# Build with the changes
echo "Rebuilding application with client-side fixes..."
./build.sh

echo "Client-side WebSocket fixes have been applied!"
echo
echo "Key improvements:"
echo "1. Fixed terminal recreation on reconnection"
echo "2. Enhanced terminal initialization sequence"
echo "3. Improved terminal dimension handling"
echo "4. Proper cleanup on component unmount"
echo
echo "For more details, see WEBSOCKET_FIX_UPDATE.md"
echo
echo "To test the fixed implementation, run:"
echo "cs -s --web --react"