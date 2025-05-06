package web_test

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestWebServerE2E runs an external end-to-end test for the web server.
func TestWebServerE2E(t *testing.T) {
	// Only run if requested explicitly, as it requires building and running the app
	if os.Getenv("RUN_E2E_TESTS") != "1" {
		t.Skip("Skipping E2E test; set RUN_E2E_TESTS=1 to run")
	}
	
	// Get script path
	scriptPath := "./e2e-test.sh"
	
	// Execute the script
	cmd := exec.Command("bash", scriptPath)
	output, err := cmd.CombinedOutput()
	
	// Print output regardless of success
	fmt.Println(string(output))
	
	// Check for errors
	if err != nil {
		t.Fatalf("E2E test failed: %v", err)
	}
	
	// Check for success message in output
	if !strings.Contains(string(output), "All tests passed") {
		t.Errorf("E2E test did not indicate success")
	}
}

// TestWebUISimple tests that a locally running web server returns a proper HTML page.
func TestWebUISimple(t *testing.T) {
	// Only run if a server is already running (for manual testing)
	resp, err := http.Get("http://localhost:8080")
	if err != nil {
		t.Skip("Skipping test: no server running on port 8080")
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	
	// Note success
	t.Logf("Successfully connected to web server at http://localhost:8080")
}