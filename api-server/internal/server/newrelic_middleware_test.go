package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// mockApp is a mock of newrelic.Application for testing
type mockNewRelicApp struct {
	startTxnCalled bool
	lastTxnName    string
}

func (m *mockNewRelicApp) StartTransaction(name string) *mockTransaction {
	m.startTxnCalled = true
	m.lastTxnName = name
	return &mockTransaction{}
}

// Implement the required methods from the newrelic.Application interface
func (m *mockNewRelicApp) RecordCustomEvent(eventType string, params map[string]interface{}) error { return nil }
func (m *mockNewRelicApp) RecordCustomMetric(name string, value float64) error { return nil }
func (m *mockNewRelicApp) WaitForConnection(timeout time.Duration) bool { return true }
func (m *mockNewRelicApp) Shutdown(timeout time.Duration) {}

// mockTransaction is a mock of newrelic.Transaction for testing
type mockTransaction struct {
	ended bool
}

func (m *mockTransaction) End() error {
	m.ended = true
	return nil
}

func (m *mockTransaction) SetWebResponse(w http.ResponseWriter) http.ResponseWriter {
	return w
}

// Implement the required methods from the newrelic.Transaction interface
func (m *mockTransaction) AddAttribute(key string, value interface{}) error { return nil }
func (m *mockTransaction) SetName(name string) error { return nil }
func (m *mockTransaction) NoticeError(err error) error { return nil }
func (m *mockTransaction) StartSegmentNow() interface{} { return struct{}{} }
func (m *mockTransaction) SetWebRequestHTTP(r *http.Request) interface{} { return m }
func (m *mockTransaction) InsertDistributedTraceHeaders(hdrs http.Header) {}
func (m *mockTransaction) AcceptDistributedTraceHeaders(method string, hdrs http.Header) {}
func (m *mockTransaction) Application() interface{} { return &mockNewRelicApp{} }
func (m *mockTransaction) Context() context.Context { return context.Background() }
func (m *mockTransaction) GetTraceMetadata() interface{} { return struct{}{} }

func TestNewRelicMiddleware(t *testing.T) {
	// Test cases
	tests := []struct {
		name        string
		setupServer func() *Server
		path        string
	}{
		{
			name: "Without New Relic",
			setupServer: func() *Server {
				return &Server{
					nrApp: nil,
				}
			},
			path: "/test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := tc.setupServer()
			
			// Create a test handler that the middleware will wrap
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("OK"))
			})
			
			// Apply the middleware
			handler := server.NewRelicMiddleware(testHandler)
			
			// Create a test request
			req := httptest.NewRequest("GET", tc.path, nil)
			rr := httptest.NewRecorder()
			
			// Execute request
			handler.ServeHTTP(rr, req)
			
			// Check response
			if rr.Code != http.StatusOK {
				t.Errorf("Expected status code %d, got %d", http.StatusOK, rr.Code)
			}
		})
	}
}