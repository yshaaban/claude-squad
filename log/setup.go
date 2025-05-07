package log

import (
	"io"
	"log"
	"os"
)

// SetupLogging initializes logging to a file
func SetupLogging(logFile string) {
	// Start with stdout/stderr logging
	InfoLog = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLog = log.New(os.Stderr, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLog = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	
	// Set up file logging
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		WarningLog.Printf("Could not open log file: %s (using stderr instead)", err)
		return
	}
	
	// Set file-only loggers
	FileOnlyInfoLog = log.New(f, "WEB-INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	FileOnlyWarningLog = log.New(f, "WEB-WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	FileOnlyErrorLog = log.New(f, "WEB-ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	
	// Set standard loggers to write to both console and file
	InfoLog = log.New(io.MultiWriter(os.Stdout, f), "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLog = log.New(io.MultiWriter(os.Stderr, f), "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLog = log.New(io.MultiWriter(os.Stderr, f), "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	
	globalLogFile = f
}