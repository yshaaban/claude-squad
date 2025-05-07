#!/bin/bash
# Script to fix terminal rendering issues while preserving web functionality

set -e  # Exit immediately if a command exits with non-zero status

echo "Fixing terminal rendering issues while preserving web functionality..."

# Fix main.go - restore terminal behavior but keep web flags
git checkout HEAD -- main.go
# Keep the web flag definitions but fix the logging setup
git apply << 'EOF'
diff --git a/main.go b/main.go
index 841c9d3..7fffe9a 100644
--- a/main.go
+++ b/main.go
@@ -35,10 +35,6 @@ var (
 			// Enable file logging if requested
 			if fileLoggingFlag {
 				log.EnableFileLogging()
-				
-				// When web monitoring is enabled, keep console logging for terminal UI
-				if webMonitoringFlag {
-					log.SetConsoleLoggingDisabled(false)
-				}
 			}
 			
 			log.Initialize(daemonFlag)
EOF

# Fix app/app.go - restore terminal rendering but keep web server initialization
git checkout HEAD -- app/app.go
# Fix the NoTTY implementation that breaks terminal rendering
git apply << 'EOF'
diff --git a/app/app.go b/app/app.go
index 5caeda1..0f26e9a 100644
--- a/app/app.go
+++ b/app/app.go
@@ -190,17 +190,16 @@ func newHome(ctx context.Context, startOptions StartOptions) *home {
 			Program:   startOptions.Program,
 			AutoYes:   true,
 			InPlace:   true,
-			NoTTY:     startOptions.NoTTY,
 		})
-		
+
 		// Do not auto-send empty prompts - always show prompt dialog
 		h.state = statePrompt
 		h.menu.SetState(ui.StatePrompt)
 		h.textInputOverlay = overlay.NewTextInputOverlay("Enter prompt", "")
 
 		// If web server is enabled, start it in a goroutine
