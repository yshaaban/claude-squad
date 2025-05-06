package debugtools

import (
	"fmt"
	"os"
	"time"
)

// Debug logs a message to a debug file
func Debug(format string, args ...interface{}) {
	f, err := os.OpenFile("/tmp/cs_debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	
	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("15:04:05.000")
	
	fmt.Fprintf(f, "[%s] %s\n", timestamp, msg)
}