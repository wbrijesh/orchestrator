// Package browser provides a client for interacting with the browser server
package browser

import (
	"context"
	"time"
)

// MockClient provides a mock implementation for testing
type MockClient struct {
	CreateSessionFunc func(ctx context.Context, req CreateSessionRequest) (*SessionResponse, error)
	DeleteSessionFunc func(ctx context.Context, sessionID string) error
}

// CreateSession calls the mocked function
func (m *MockClient) CreateSession(ctx context.Context, req CreateSessionRequest) (*SessionResponse, error) {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(ctx, req)
	}
	// Default mock response
	now := time.Now()
	expireTime := now.Add(1 * time.Hour)
	return &SessionResponse{
		ID:          "mock-browser-id",
		BrowserType: req.BrowserType,
		Headless:    req.Headless,
		CreatedAt:   FlexibleTime(now),
		ExpiresAt:   FlexibleTime(expireTime),
		CdpURL:      "ws://localhost:9222/devtools/browser/mock",
		ViewportSize: ViewportSize{
			Width:  req.ViewportSize.Width,
			Height: req.ViewportSize.Height,
		},
		UserAgent: req.UserAgent,
	}, nil
}

// DeleteSession calls the mocked function
func (m *MockClient) DeleteSession(ctx context.Context, sessionID string) error {
	if m.DeleteSessionFunc != nil {
		return m.DeleteSessionFunc(ctx, sessionID)
	}
	return nil
}