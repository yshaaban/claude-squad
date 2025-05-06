#!/bin/bash
set -e

# Print header
echo "============================="
echo "Claude Squad Web Server Tests"
echo "============================="

# Get script directory 
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
REPO_ROOT="$(dirname "$DIR")"

# Go to repo root
cd "$REPO_ROOT"

# Make script executable
chmod +x "$DIR/test_integration.sh"

# Run integration tests
echo "Running integration tests..."
go test -v ./web/integration/...

# Return to original directory
echo "============================="
echo "All tests completed successfully!"
echo "============================="