-		if startOptions.WebServerEnabled { 
-		   go h.StartWebServer()
+		if startOptions.WebServerEnabled {
+			go h.StartWebServer()
 		}
 	} else {
 		// Standard mode behavior remains unchanged
EOF

# Fix log/log.go - keep FileOnlyLog loggers but restore console output
git checkout HEAD -- log/log.go
# Fix the logging setup to never disable console output for terminal
git apply << 'EOF'
diff --git a/log/log.go b/log/log.go
index 5f3aa7c..6fcbf86 100644
--- a/log/log.go
+++ b/log/log.go
@@ -13,55 +13,64 @@ var (
 	WarningLog *log.Logger
 	InfoLog    *log.Logger
 	ErrorLog   *log.Logger
+	
+	// Special loggers that only log to file, never to console
+	FileOnlyInfoLog    *log.Logger
+	FileOnlyWarningLog *log.Logger
+	FileOnlyErrorLog   *log.Logger
 )
 
 var logFileName = filepath.Join(os.TempDir(), "claudesquad.log")
 
 var globalLogFile *os.File
 var enableFileLogging = false // Disabled by default
-var disableConsoleLogging = false // Console logging always enabled for terminal
 
 // EnableFileLogging enables logging to a file
 func EnableFileLogging() {
 	enableFileLogging = true
 }
 
-// Initialize should be called once at the beginning of the program to set up logging.
-// defer Close() after calling this function. 
-// By default, logs only go to stdout/stderr. Set enableFileLogging to true to also write to a file.
-
 func Initialize(daemon bool) {
-	// Create default loggers to stdout/stderr
 	prefix := ""
 	if daemon {
 		prefix = "[DAEMON] "
 	}
 	
+	// Normal console logging - always enabled for terminal UI
 	InfoLog = log.New(os.Stdout, prefix+"INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
 	WarningLog = log.New(os.Stderr, prefix+"WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
 	ErrorLog = log.New(os.Stderr, prefix+"ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
+	
+	// Set up file-only loggers to discard initially
+	FileOnlyInfoLog = log.New(io.Discard, "", 0)
+	FileOnlyWarningLog = log.New(io.Discard, "", 0)
+	FileOnlyErrorLog = log.New(io.Discard, "", 0)
 
-	// Skip file logging unless explicitly enabled
 	if !enableFileLogging {
 		return
 	}
-
-	// Try to open the log file
+	
+	// If file logging is enabled, set up file loggers
 	f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
 	if err != nil {
 		WarningLog.Printf("Could not open log file: %s (using stderr instead)", err)
 		return
 	}
 
+	// Set up the file-only loggers that will never log to stdout/stderr
+	FileOnlyInfoLog = log.New(f, prefix+"WEB-INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
+	FileOnlyWarningLog = log.New(f, prefix+"WEB-WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
+	FileOnlyErrorLog = log.New(f, prefix+"WEB-ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
+
 	// Set up multi-writer to log to both file and stdout/stderr
 	infoWriter := io.MultiWriter(os.Stdout, f)
 	warnWriter := io.MultiWriter(os.Stderr, f)
 	errorWriter := io.MultiWriter(os.Stderr, f)
 
-	// Set log format to include timestamp and file/line number
+	// Always log to both console and file for terminal UI
 	InfoLog = log.New(infoWriter, prefix+"INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
 	WarningLog = log.New(warnWriter, prefix+"WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
 	ErrorLog = log.New(errorWriter, prefix+"ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
-
+	
 	globalLogFile = f
 }
EOF

# Fix session/tmux/tmux.go - remove NoTTY mode completely
git checkout HEAD -- session/tmux/tmux.go

# Fix session/instance.go - remove NoTTY flag but keep other improvements
git checkout HEAD -- session/instance.go
# Keep InPlace flag and other useful changes
git apply << 'EOF'
diff --git a/session/instance.go b/session/instance.go
index 334cab3..d1c7b9e 100644
--- a/session/instance.go
+++ b/session/instance.go
@@ -53,8 +53,6 @@ type Instance struct {
 	Prompt string
 	// InPlace is true if the instance should run in the current directory without creating a worktree
 	InPlace bool
-	// NoTTY removed - always use terminal
-
 	// DiffStats stores the current git diff statistics
 	diffStats *git.DiffStats
 
@@ -81,7 +79,6 @@ func (i *Instance) ToInstanceData() InstanceData {
 		Program:   i.Program,
 		AutoYes:   i.AutoYes,
 		InPlace:   i.InPlace,
-		// NoTTY removed
 	}
 
 	// Only include worktree data if gitWorktree is initialized
EOF

# Fix web/handlers files with safe web monitoring
git checkout HEAD -- web/handlers/instances.go
git checkout HEAD -- web/handlers/websocket.go

# Fix app/web.go - use proper logging for web server
git checkout HEAD -- app/web.go
git apply << 'EOF'
diff --git a/app/web.go b/app/web.go
index 68fc9f4..63ece24 100644
--- a/app/web.go
+++ b/app/web.go
@@ -16,7 +16,6 @@ type StartOptions struct {
 	SimpleMode       bool
 	WebServerEnabled bool
 	WebServerPort    int
-	NoTTY            bool // removed, always use terminal
 }
 
 // StartWebServer initializes and starts the web monitoring server.
@@ -32,7 +31,8 @@ func (h *home) StartWebServer() error {
 		return err
 	}
 
-	log.FileOnlyInfoLog.Printf("Web monitoring server started on %s:%d", 
+	// Use FileOnlyLog to avoid corrupting terminal UI
+	log.FileOnlyInfoLog.Printf("Web monitoring server started on http://%s:%d", 
 		h.appConfig.WebServerHost, h.appConfig.WebServerPort)
 	
 	return nil
EOF

# Remove all rejected merge files
rm -f app/app.go.orig app/app.go.patch app/app.go.rej
rm -f web/server.go.orig web/server.go.patch web/server.go.rej
rm -f web/monitor.go.bak web/monitor.go.patch

echo "Rebuilding application..."
go build -o cs

echo "Done! Terminal rendering issues should be fixed."
echo ""
echo "You can now run 'cs -s --web' to use Simple Mode with working"
echo "terminal rendering and a complementary web UI."