package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	WarningLog *log.Logger
	InfoLog    *log.Logger
	ErrorLog   *log.Logger
	
	// Special loggers that only log to file, never to console
	FileOnlyInfoLog    *log.Logger
	FileOnlyWarningLog *log.Logger
	FileOnlyErrorLog   *log.Logger
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
	prefix := ""
	if daemon {
		prefix = "[DAEMON] "
	}
	
	// Always set up console logging for terminal UI
	InfoLog = log.New(os.Stdout, prefix+"INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLog = log.New(os.Stderr, prefix+"WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLog = log.New(os.Stderr, prefix+"ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	
	// Set up file-only loggers to discard initially
	FileOnlyInfoLog = log.New(io.Discard, "", 0)
	FileOnlyWarningLog = log.New(io.Discard, "", 0)
	FileOnlyErrorLog = log.New(io.Discard, "", 0)

	if !enableFileLogging {
		return
	}
	
	// If file logging is enabled, set up file loggers
	f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		WarningLog.Printf("Could not open log file: %s (using stderr instead)", err)
		return
	}

	// Set up the file-only loggers that will never log to stdout/stderr
	// These are used for web server messages that should never appear in the terminal
	FileOnlyInfoLog = log.New(f, prefix+"WEB-INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	FileOnlyWarningLog = log.New(f, prefix+"WEB-WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	FileOnlyErrorLog = log.New(f, prefix+"WEB-ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Always log to both console and file for terminal UI
	InfoLog = log.New(io.MultiWriter(os.Stdout, f), prefix+"INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLog = log.New(io.MultiWriter(os.Stderr, f), prefix+"WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLog = log.New(io.MultiWriter(os.Stderr, f), prefix+"ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	
	globalLogFile = f
}

func Close() {
	if globalLogFile != nil {
		_ = globalLogFile.Close()
		
		// Print log file location when exiting the app
		// This helps users find logs, but only shows at the very end
		// to avoid interfering with terminal UI during operation
		fmt.Printf("\nLogs written to: %s\n", logFileName)
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
