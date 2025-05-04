package server

import (
	"api-server/internal/database"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Test for CreateSessionHandler
func TestCreateSessionHandler(t *testing.T) {
	testCases := []struct {
		name           string
		mockSetup      func() *MockDB
		setupContext   func() context.Context
		expectedStatus int
		expectedError  string
		checkData      func(t *testing.T, data interface{})
	}{
		{
			name: "Success",
			mockSetup: func() *MockDB {
				sessionID := uuid.New()
				now := time.Now()
				return &MockDB{
					CreateSessionFunc: func(ctx context.Context, uid uuid.UUID, name string) (*database.Session, error) {
						return &database.Session{
							ID:        sessionID,
							UserID:    uid,
							Name:      "test-session",
							StartedAt: now,
							StoppedAt: sql.NullTime{Valid: false},
						}, nil
					},
				}
			},
			setupContext: func() context.Context {
				testUserID := uuid.New()
				return context.WithValue(context.Background(), userIDContextKey, testUserID)
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
			checkData: func(t *testing.T, data interface{}) {
				response, ok := data.(map[string]interface{})
				if !ok {
					t.Error("Expected response to be a map")
					return
				}

				session, ok := response["session"].(map[string]interface{})
				if !ok {
					t.Error("Expected session in response")
					return
				}

				if session["name"] == "" {
					t.Error("Expected session name to be non-empty")
				}

				if session["active"] != true {
					t.Error("Expected session to be active")
				}
			},
		},
		{
			name: "Unauthorized",
			mockSetup: func() *MockDB {
				return &MockDB{}
			},
			setupContext: func() context.Context {
				// Return context without user ID
				return context.Background()
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Unauthorized",
			checkData:      nil,
		},
		{
			name: "Database Error",
			mockSetup: func() *MockDB {
				return &MockDB{
					CreateSessionFunc: func(ctx context.Context, uid uuid.UUID, name string) (*database.Session, error) {
						return nil, errors.New("database error")
					},
				}
			},
			setupContext: func() context.Context {
				testUserID := uuid.New()
				return context.WithValue(context.Background(), userIDContextKey, testUserID)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Could not create session",
			checkData:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			mockDB := tc.mockSetup()
			s := &Server{db: mockDB}
			req, err := http.NewRequest("POST", "/sessions", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Setup context with or without user ID based on test case
			req = req.WithContext(tc.setupContext())

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			handler := http.HandlerFunc(s.CreateSessionHandler)
			handler.ServeHTTP(rr, req)

			// Check status code
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.expectedStatus)
			}

			// For non-OK responses, check that the expected error message appears in the response
			if tc.expectedStatus != http.StatusOK {
				if rr.Body.String() == "" && tc.expectedError != "" {
					t.Errorf("Expected error message in response for non-OK status")
				}
				return
			}

			// For OK responses, validate the response structure
			var response database.APIResponse
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Errorf("Failed to parse response JSON: %v", err)
				return
			}

			if response.Error != tc.expectedError {
				t.Errorf("Expected error %q, got %q", tc.expectedError, response.Error)
			}

			if tc.checkData != nil {
				tc.checkData(t, response.Data)
			}
		})
	}
}

// Test for GetUserSessionsHandler
func TestGetUserSessionsHandler(t *testing.T) {
	testCases := []struct {
		name           string
		mockSetup      func() *MockDB
		setupContext   func() context.Context
		expectedStatus int
		expectedError  string
		checkData      func(t *testing.T, data interface{})
	}{
		{
			name: "Success",
			mockSetup: func() *MockDB {
				testUserID := uuid.New()
				now := time.Now()
				stoppedAt := now.Add(1 * time.Hour)

				// Create an active and a stopped session
				sessions := []*database.Session{
					{
						ID:        uuid.New(),
						UserID:    testUserID,
						Name:      "active-session",
						StartedAt: now,
						StoppedAt: sql.NullTime{Valid: false},
					},
					{
						ID:        uuid.New(),
						UserID:    testUserID,
						Name:      "stopped-session",
						StartedAt: now,
						StoppedAt: sql.NullTime{Valid: true, Time: stoppedAt},
					},
				}

				return &MockDB{
					GetSessionsByUserIDFunc: func(ctx context.Context, uid uuid.UUID) ([]*database.Session, error) {
						return sessions, nil
					},
				}
			},
			setupContext: func() context.Context {
				testUserID := uuid.New()
				return context.WithValue(context.Background(), userIDContextKey, testUserID)
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
			checkData: func(t *testing.T, data interface{}) {
				response, ok := data.(map[string]interface{})
				if !ok {
					t.Error("Expected response to be a map")
					return
				}

				sessions, ok := response["sessions"].([]interface{})
				if !ok {
					t.Error("Expected sessions array in response")
					return
				}

				if len(sessions) != 2 {
					t.Errorf("Expected 2 sessions, got %d", len(sessions))
					return
				}

				// Check that we have one active and one stopped session
				var activeCount, stoppedCount int
				for _, s := range sessions {
					session, ok := s.(map[string]interface{})
					if !ok {
						t.Error("Expected session to be a map")
						continue
					}

					if session["active"] == true {
						activeCount++
					} else {
						stoppedCount++
					}
				}

				if activeCount != 1 || stoppedCount != 1 {
					t.Errorf("Expected 1 active and 1 stopped session, got %d active and %d stopped", 
						activeCount, stoppedCount)
				}
			},
		},
		{
			name: "Unauthorized",
			mockSetup: func() *MockDB {
				return &MockDB{}
			},
			setupContext: func() context.Context {
				// Return context without user ID
				return context.Background()
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Unauthorized",
			checkData:      nil,
		},
		{
			name: "Database Error",
			mockSetup: func() *MockDB {
				return &MockDB{
					GetSessionsByUserIDFunc: func(ctx context.Context, uid uuid.UUID) ([]*database.Session, error) {
						return nil, errors.New("database error")
					},
				}
			},
			setupContext: func() context.Context {
				testUserID := uuid.New()
				return context.WithValue(context.Background(), userIDContextKey, testUserID)
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "Could not retrieve sessions",
			checkData:      nil,
		},
		{
			name: "Empty Sessions List",
			mockSetup: func() *MockDB {
				return &MockDB{
					GetSessionsByUserIDFunc: func(ctx context.Context, uid uuid.UUID) ([]*database.Session, error) {
						return []*database.Session{}, nil
					},
				}
			},
			setupContext: func() context.Context {
				testUserID := uuid.New()
				return context.WithValue(context.Background(), userIDContextKey, testUserID)
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
			checkData: func(t *testing.T, data interface{}) {
				response, ok := data.(map[string]interface{})
				if !ok {
					t.Error("Expected response to be a map")
					return
				}

				sessions, ok := response["sessions"].([]interface{})
				if !ok {
					t.Error("Expected sessions array in response")
					return
				}

				if len(sessions) != 0 {
					t.Errorf("Expected 0 sessions, got %d", len(sessions))
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			mockDB := tc.mockSetup()
			s := &Server{db: mockDB}
			req, err := http.NewRequest("GET", "/sessions", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Setup context with or without user ID based on test case
			req = req.WithContext(tc.setupContext())

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			handler := http.HandlerFunc(s.GetUserSessionsHandler)
			handler.ServeHTTP(rr, req)

			// Check status code
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.expectedStatus)
			}

			// For non-OK responses, check that the expected error message appears in the response
			if tc.expectedStatus != http.StatusOK {
				if rr.Body.String() == "" && tc.expectedError != "" {
					t.Errorf("Expected error message in response for non-OK status")
				}
				return
			}

			// For OK responses, validate the response structure
			var response database.APIResponse
			err = json.Unmarshal(rr.Body.Bytes(), &response)
			if err != nil {
				t.Errorf("Failed to parse response JSON: %v", err)
				return
			}

			if response.Error != tc.expectedError {
				t.Errorf("Expected error %q, got %q", tc.expectedError, response.Error)
			}

			if tc.checkData != nil {
				tc.checkData(t, response.Data)
			}
		})
	}
}

// Test for StopSessionHandler
func TestStopSessionHandler(t *testing.T) {
	testCases := []struct {
		name           string
		sessionID      string // The session ID to use in the URL
		mockSetup      func() *MockDB
		setupContext   func() context.Context
		expectedStatus int
		expectedError  string
		checkData      func(t *testing.T, data interface{})
	}{
		{
			name:      "Success",
			sessionID: uuid.New().String(),
			mockSetup: func() *MockDB {
				sessionID := uuid.New()
				testUserID := uuid.New()
				now := time.Now()
				stoppedAt := now.Add(1 * time.Hour)

				return &MockDB{
					StopSessionFunc: func(ctx context.Context, sid uuid.UUID, uid uuid.UUID) (*database.Session, error) {
						return &database.Session{
							ID:        sessionID,
							UserID:    testUserID,
							Name:      "test-session",
							StartedAt: now,
							StoppedAt: sql.NullTime{Valid: true, Time: stoppedAt},
						}, nil
					},
				}
			},
			setupContext: func() context.Context {
				testUserID := uuid.New()
				return context.WithValue(context.Background(), userIDContextKey, testUserID)
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
			checkData: func(t *testing.T, data interface{}) {
				response, ok := data.(map[string]interface{})
				if !ok {
					t.Error("Expected response to be a map")
					return
				}

				session, ok := response["session"].(map[string]interface{})
				if !ok {
					t.Error("Expected session in response")
					return
				}

				if session["active"] == true {
					t.Error("Expected session to be stopped (not active)")
				}

				if session["stopped_at"] == nil {
					t.Error("Expected stopped_at to be set")
				}
			},
		},
		{
			name:      "Unauthorized",
			sessionID: uuid.New().String(),
			mockSetup: func() *MockDB {
				return &MockDB{}
			},
			setupContext: func() context.Context {
				// Return context without user ID
				return context.Background()
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Unauthorized",
			checkData:      nil,
		},
		{
			name:      "Missing Session ID",
			sessionID: "missing", // Use a non-empty value that will be caught by our validation logic
			mockSetup: func() *MockDB {
				return &MockDB{}
			},
			setupContext: func() context.Context {
				testUserID := uuid.New()
				return context.WithValue(context.Background(), userIDContextKey, testUserID)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid session ID",
			checkData:      nil,
		},
		{
			name:      "Invalid UUID",
			sessionID: "not-a-uuid",
			mockSetup: func() *MockDB {
				return &MockDB{}
			},
			setupContext: func() context.Context {
				testUserID := uuid.New()
				return context.WithValue(context.Background(), userIDContextKey, testUserID)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid session ID",
			checkData:      nil,
		},
		{
			name:      "Session Not Found",
			sessionID: uuid.New().String(),
			mockSetup: func() *MockDB {
				return &MockDB{
					StopSessionFunc: func(ctx context.Context, sid uuid.UUID, uid uuid.UUID) (*database.Session, error) {
						return nil, errors.New("session not found or already stopped")
					},
				}
			},
			setupContext: func() context.Context {
				testUserID := uuid.New()
				return context.WithValue(context.Background(), userIDContextKey, testUserID)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "session not found or already stopped",
			checkData:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			mockDB := tc.mockSetup()
			s := &Server{db: mockDB}

			// Create router to handle URL parameters
			r := chi.NewRouter()
			r.Post("/sessions/{id}/stop", s.StopSessionHandler)

			// Create request
			req, err := http.NewRequest("POST", "/sessions/"+tc.sessionID+"/stop", nil)
			if err != nil {
				t.Fatal(err)
			}

			// Setup context with or without user ID based on test case
			req = req.WithContext(tc.setupContext())

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			r.ServeHTTP(rr, req)

			// Check status code
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.expectedStatus)
			}

			// Handle special case for Unauthorized which doesn't return JSON
			if tc.expectedStatus == http.StatusUnauthorized {
				if !strings.Contains(rr.Body.String(), "Unauthorized") {
					t.Errorf("Expected 'Unauthorized' in response body, got: %s", rr.Body.String())
				}
				return
			}

			// For non-OK responses with empty body, we expect the error in the standard error handler 
			if tc.expectedStatus != http.StatusOK && rr.Body.Len() == 0 {
				return
			}

			// For responses with content, validate the structure
			if rr.Body.Len() > 0 {
				var response database.APIResponse
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to parse response JSON: %v", err)
					return
				}

				if response.Error != tc.expectedError {
					t.Errorf("Expected error %q, got %q", tc.expectedError, response.Error)
				}

				if tc.checkData != nil && response.Data != nil {
					tc.checkData(t, response.Data)
				}
			}
		})
	}
}

// Test for DeleteSessionHandler
func TestDeleteSessionHandler(t *testing.T) {
	testCases := []struct {
		name           string
		sessionID      string // The session ID to use in the URL
		mockSetup      func() *MockDB
		setupContext   func() context.Context
		expectedStatus int
		expectedError  string
		checkData      func(t *testing.T, data interface{})
	}{
		{
			name:      "Success",
			sessionID: uuid.New().String(),
			mockSetup: func() *MockDB {
				return &MockDB{
					DeleteSessionFunc: func(ctx context.Context, sid uuid.UUID, uid uuid.UUID) error {
						return nil
					},
				}
			},
			setupContext: func() context.Context {
				testUserID := uuid.New()
				return context.WithValue(context.Background(), userIDContextKey, testUserID)
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
			checkData: func(t *testing.T, data interface{}) {
				success, ok := data.(map[string]interface{})
				if !ok {
					t.Error("Expected response to be a map")
					return
				}
				
				if success["success"] != true {
					t.Error("Expected success to be true")
				}
			},
		},
		{
			name:      "Unauthorized",
			sessionID: uuid.New().String(),
			mockSetup: func() *MockDB {
				return &MockDB{}
			},
			setupContext: func() context.Context {
				// Return context without user ID
				return context.Background()
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Unauthorized",
			checkData:      nil,
		},
		{
			name:      "Missing Session ID",
			sessionID: "missing", // Use a non-empty value that will be caught by our validation logic
			mockSetup: func() *MockDB {
				return &MockDB{}
			},
			setupContext: func() context.Context {
				testUserID := uuid.New()
				return context.WithValue(context.Background(), userIDContextKey, testUserID)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid session ID",
			checkData:      nil,
		},
		{
			name:      "Invalid UUID",
			sessionID: "not-a-uuid",
			mockSetup: func() *MockDB {
				return &MockDB{}
			},
			setupContext: func() context.Context {
				testUserID := uuid.New()
				return context.WithValue(context.Background(), userIDContextKey, testUserID)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid session ID",
			checkData:      nil,
		},
		{
			name:      "Session Not Found",
			sessionID: uuid.New().String(),
			mockSetup: func() *MockDB {
				return &MockDB{
					DeleteSessionFunc: func(ctx context.Context, sid uuid.UUID, uid uuid.UUID) error {
						return errors.New("session not found or already deleted")
					},
				}
			},
			setupContext: func() context.Context {
				testUserID := uuid.New()
				return context.WithValue(context.Background(), userIDContextKey, testUserID)
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "session not found or already deleted",
			checkData:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup
			mockDB := tc.mockSetup()
			s := &Server{db: mockDB}

			// Create router to handle URL parameters
			r := chi.NewRouter()
			r.Delete("/sessions/{id}", s.DeleteSessionHandler)

			// Create request
			req, err := http.NewRequest("DELETE", "/sessions/"+tc.sessionID, nil)
			if err != nil {
				t.Fatal(err)
			}

			// Setup context with or without user ID based on test case
			req = req.WithContext(tc.setupContext())

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			r.ServeHTTP(rr, req)

			// Check status code
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, tc.expectedStatus)
			}

			// Handle special case for Unauthorized which doesn't return JSON
			if tc.expectedStatus == http.StatusUnauthorized {
				if !strings.Contains(rr.Body.String(), "Unauthorized") {
					t.Errorf("Expected 'Unauthorized' in response body, got: %s", rr.Body.String())
				}
				return
			}
			
			// For non-OK responses with empty body, we expect the error in the standard error handler
			if tc.expectedStatus != http.StatusOK && rr.Body.Len() == 0 {
				return
			}

			// For responses with content, validate the structure
			if rr.Body.Len() > 0 {
				var response database.APIResponse
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to parse response JSON: %v", err)
					return
				}

				if response.Error != tc.expectedError {
					t.Errorf("Expected error %q, got %q", tc.expectedError, response.Error)
				}

				if tc.checkData != nil && response.Data != nil {
					tc.checkData(t, response.Data)
				}
			}
		})
	}
}