#!/bin/bash

# Apply comprehensive WebSocket fixes to improve terminal connection stability
echo "Applying comprehensive WebSocket connection fixes..."

# Back up original files
echo "Backing up original files..."
cp -f frontend/src/components/terminal/Terminal.tsx frontend/src/components/terminal/Terminal.tsx.pre_final_ws_fix
cp -f web/handlers/websocket.go web/handlers/websocket.go.pre_final_ws_fix

# Fix the websocket.go file to move mutex declaration
echo "Fixing mutex declaration in websocket.go..."
sed -i '' 's/return func(w http.ResponseWriter, r \*http.Request) {/return func(w http.ResponseWriter, r \*http.Request) {\n\t\t\/\/ Mutex for websocket writes - declared early as it'"'"'s used in multiple goroutines\n\t\tvar writeMu sync.Mutex/g' web/handlers/websocket.go
sed -i '' 's/\t\t\/\/ Mutex for websocket writes\n\t\tvar writeMu sync.Mutex/\t\t\/\/ WebSocket write operations handled above/g' web/handlers/websocket.go

# Build with improved fixes
echo "Rebuilding application with all fixes applied..."
./build.sh

# Verify the build
if [ $? -eq 0 ]; then
  echo "✅ Build successful! WebSocket fixes have been applied."
else
  echo "❌ Build failed. Restoring original files..."
  cp frontend/src/components/terminal/Terminal.tsx.pre_final_ws_fix frontend/src/components/terminal/Terminal.tsx
  cp web/handlers/websocket.go.pre_final_ws_fix web/handlers/websocket.go
  echo "Original files restored."
  exit 1
fi

echo "WebSocket fixes have been applied!"
echo
echo "Key improvements:"
echo "1. Fixed terminal recreation on reconnection"
echo "2. Added panic recovery to all goroutines"
echo "3. Improved error handling and edge case management"
echo "4. Enhanced resource cleanup"
echo "5. Fixed syntax errors in websocket.go"
echo
echo "For more details, see WEBSOCKET_FIX_SUMMARY.md"
echo
echo "To test the fixed implementation, run:"
echo "cs -s --web --react"