package browser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	// Save original env var and function
	originalURL := os.Getenv("BROWSER_SERVER_URL")
	originalNewClient := NewClient
	defer func() {
		// Restore original env var
		if originalURL != "" {
			os.Setenv("BROWSER_SERVER_URL", originalURL)
		} else {
			os.Unsetenv("BROWSER_SERVER_URL")
		}
		// Restore original function
		NewClient = originalNewClient
	}()

	// Test with no environment variable
	os.Unsetenv("BROWSER_SERVER_URL")
	client := NewClient()
	if c, ok := client.(*Client); ok {
		if c.baseURL != "http://0.0.0.0:8000" && c.baseURL != "http://0.0.0.0:8000" {
			t.Errorf("Expected default base URL to be http://0.0.0.0:8000 or http://0.0.0.0:8000, got %s", c.baseURL)
		}
	} else {
		t.Errorf("Expected client to be of type *Client")
	}

	// Test with custom environment variable
	os.Setenv("BROWSER_SERVER_URL", "http://custom-browser-server:9000")
	client = NewClient()
	if c, ok := client.(*Client); ok {
		if c.baseURL != "http://custom-browser-server:9000" {
			t.Errorf("Expected base URL to be http://custom-browser-server:9000, got %s", c.baseURL)
		}
	} else {
		t.Errorf("Expected client to be of type *Client")
	}
}

func TestCreateSession(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and path
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/sessions" {
			t.Errorf("Expected path /sessions, got %s", r.URL.Path)
		}

		// Check content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Decode request
		var req CreateSessionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Error decoding request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Check request fields
		if req.BrowserType != "firefox" {
			t.Errorf("Expected browser_type firefox, got %s", req.BrowserType)
		}
		if req.Headless != false {
			t.Errorf("Expected headless false, got %v", req.Headless)
		}

		// Prepare response with correctly formatted times
		now := time.Now()
		resp := map[string]interface{}{
			"id":           "test-session-id",
			"browser_type": req.BrowserType,
			"headless":     req.Headless,
			"created_at":   now.Format(time.RFC3339),
			"expires_at":   now.Add(5 * time.Minute).Format(time.RFC3339),
			"cdp_url":      "ws://localhost:9222/devtools/browser/abcdef123",
			"viewport_size": map[string]int{
				"width":  1280,
				"height": 720,
			},
		}

		// Set response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client with test server URL
	client := &Client{
		baseURL:    server.URL,
		httpClient: &http.Client{},
	}

	// Create session request
	req := CreateSessionRequest{
		BrowserType: "firefox",
		Headless:    false,
		ViewportSize: &ViewportSize{
			Width:  1280,
			Height: 720,
		},
	}

	// Call CreateSession
	session, err := client.CreateSession(context.Background(), req)
	if err != nil {
		t.Fatalf("Error creating session: %v", err)
	}

	// Check response
	if session.ID != "test-session-id" {
		t.Errorf("Expected session ID test-session-id, got %s", session.ID)
	}
	if session.BrowserType != "firefox" {
		t.Errorf("Expected browser_type firefox, got %s", session.BrowserType)
	}
	if session.Headless != false {
		t.Errorf("Expected headless false, got %v", session.Headless)
	}
	if session.CdpURL != "ws://localhost:9222/devtools/browser/abcdef123" {
		t.Errorf("Expected CDP URL ws://localhost:9222/devtools/browser/abcdef123, got %s", session.CdpURL)
	}
}

func TestCreateSessionError(t *testing.T) {
	// Setup test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{
			Detail: "Failed to create browser session",
		})
	}))
	defer server.Close()

	// Create client with test server URL
	client := &Client{
		baseURL:    server.URL,
		httpClient: &http.Client{},
	}

	// Call CreateSession
	_, err := client.CreateSession(context.Background(), CreateSessionRequest{BrowserType: "firefox"})
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Check error message
	expectedErrMsg := "browser server error: Failed to create browser session (status code 500)"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message %q, got %q", expectedErrMsg, err.Error())
	}
}

func TestDeleteSession(t *testing.T) {
	// Setup test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request method and path
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}
		if r.URL.Path != "/sessions/test-session-id" {
			t.Errorf("Expected path /sessions/test-session-id, got %s", r.URL.Path)
		}

		// Set response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"message": "Session test-session-id terminated",
		})
	}))
	defer server.Close()

	// Create client with test server URL
	client := &Client{
		baseURL:    server.URL,
		httpClient: &http.Client{},
	}

	// Call DeleteSession
	err := client.DeleteSession(context.Background(), "test-session-id")
	if err != nil {
		t.Fatalf("Error deleting session: %v", err)
	}
}

func TestDeleteSessionError(t *testing.T) {
	// Setup test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Detail: "Session not found",
		})
	}))
	defer server.Close()

	// Create client with test server URL
	client := &Client{
		baseURL:    server.URL,
		httpClient: &http.Client{},
	}

	// Call DeleteSession
	err := client.DeleteSession(context.Background(), "nonexistent-id")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Check error message
	expectedErrMsg := "browser server error: Session not found (status code 404)"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message %q, got %q", expectedErrMsg, err.Error())
	}
}
