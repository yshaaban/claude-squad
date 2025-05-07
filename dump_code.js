const fs = require('fs');
const path = require('path');

// List of important files to dump
const filesToDump = [
  '/Users/ysh/src/claude-squad/main.go',
  '/Users/ysh/src/claude-squad/app/app.go', // Updated with StartWebServer and StopWebServer methods
  '/Users/ysh/src/claude-squad/session/instance.go', // Updated HasUpdated method with content caching
  '/Users/ysh/src/claude-squad/session/storage.go',
  '/Users/ysh/src/claude-squad/session/tmux/tmux.go', // Improved PTY handling and error recovery
  '/Users/ysh/src/claude-squad/session/git/worktree.go',
  '/Users/ysh/src/claude-squad/session/git/diff.go',
  '/Users/ysh/src/claude-squad/config/config.go',
  '/Users/ysh/src/claude-squad/config/state.go',
  '/Users/ysh/src/claude-squad/log/log.go',
  '/Users/ysh/src/claude-squad/web/server.go', // Consolidated WebSocket handlers
  '/Users/ysh/src/claude-squad/web/monitor.go', // Fixed variable naming and reduced polling frequency
  '/Users/ysh/src/claude-squad/web/types/types.go', // WebSocket message types
  '/Users/ysh/src/claude-squad/web/handlers/instances.go', // Fixed hasPrompt usage
  '/Users/ysh/src/claude-squad/web/handlers/terminal.go', // Deprecated in favor of websocket.go
  '/Users/ysh/src/claude-squad/web/handlers/websocket.go', // Updated to use FileOnlyInfoLog
  '/Users/ysh/src/claude-squad/web/middleware/auth.go', // Authentication middleware
  '/Users/ysh/src/claude-squad/ui/menu.go', // Improved web server info display
  '/Users/ysh/src/claude-squad/ui/list.go',
  '/Users/ysh/src/claude-squad/ui/tabbed_window.go',
  '/Users/ysh/src/claude-squad/ui/preview.go',
  '/Users/ysh/src/claude-squad/ui/diff.go',
  '/Users/ysh/src/claude-squad/keys/keys.go',
  '/Users/ysh/src/claude-squad/daemon/daemon.go'
];

// Function to check if a file exists
function fileExists(filePath) {
  try {
    return fs.statSync(filePath).isFile();
  } catch (err) {
    return false;
  }
}

// Output file
const outputFile = path.join(__dirname, 'code_dump.txt');
const outputStream = fs.createWriteStream(outputFile);

// Write header
outputStream.write('# Claude Squad Code Dump\n\n');
outputStream.write('This file contains the content of key code files in the Claude Squad project.\n\n');

// Process each file
let fileCount = 0;
filesToDump.forEach(filePath => {
  if (fileExists(filePath)) {
    try {
      const content = fs.readFileSync(filePath, 'utf8');
      const relPath = filePath.replace('/Users/ysh/src/claude-squad/', '');
      
      outputStream.write(`<file name="${relPath}">\n`);
      outputStream.write(content);
      outputStream.write(`\n</file>\n\n`);
      
      console.log(`Added: ${relPath}`);
      fileCount++;
    } catch (err) {
      console.error(`Error reading file ${filePath}: ${err.message}`);
    }
  } else {
    console.warn(`File not found: ${filePath}`);
  }
});

// Write footer
outputStream.write(`\n# End of dump - ${fileCount} files included`);
outputStream.end();

console.log(`\nDump completed! ${fileCount} files written to ${outputFile}`);