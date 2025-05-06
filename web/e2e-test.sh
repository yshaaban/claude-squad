#!/bin/bash
set -e

# End-to-end test for the Claude Squad web monitoring server
echo "========================================"
echo "Claude Squad Web Monitoring Server Test"
echo "========================================"

# Test directory
TEST_DIR=$(mktemp -d)
echo "Using test directory: $TEST_DIR"
cd "$TEST_DIR"

# Create a test git repository
echo "Setting up test git repository..."
git init
git config user.email "test@example.com"
git config user.name "Test User"
echo "# Test Repository" > README.md
git add README.md
git commit -m "Initial commit"

# Build Claude Squad with web server enabled
echo "Building Claude Squad..."
go build -o cs

# Start Claude Squad with web monitoring in the background
echo "Starting Claude Squad with web monitoring..."
./cs --web --web-port=8099 -s &
CS_PID=$!

# Wait for server to start
echo "Waiting for server to start..."
sleep 2

# Test API endpoints
echo "Testing API endpoints..."

# Test /api/instances endpoint
echo "Testing /api/instances..."
INSTANCES_RESPONSE=$(curl -s http://localhost:8099/api/instances)
echo "Response: $INSTANCES_RESPONSE"

# Check if we have instances
if [[ "$INSTANCES_RESPONSE" == *"instances"* ]]; then
  echo "✅ /api/instances endpoint working"
else
  echo "❌ /api/instances endpoint failed"
  kill $CS_PID
  exit 1
fi

# Test web UI
echo "Testing web UI..."
WEB_UI_RESPONSE=$(curl -s http://localhost:8099/)
if [[ "$WEB_UI_RESPONSE" == *"Claude Squad Monitor"* ]]; then
  echo "✅ Web UI is working"
else
  echo "❌ Web UI failed"
  kill $CS_PID
  exit 1
fi

# Kill Claude Squad
echo "Cleaning up..."
kill $CS_PID

echo "========================================"
echo "All tests passed! Web server is working."
echo "========================================"

# Clean up
rm -rf "$TEST_DIR"