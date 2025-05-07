#!/bin/bash
set -e

echo "Building and testing React UI..."

# Kill any existing claudesquad processes
pkill -f "cs.*--web" > /dev/null 2>&1 || true
sleep 1

# Clean up any temporary files
rm -f /tmp/claudesquad*.sock > /dev/null 2>&1 || true
rm -f /tmp/cs_*.sock > /dev/null 2>&1 || true

# Build the application
echo "Building the application..."
./build.sh

# Running application with React UI and keeping process in foreground
echo "Starting claudesquad with React UI..."
echo "Running: ./cs -s --web --web-port 8086 --react"

# Run in foreground so you can see output and errors
./cs -s --web --web-port 8086 --react

# This script will stay in foreground until the process is terminated