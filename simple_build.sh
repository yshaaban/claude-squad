#!/bin/bash
# Super simple build script for iTerm2

echo "Building Claude Squad..."
go build -o cs
echo "Done. Binary created at $(pwd)/cs"
echo "Run with: ./cs -s --web"