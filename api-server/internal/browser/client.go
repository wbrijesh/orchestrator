package browser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client for interacting with the browser server
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// BrowserClient defines the interface for browser operations
type BrowserClient interface {
	CreateSession(ctx context.Context, req CreateSessionRequest) (*SessionResponse, error)
	DeleteSession(ctx context.Context, sessionID string) error
}

// NewClientFunc is the function type for creating a new browser client
type NewClientFunc func() BrowserClient

// defaultNewClient is the default implementation for creating a new browser client
func defaultNewClient() BrowserClient {
	// Check if we're in mock mode
	if os.Getenv("BROWSER_SERVER_MOCK") == "true" {
		log.Printf("Using mock browser client - no actual browser will be launched")
		return &MockClient{}
	}

	// Get base URL from environment or use default
	baseURL := os.Getenv("BROWSER_SERVER_URL")
	if baseURL == "" {
		// For Docker Compose setups, we should use the service name
		baseURL = "http://host.docker.internal:8000"
	}

	// Log the URL we're using
	log.Printf("Connecting to browser server at %s", baseURL)

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClient is the exported function variable that can be overridden in tests
var NewClient NewClientFunc = defaultNewClient

// CreateSessionRequest represents the parameters for creating a browser session
type CreateSessionRequest struct {
	BrowserType  string        `json:"browser_type"`
	Headless     bool          `json:"headless"`
	ViewportSize *ViewportSize `json:"viewport_size,omitempty"`
	UserAgent    *string       `json:"user_agent,omitempty"`
	Timeout      *int          `json:"timeout,omitempty"`
}

// ViewportSize represents browser viewport dimensions
type ViewportSize struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// SessionResponse represents a browser session from the browser server
type SessionResponse struct {
	ID           string       `json:"id"`
	BrowserType  string       `json:"browser_type"`
	Headless     bool         `json:"headless"`
	CreatedAt    FlexibleTime `json:"created_at"`
	ExpiresAt    FlexibleTime `json:"expires_at"`
	CdpURL       string       `json:"cdp_url"`
	ViewportSize ViewportSize `json:"viewport_size"`
	UserAgent    *string      `json:"user_agent,omitempty"`
}

// ErrorResponse represents an error from the browser server
type ErrorResponse struct {
	Detail string `json:"detail"`
}

// FlexibleTime is a custom time type that can handle multiple time formats
type FlexibleTime time.Time

// UnmarshalJSON implements custom JSON unmarshaling for FlexibleTime
func (ft *FlexibleTime) UnmarshalJSON(data []byte) error {
	// Remove quotes
	s := strings.Trim(string(data), "\"")
	if s == "null" || s == "" {
		return nil
	}

	// Try parsing with different formats, starting with the most specific
	formats := []string{
		time.RFC3339Nano,             // With timezone, nanoseconds
		time.RFC3339,                 // With timezone
		"2006-01-02T15:04:05.999999", // No timezone, microseconds
		"2006-01-02T15:04:05.999",    // No timezone, milliseconds
		"2006-01-02T15:04:05",        // No timezone
	}

	var err error
	var t time.Time
	for _, format := range formats {
		t, err = time.Parse(format, s)
		if err == nil {
			*ft = FlexibleTime(t)
			return nil
		}
	}

	// None of the formats worked
	return fmt.Errorf("cannot parse %q as FlexibleTime using known formats", s)
}

// Time converts FlexibleTime to standard time.Time
func (ft FlexibleTime) Time() time.Time {
	return time.Time(ft)
}

// CreateSession creates a new browser session
func (c *Client) CreateSession(ctx context.Context, req CreateSessionRequest) (*SessionResponse, error) {
	// Marshal request to JSON
	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create session request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/sessions", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Handle error
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, fmt.Errorf("received non-OK status %d and failed to decode error response", resp.StatusCode)
		}
		return nil, fmt.Errorf("browser server error: %s (status code %d)", errResp.Detail, resp.StatusCode)
	}

	// Decode response
	var session SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil, fmt.Errorf("failed to decode session response: %w", err)
	}

	return &session, nil
}

// DeleteSession deletes a browser session
func (c *Client) DeleteSession(ctx context.Context, sessionID string) error {
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("%s/sessions/%s", c.baseURL, sessionID), nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Handle error
	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return fmt.Errorf("received non-OK status %d and failed to decode error response", resp.StatusCode)
		}
		return fmt.Errorf("browser server error: %s (status code %d)", errResp.Detail, resp.StatusCode)
	}

	return nil
}
