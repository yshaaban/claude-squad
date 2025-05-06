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
	// Create default loggers to stdout/stderr
	prefix := ""
	if daemon {
		prefix = "[DAEMON] "
	}
	
	InfoLog = log.New(os.Stdout, prefix+"INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLog = log.New(os.Stderr, prefix+"WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLog = log.New(os.Stderr, prefix+"ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	// Skip file logging unless explicitly enabled
	if !enableFileLogging {
		return
	}

	// Try to open the log file
	f, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		WarningLog.Printf("Could not open log file: %s (using stderr instead)", err)
		return
	}

	// Set up multi-writer to log to both file and stdout/stderr
	infoWriter := io.MultiWriter(os.Stdout, f)
	warnWriter := io.MultiWriter(os.Stderr, f)
	errorWriter := io.MultiWriter(os.Stderr, f)

	// Set log format to include timestamp and file/line number
	InfoLog = log.New(infoWriter, prefix+"INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLog = log.New(warnWriter, prefix+"WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLog = log.New(errorWriter, prefix+"ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)

	globalLogFile = f
}

func Close() {
	if globalLogFile != nil {
		_ = globalLogFile.Close()
		
		// Only print the log file location if file logging is enabled
		if enableFileLogging {
			fmt.Println("wrote logs to " + logFileName)
		}
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
