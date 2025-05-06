package test

import (
	"claude-squad/config"
	"claude-squad/web"
	"claude-squad/web/mock"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestWebServer tests the entire web server with simulated terminal sessions.
func TestWebServer(t *testing.T) {
	// Create mock storage with sample instances
	storage := mock.NewMockStorage()
	
	// Create test config
	cfg := config.DefaultConfig()
	cfg.WebServerEnabled = true
	cfg.WebServerPort = 8080
	cfg.WebServerHost = "localhost"
	cfg.WebServerAllowLocalhost = true  // Allow localhost without auth
	
	// Create server with mock storage
	server, err := web.NewServer(storage, cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	
	// Start server for testing
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	
	// Set up a test HTTP server
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()
	
	// Run tests
	t.Run("ListInstances", func(t *testing.T) {
		testListInstances(t, ts.URL)
	})
	
	t.Run("InstanceDetail", func(t *testing.T) {
		testInstanceDetail(t, ts.URL)
	})
	
	t.Run("InstanceOutput", func(t *testing.T) {
		testInstanceOutput(t, ts.URL)
	})
	
	t.Run("InstanceDiff", func(t *testing.T) {
		testInstanceDiff(t, ts.URL)
	})
	
	// Allow time for simulated activity
	time.Sleep(100 * time.Millisecond)
	
	// Shut down the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	server.Stop()
}

// testListInstances tests the /api/instances endpoint.
func testListInstances(t *testing.T, baseURL string) {
	url := fmt.Sprintf("%s/api/instances", baseURL)
	
	// Make request
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	
	// Decode response
	var result struct {
		Instances []map[string]interface{} `json:"instances"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Verify instances
	if len(result.Instances) == 0 {
		t.Errorf("Expected instances, got none")
	}
	
	// Check instance fields
	for _, instance := range result.Instances {
		// Check required fields
		requiredFields := []string{"title", "status", "path"}
		for _, field := range requiredFields {
			if _, ok := instance[field]; !ok {
				t.Errorf("Instance missing required field: %s", field)
			}
		}
	}
}

// testInstanceDetail tests the /api/instances/{name} endpoint.
func testInstanceDetail(t *testing.T, baseURL string) {
	url := fmt.Sprintf("%s/api/instances/instance1", baseURL)
	
	// Make request
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	
	// Decode response
	var instance map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&instance); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Check instance fields
	if instance["title"] != "instance1" {
		t.Errorf("Expected title 'instance1', got '%v'", instance["title"])
	}
	
	// Also test non-existent instance
	url = fmt.Sprintf("%s/api/instances/nonexistent", baseURL)
	resp, err = http.Get(url)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status code %d for nonexistent instance, got %d", 
			http.StatusNotFound, resp.StatusCode)
	}
}

// testInstanceOutput tests the /api/instances/{name}/output endpoint.
func testInstanceOutput(t *testing.T, baseURL string) {
	url := fmt.Sprintf("%s/api/instances/instance1/output", baseURL)
	
	// Make request
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}
	
	// Decode response
	var output map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Check output fields
	if _, ok := output["content"]; !ok {
		t.Errorf("Output missing content field")
	}
	
	// Test different formats
	formats := []string{"ansi", "text", "html"}
	for _, format := range formats {
		url := fmt.Sprintf("%s/api/instances/instance1/output?format=%s", baseURL, format)
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status code %d for format %s, got %d", 
				http.StatusOK, format, resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// testInstanceDiff tests the /api/instances/{name}/diff endpoint.
func testInstanceDiff(t *testing.T, baseURL string) {
	url := fmt.Sprintf("%s/api/instances/instance1/diff", baseURL)
	
	// Make request
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	// For mock instances, diff might not be available
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status code %d or %d, got %d", 
			http.StatusOK, http.StatusBadRequest, resp.StatusCode)
	}
	
	// Test different formats
	formats := []string{"raw", "stats", "parsed"}
	for _, format := range formats {
		url := fmt.Sprintf("%s/api/instances/instance1/diff?format=%s", baseURL, format)
		resp, err := http.Get(url)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		// Just ensuring no server errors
		if resp.StatusCode >= 500 {
			t.Errorf("Got server error %d for format %s", resp.StatusCode, format)
		}
		resp.Body.Close()
	}
}