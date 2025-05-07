#!/bin/bash
set -e

# End-to-end test for the Claude Squad terminal streaming functionality
echo "========================================"
echo "Claude Squad Terminal Streaming E2E Test"
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

# Create a temporary modified log.go that only logs to file when file logging is enabled
create_file_only_logger() {
  TEMP_LOG_GO="/tmp/modified_log.go"
  cat > "$TEMP_LOG_GO" << 'EOL'
package log

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	WarningLog *log.Logger
	InfoLog    *log.Logger
	ErrorLog   *log.Logger
)

var logFileName = filepath.Join(os.TempDir(), "claudesquad.log")

var globalLogFile *os.File
var enableFileLogging = false // Disabled by default

// EnableFileLogging enables logging to a file
func EnableFileLogging() {
	enableFileLogging = true
}

// Initialize should be called once at the beginning of the program to set up logging.
// defer Close() after calling this function. 
// By default, logs only go to stdout/stderr. Set enableFileLogging to true to also write to a file.

func Initialize(daemon bool) {
	// Create default loggers to stdout/stderr if file logging is not enabled
	prefix := ""
	if daemon {
		prefix = "[DAEMON] "
	}
	
	if !enableFileLogging {
		// Log to console only if file logging is not enabled
		InfoLog = log.New(os.Stdout, prefix+"INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
		WarningLog = log.New(os.Stderr, prefix+"WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
		ErrorLog = log.New(os.Stderr, prefix+"ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
		return
	}
	
	// If file logging is enabled, create loggers that only write to the file
	// Try to open the log file
	f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		// Fallback to stderr only for errors
		WarningLog = log.New(os.Stderr, prefix+"WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
		ErrorLog = log.New(os.Stderr, prefix+"ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
		InfoLog = log.New(ioutil.Discard, "", 0) // Discard info logs
		
		WarningLog.Printf("Could not open log file: %s (using stderr for errors only)", err)
		return
	}

	// Create loggers that ONLY write to the file, not to stdout/stderr
	InfoLog = log.New(f, prefix+"INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLog = log.New(f, prefix+"WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLog = log.New(f, prefix+"ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	
	globalLogFile = f
}

func Close() {
	if globalLogFile != nil {
		_ = globalLogFile.Close()
		
		// Don't print anything to stdout when file logging is enabled
		// This prevents log messages from interfering with full-screen UI rendering
		// IMPORTANT: The log file location message has been removed intentionally
	}
}

// Every is used to log at most once every timeout duration.
type Every struct {
	timeout time.Duration
	timer   *time.Timer
}

func NewEvery(timeout time.Duration) *Every {
	return &Every{timeout: timeout}
}

// ShouldLog returns true if the timeout has passed since the last log.
func (e *Every) ShouldLog() bool {
	if e.timer == nil {
		e.timer = time.NewTimer(e.timeout)
		e.timer.Reset(e.timeout)
		return true
	}

	select {
	case <-e.timer.C:
		e.timer.Reset(e.timeout)
		return true
	default:
		return false
	}
}
EOL
}

# Build Claude Squad with web server enabled and custom logger
echo "Building Claude Squad with modified logger..."
cd /Users/ysh/src/claude-squad

# Create the modified logger that only logs to file
create_file_only_logger

# Backup original log.go
if [ -f log/log.go ]; then
  cp log/log.go log/log.go.backup
  # Replace with our modified version
  cp /tmp/modified_log.go log/log.go
  echo "Replaced log.go with file-only version"
fi

# Build with the modified logger
go build -o cs

# Restore original log.go
if [ -f log/log.go.backup ]; then
  mv log/log.go.backup log/log.go
  echo "Restored original log.go"
fi

# Copy the built binary
cp cs "$TEST_DIR/cs"
cd "$TEST_DIR"

# Create a log file
LOG_FILE="$TEST_DIR/test-log.txt"
touch "$LOG_FILE"
echo "Logging to: $LOG_FILE"

# Special function that writes to log file only, not to console
log_file() {
  echo "$1" >> "$LOG_FILE"
}

# Log system info (to file only)
log_file "=== SYSTEM INFO ==="
log_file "$(uname -a)"
log_file "$(date)"
log_file ""

# Start Claude Squad with web monitoring in the background (with file logging enabled)
echo "Starting Claude Squad with web monitoring and file logging..."
log_file "Starting Claude Squad with web monitoring and file logging..."
# Note: --log-to-file means logs go ONLY to file, not to stdout/stderr with our modified logger
# Add --no-tty flag to prevent TTY detection
# Redirect both stdout and stderr to /dev/null to avoid displaying any error or log messages
./cs --web --web-port=8099 -s --log-to-file --no-tty 
CS_PID=$!
log_file "Claude Squad server PID: $CS_PID"

# Wait for server to start
echo "Waiting for server to start..."
sleep 2

# Create a test instance
echo "Creating a test instance..."
mkdir -p test-instance
cd test-instance
echo "# Test Project" > README.md
echo "console.log('Hello world');" > index.js
git init
git add .
git commit -m "Initial commit"
cd ..

# Launch a Claude instance in simple mode
echo "Launching Claude instance in simple mode..."
log_file "Launching Claude instance in simple mode..."
cd test-instance
log_file "Current directory: $(pwd)"
log_file "$(ls -la)"
log_file "Starting simple mode instance..."
# Add --no-tty flag to prevent TTY detection
# Redirect both stdout and stderr to /dev/null to avoid displaying any error or log messages
../cs -s --log-to-file --no-tty >/dev/null &
INSTANCE_PID=$!
log_file "Simple mode instance PID: $INSTANCE_PID"
cd ..

# Wait for instance to initialize
echo "Waiting for instance to initialize..."
sleep 5

# Test API endpoints to verify instance exists
echo "Testing API endpoints..."
log_file "=== TESTING API ENDPOINTS ==="
log_file "Testing API endpoints..."

# Get running processes for debugging
log_file "Running processes:"
log_file "$(ps aux | grep cs)"
log_file ""

# Test server status
log_file "Testing server status..."
STATUS_RESPONSE=$(curl -s http://localhost:8099/api/status)
log_file "Status response: $STATUS_RESPONSE"

# Test instances endpoint
log_file "Testing instances API..."
INSTANCES_RESPONSE=$(curl -s http://localhost:8099/api/instances)
log_file "Instances response: $INSTANCES_RESPONSE"

# Check log file for debugging
log_file "Checking log file..."
if [ -f "/tmp/claudesquad.log" ]; then
    log_file "Found log file at /tmp/claudesquad.log"
    log_file "$(tail -50 /tmp/claudesquad.log)"
else
    log_file "Log file not found at /tmp/claudesquad.log"
fi

# Check if we have instances
if [[ "$INSTANCES_RESPONSE" == *"instances"* ]]; then
  echo "✅ Instances found in API response"
  log_file "✅ Instances found in API response"
  
  # Extract the first instance title
  INSTANCE_TITLE=$(echo "$INSTANCES_RESPONSE" | grep -o '"title":"[^"]*"' | head -1 | cut -d'"' -f4)
  
  if [ -z "$INSTANCE_TITLE" ]; then
    echo "❌ Failed to extract instance title from response"
    log_file "❌ Failed to extract instance title from response"
    log_file "Killing processes: $INSTANCE_PID, $CS_PID"
    
    # More debugging info
    log_file "Final process state:"
    log_file "$(ps aux | grep -E "$INSTANCE_PID|$CS_PID")"
    
    kill $INSTANCE_PID
    kill $CS_PID
    exit 1
  fi
  
  echo "Found instance: $INSTANCE_TITLE"
  log_file "Found instance: $INSTANCE_TITLE"
  
  # Get specific instance details
  log_file "Getting instance details..."
  INSTANCE_DETAILS=$(curl -s "http://localhost:8099/api/instances/$INSTANCE_TITLE")
  log_file "Instance details: $INSTANCE_DETAILS"
else
  echo "❌ No instances found in API response"
  log_file "❌ No instances found in API response"
  
  log_file "Checking for any processes..."
  log_file "$(ps aux | grep cs)"
  
  log_file "Killing processes: $INSTANCE_PID, $CS_PID"
  kill $INSTANCE_PID
  kill $CS_PID
  exit 1
fi

# Test WebSocket terminal streaming
echo "Testing WebSocket terminal streaming..."

# Create a simple WebSocket client in Node.js
cat << 'EOF' > ws-test.js
const WebSocket = require('ws');
const fs = require('fs');

// Get instance title from command line args
const instanceTitle = process.argv[2];
console.log(`Using instance title: ${instanceTitle}`);

// Connect to WebSocket server
const ws = new WebSocket(`ws://localhost:8099/ws/terminal/${instanceTitle}?format=ansi&privileges=read-only`);

let messageCount = 0;
let hasContent = false;
let contentLength = 0;

// Log file for received messages
const logFile = fs.createWriteStream('ws-messages.log');

ws.on('open', function open() {
  console.log('Connected to terminal WebSocket');
  logFile.write('=== WebSocket Connection Opened ===\n');
  
  // Close the connection after 10 seconds
  setTimeout(() => {
    ws.close();
  }, 10000);
});

ws.on('message', function incoming(data) {
  messageCount++;
  
  try {
    const message = JSON.parse(data);
    logFile.write(`\n=== Message #${messageCount} ===\n`);
    logFile.write(JSON.stringify(message, null, 2) + '\n');
    
    // Check if we have content
    if (message.content && message.content.length > 0) {
      hasContent = true;
      contentLength += message.content.length;
      console.log(`Received content in message #${messageCount}, length: ${message.content.length}`);
    } else if (message.type === 'config') {
      console.log(`Received config message: ${JSON.stringify(message)}`);
    }
  } catch (e) {
    console.error('Error parsing message:', e);
    logFile.write(`\n=== Invalid Message #${messageCount} ===\n`);
    logFile.write(data + '\n');
  }
});

ws.on('close', function close() {
  console.log(`WebSocket connection closed. Received ${messageCount} messages.`);
  console.log(`Content received: ${hasContent ? 'Yes' : 'No'}, Total length: ${contentLength}`);
  
  logFile.write('\n=== WebSocket Connection Closed ===\n');
  logFile.write(`Total messages: ${messageCount}\n`);
  logFile.write(`Content received: ${hasContent ? 'Yes' : 'No'}, Total length: ${contentLength}\n`);
  logFile.end();
  
  // Write results to a file for the shell script to check
  fs.writeFileSync('ws-results.json', JSON.stringify({
    messageCount,
    hasContent,
    contentLength
  }));
  
  process.exit(hasContent ? 0 : 1);
});

ws.on('error', function error(err) {
  console.error('WebSocket error:', err);
  logFile.write(`\n=== WebSocket Error ===\n`);
  logFile.write(err.toString() + '\n');
  process.exit(1);
});
EOF

# Install ws if not already installed
if ! command -v npm &> /dev/null; then
  echo "❌ npm not found, skipping WebSocket test"
else
  echo "Installing WebSocket client dependencies..."
  npm install ws
  
  echo "Running WebSocket client test for instance '$INSTANCE_TITLE'..."
  node ws-test.js "$INSTANCE_TITLE"
  
  # Wait for client to complete
  sleep 22
  
  # Check results
  if [ -f ws-results.json ]; then
    MESSAGE_COUNT=$(grep -o '"messageCount":[0-9]*' ws-results.json | cut -d: -f2)
    HAS_CONTENT=$(grep -o '"hasContent":\w*' ws-results.json | cut -d: -f2)
    
    echo "WebSocket test completed with $MESSAGE_COUNT messages"
    
    if [ "$HAS_CONTENT" == "true" ]; then
      echo "✅ Terminal content received successfully"
    else
      echo "❌ No terminal content received"
      cat ws-messages.log
      kill $INSTANCE_PID
      kill $CS_PID
      exit 1
    fi
  else
    echo "❌ WebSocket test failed, no results file found"
    kill $INSTANCE_PID
    kill $CS_PID
    exit 1
  fi
fi

# Test terminal content visibility in web UI
echo "Testing terminal content visibility in web UI..."
WEB_UI_RESPONSE=$(curl -s http://localhost:8099/)

if [[ "$WEB_UI_RESPONSE" == *"terminal-container"* ]]; then
  echo "✅ Terminal container found in web UI"
else
  echo "❌ Terminal container not found in web UI"
  kill $INSTANCE_PID
  kill $CS_PID
  exit 1
fi

# Clean up
echo "Cleaning up..."
log_file "Cleaning up..."
log_file "Killing process $INSTANCE_PID..."
kill $INSTANCE_PID || true
log_file "Killing process $CS_PID..."
kill $CS_PID || true

# Copy log files to the home directory for future reference
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
LOG_DEST="$HOME/cs_test_logs_${TIMESTAMP}"
mkdir -p "$LOG_DEST"

log_file "Saving logs to $LOG_DEST"
echo "Saving logs to $LOG_DEST"
cp "$LOG_FILE" "$LOG_DEST/"
if [ -f "/tmp/claudesquad.log" ]; then
    cp "/tmp/claudesquad.log" "$LOG_DEST/"
fi
if [ -f "ws-messages.log" ]; then
    cp "ws-messages.log" "$LOG_DEST/"
fi
if [ -f "ws-results.json" ]; then
    cp "ws-results.json" "$LOG_DEST/"
fi

echo "========================================"
echo "Test completed. Logs saved to $LOG_DEST"
echo "========================================"

# Check if we want to clean up
if [ "${KEEP_TEST_DIR:-no}" != "yes" ]; then
    echo "Removing test directory: $TEST_DIR"
    rm -rf "$TEST_DIR"
else
    echo "Keeping test directory: $TEST_DIR"
fi
