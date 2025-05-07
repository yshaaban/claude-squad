#!/bin/bash

# Clean fix to apply WebSocket fixes to Terminal.tsx while keeping websocket.go unchanged
echo "Applying clean WebSocket connection fixes..."

# Back up original Terminal.tsx file if it doesn't exist yet
if [ ! -f frontend/src/components/terminal/Terminal.tsx.original ]; then
    echo "Backing up original Terminal.tsx file..."
    cp -f frontend/src/components/terminal/Terminal.tsx frontend/src/components/terminal/Terminal.tsx.original
fi

# Copy the fixed version of Terminal.tsx if it exists
if [ -f frontend/src/components/terminal/Terminal.fixed.tsx ]; then
    echo "Applying Terminal.tsx fix from Terminal.fixed.tsx..."
    cp -f frontend/src/components/terminal/Terminal.fixed.tsx frontend/src/components/terminal/Terminal.tsx
    echo "✅ Applied Terminal.tsx fixes"
else
    echo "WARNING: Terminal.fixed.tsx not found. Using existing Terminal.tsx."
fi

# Build with the changes
echo "Rebuilding application with client-side fixes..."
./build.sh

if [ $? -eq 0 ]; then
    echo "✅ Build successful! WebSocket fixes have been applied."
else
    echo "❌ Build failed. Restoring original files..."
    if [ -f frontend/src/components/terminal/Terminal.tsx.original ]; then
        cp -f frontend/src/components/terminal/Terminal.tsx.original frontend/src/components/terminal/Terminal.tsx
    fi
    echo "Original files restored."
    exit 1
fi

echo "Client-side WebSocket fixes have been applied!"
echo
echo "Key improvements:"
echo "1. Fixed terminal recreation on reconnection"
echo "2. Enhanced terminal initialization sequence"
echo "3. Improved terminal dimension handling"
echo "4. Proper cleanup on component unmount"
echo
echo "For more details, see WEBSOCKET_FIX_SUMMARY.md"
echo
echo "To test the fixed implementation, run:"
echo "cs -s --web --react"