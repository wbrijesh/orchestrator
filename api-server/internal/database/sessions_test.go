package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Helper function for creating test browser parameters
func getTestBrowserParams() (string, string, string, bool, int, int) {
	return "test-browser-id", "firefox", "ws://localhost:9222/devtools/browser/test", false, 1280, 720
}

func setupSessionsTable(t *testing.T, s *service) {
	ctx := context.Background()
	
	// Create schema (use public for testing)
	schema = "public"
	
	// First ensure we can connect to the database
	if err := s.db.PingContext(ctx); err != nil {
		t.Skipf("Skipping test: database connection failed: %v", err)
		return
	}
	
	// Set search_path explicitly
	_, err := s.db.ExecContext(ctx, `SET search_path TO public`)
	if err != nil {
		t.Fatalf("failed to set search path: %v", err)
	}
	
	// Create the users table first (if it doesn't exist), as sessions depends on it
	_, err = s.db.ExecContext(ctx, `
		CREATE EXTENSION IF NOT EXISTS pgcrypto;
		
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			first_name VARCHAR(255) NOT NULL,
			last_name VARCHAR(255) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}
	
	// Create sessions table with browser fields
	_, err = s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS sessions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			user_id UUID NOT NULL REFERENCES users(id),
			name VARCHAR(255) NOT NULL,
			started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			stopped_at TIMESTAMP WITH TIME ZONE,
			browser_id VARCHAR(255) DEFAULT '' NOT NULL,
			browser_type VARCHAR(50) DEFAULT 'firefox' NOT NULL,
			cdp_url TEXT DEFAULT '' NOT NULL,
			headless BOOLEAN DEFAULT false NOT NULL,
			viewport_w INTEGER DEFAULT 1280 NOT NULL,
			viewport_h INTEGER DEFAULT 720 NOT NULL,
			user_agent TEXT DEFAULT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create sessions table: %v", err)
	}
}

// Create a test user and return the ID
func createTestUser(t *testing.T, s *service) uuid.UUID {
	ctx := context.Background()
	
	// Generate a unique email using UUID to avoid conflicts
	uuid := uuid.New()
	testUser := &User{
		Email:        "session-test-" + uuid.String() + "@example.com",
		FirstName:    "Session",
		LastName:     "Test",
		PasswordHash: "hashedpassword",
	}
	
	id, err := s.CreateUser(ctx, testUser)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	
	return id
}

func TestCreateSession(t *testing.T) {
	// Reset dbInstance to get fresh connection
	originalInstance := dbInstance
	dbInstance = nil
	defer func() {
		dbInstance = originalInstance
	}()
	
	srv := New()
	s, ok := srv.(*service)
	if !ok {
		t.Fatal("expected srv to be of type *service")
	}
	
	// Setup test table
	setupSessionsTable(t, s)
	
	// Create a test user
	userID := createTestUser(t, s)
	
	// Test creating a session
	ctx := context.Background()
	sessionName := "test-session"
	browserID := "test-browser-id"
	browserType := "firefox"
	cdpURL := "ws://localhost:9222/devtools/browser/test"
	headless := false
	viewportW := 1280
	viewportH := 720
	
	session, err := s.CreateSession(ctx, userID, sessionName, 
		browserID, browserType, cdpURL, headless, viewportW, viewportH, nil)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	
	// Verify session was created with correct data
	assert.NotEqual(t, uuid.Nil, session.ID, "Expected non-nil session ID")
	assert.Equal(t, userID, session.UserID, "Expected session to be linked to correct user")
	assert.Equal(t, sessionName, session.Name, "Expected session name to match")
	assert.False(t, session.StartedAt.IsZero(), "Expected StartedAt to be set")
	assert.False(t, session.StoppedAt.Valid, "Expected StoppedAt to be null initially")
	assert.Equal(t, browserID, session.BrowserID, "Expected browser ID to match")
	assert.Equal(t, browserType, session.BrowserType, "Expected browser type to match")
	assert.Equal(t, cdpURL, session.CdpURL, "Expected CDP URL to match")
	assert.Equal(t, headless, session.Headless, "Expected headless to match")
	assert.Equal(t, viewportW, session.ViewportW, "Expected viewport width to match")
	assert.Equal(t, viewportH, session.ViewportH, "Expected viewport height to match")
	
	// Test creating a session with invalid user ID
	invalidID := uuid.New()
	_, err = s.CreateSession(ctx, invalidID, "invalid-user-session", 
		browserID, browserType, cdpURL, headless, viewportW, viewportH, nil)
	assert.Error(t, err, "Expected error when creating session for non-existent user")
}

func TestGetSessionsByUserID(t *testing.T) {
	// Reset dbInstance to get fresh connection
	originalInstance := dbInstance
	dbInstance = nil
	defer func() {
		dbInstance = originalInstance
	}()
	
	srv := New()
	s, ok := srv.(*service)
	if !ok {
		t.Fatal("expected srv to be of type *service")
	}
	
	// Setup test table
	setupSessionsTable(t, s)
	
	// Create a test user
	userID := createTestUser(t, s)
	ctx := context.Background()
	
	// Create multiple sessions for the user
	sessions := []string{"session1", "session2", "session3"}
	browserID, browserType, cdpURL, headless, viewportW, viewportH := getTestBrowserParams()
	for _, name := range sessions {
		_, err := s.CreateSession(ctx, userID, name, 
			browserID+"-"+name, browserType, cdpURL, headless, viewportW, viewportH, nil)
		if err != nil {
			t.Fatalf("failed to create test session %s: %v", name, err)
		}
	}
	
	// Test fetching sessions
	userSessions, err := s.GetSessionsByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("failed to get sessions for user: %v", err)
	}
	
	// We should have at least the number of sessions we created
	assert.GreaterOrEqual(t, len(userSessions), len(sessions), 
		"Expected at least %d sessions, got %d", len(sessions), len(userSessions))
	
	// Test with non-existent user ID - should return empty list, not error
	nonexistentID := uuid.New()
	emptySessions, err := s.GetSessionsByUserID(ctx, nonexistentID)
	assert.NoError(t, err, "Expected no error for non-existent user ID")
	assert.Empty(t, emptySessions, "Expected empty session list for non-existent user")
}

func TestGetSessionByID(t *testing.T) {
	// Reset dbInstance to get fresh connection
	originalInstance := dbInstance
	dbInstance = nil
	defer func() {
		dbInstance = originalInstance
	}()
	
	srv := New()
	s, ok := srv.(*service)
	if !ok {
		t.Fatal("expected srv to be of type *service")
	}
	
	// Setup test table
	setupSessionsTable(t, s)
	
	// Create a test user
	userID := createTestUser(t, s)
	ctx := context.Background()
	
	// Create a session
	browserID, browserType, cdpURL, headless, viewportW, viewportH := getTestBrowserParams()
	session, err := s.CreateSession(ctx, userID, "test-session-byid", 
		browserID, browserType, cdpURL, headless, viewportW, viewportH, nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	
	// Test fetching the session by ID
	fetchedSession, err := s.GetSessionByID(ctx, session.ID, userID)
	if err != nil {
		t.Fatalf("failed to get session by ID: %v", err)
	}
	
	assert.Equal(t, session.ID, fetchedSession.ID, "Expected matching session IDs")
	assert.Equal(t, session.Name, fetchedSession.Name, "Expected matching session names")
	
	// Test with non-existent session ID
	nonexistentID := uuid.New()
	_, err = s.GetSessionByID(ctx, nonexistentID, userID)
	assert.Error(t, err, "Expected error for non-existent session ID")
	assert.Contains(t, err.Error(), "session not found", "Expected 'session not found' error")
	
	// Test with wrong user ID (session exists, but belongs to a different user)
	wrongUserID := uuid.New()
	_, err = s.GetSessionByID(ctx, session.ID, wrongUserID)
	assert.Error(t, err, "Expected error when accessing session with wrong user ID")
	assert.Contains(t, err.Error(), "session not found", "Expected 'session not found' error")
}

func TestStopSession(t *testing.T) {
	// Reset dbInstance to get fresh connection
	originalInstance := dbInstance
	dbInstance = nil
	defer func() {
		dbInstance = originalInstance
	}()
	
	srv := New()
	s, ok := srv.(*service)
	if !ok {
		t.Fatal("expected srv to be of type *service")
	}
	
	// Setup test table
	setupSessionsTable(t, s)
	
	// Create a test user
	userID := createTestUser(t, s)
	ctx := context.Background()
	
	// Create a session
	browserID, browserType, cdpURL, headless, viewportW, viewportH := getTestBrowserParams()
	session, err := s.CreateSession(ctx, userID, "test-stop-session", 
		browserID, browserType, cdpURL, headless, viewportW, viewportH, nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	
	// Test stopping the session
	stoppedSession, err := s.StopSession(ctx, session.ID, userID)
	if err != nil {
		t.Fatalf("failed to stop session: %v", err)
	}
	
	assert.Equal(t, session.ID, stoppedSession.ID, "Expected matching session IDs")
	assert.True(t, stoppedSession.StoppedAt.Valid, "Expected StoppedAt to be set")
	assert.False(t, stoppedSession.StoppedAt.Time.IsZero(), "Expected StoppedAt time to be non-zero")
	
	// Test stopping an already stopped session
	_, err = s.StopSession(ctx, session.ID, userID)
	assert.Error(t, err, "Expected error when stopping an already stopped session")
	assert.Contains(t, err.Error(), "session not found or already stopped", 
		"Expected 'session not found or already stopped' error")
	
	// Test with non-existent session ID
	nonexistentID := uuid.New()
	_, err = s.StopSession(ctx, nonexistentID, userID)
	assert.Error(t, err, "Expected error for non-existent session ID")
	assert.Contains(t, err.Error(), "session not found or already stopped", 
		"Expected 'session not found or already stopped' error")
	
	// Test with wrong user ID (session exists, but belongs to a different user)
	// First create a new session
	browserID, browserType, cdpURL, headless, viewportW, viewportH = getTestBrowserParams()
	newSession, err := s.CreateSession(ctx, userID, "test-wrong-user-stop", 
		browserID+"-wrong-user", browserType, cdpURL, headless, viewportW, viewportH, nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	
	wrongUserID := uuid.New()
	_, err = s.StopSession(ctx, newSession.ID, wrongUserID)
	assert.Error(t, err, "Expected error when stopping session with wrong user ID")
	assert.Contains(t, err.Error(), "session not found or already stopped", 
		"Expected 'session not found or already stopped' error")
}

func TestDeleteSession(t *testing.T) {
	// Reset dbInstance to get fresh connection
	originalInstance := dbInstance
	dbInstance = nil
	defer func() {
		dbInstance = originalInstance
	}()
	
	srv := New()
	s, ok := srv.(*service)
	if !ok {
		t.Fatal("expected srv to be of type *service")
	}
	
	// Setup test table
	setupSessionsTable(t, s)
	
	// Create a test user
	userID := createTestUser(t, s)
	ctx := context.Background()
	
	// Create a session
	browserID, browserType, cdpURL, headless, viewportW, viewportH := getTestBrowserParams()
	session, err := s.CreateSession(ctx, userID, "test-delete-session", 
		browserID, browserType, cdpURL, headless, viewportW, viewportH, nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	
	// Test deleting the session
	err = s.DeleteSession(ctx, session.ID, userID)
	assert.NoError(t, err, "Expected no error when deleting a session")
	
	// Verify the session is deleted by trying to fetch it
	_, err = s.GetSessionByID(ctx, session.ID, userID)
	assert.Error(t, err, "Expected error when fetching deleted session")
	assert.Contains(t, err.Error(), "session not found", "Expected 'session not found' error")
	
	// Test deleting a non-existent session
	nonexistentID := uuid.New()
	err = s.DeleteSession(ctx, nonexistentID, userID)
	assert.Error(t, err, "Expected error when deleting non-existent session")
	assert.Contains(t, err.Error(), "session not found or already deleted", 
		"Expected 'session not found or already deleted' error")
	
	// Test with wrong user ID (create a new session first)
	browserID, browserType, cdpURL, headless, viewportW, viewportH = getTestBrowserParams()
	newSession, err := s.CreateSession(ctx, userID, "test-wrong-user-delete", 
		browserID+"-delete", browserType, cdpURL, headless, viewportH, viewportW, nil)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}
	
	wrongUserID := uuid.New()
	err = s.DeleteSession(ctx, newSession.ID, wrongUserID)
	assert.Error(t, err, "Expected error when deleting session with wrong user ID")
	assert.Contains(t, err.Error(), "session not found or already deleted", 
		"Expected 'session not found or already deleted' error")
}

func TestSessionToView(t *testing.T) {
	now := time.Now()
	testCases := []struct {
		name               string
		session            Session
		expectedActive     bool
		expectedStoppedAt  bool  // whether StoppedAt should be non-nil
		expectedHasDuration bool // whether Duration should be non-nil
	}{
		{
			name: "Active Session",
			session: Session{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				Name:      "active-session",
				StartedAt: now,
				StoppedAt: sql.NullTime{Valid: false},
			},
			expectedActive:     true,
			expectedStoppedAt:  false,
			expectedHasDuration: false,
		},
		{
			name: "Stopped Session",
			session: Session{
				ID:        uuid.New(),
				UserID:    uuid.New(),
				Name:      "stopped-session",
				StartedAt: now,
				StoppedAt: sql.NullTime{Valid: true, Time: now.Add(1 * time.Hour)},
			},
			expectedActive:     false,
			expectedStoppedAt:  true,
			expectedHasDuration: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			view := tc.session.ToView()
			
			assert.Equal(t, tc.session.ID.String(), view.ID, "Expected matching ID strings")
			assert.Equal(t, tc.session.UserID.String(), view.UserID, "Expected matching User ID strings")
			assert.Equal(t, tc.session.Name, view.Name, "Expected matching names")
			assert.Equal(t, tc.session.StartedAt, view.StartedAt, "Expected matching start times")
			assert.Equal(t, tc.expectedActive, view.Active, "Expected matching active status")
			
			if tc.expectedStoppedAt {
				assert.NotNil(t, view.StoppedAt, "Expected non-nil StoppedAt")
			} else {
				assert.Nil(t, view.StoppedAt, "Expected nil StoppedAt")
			}
			
			if tc.expectedHasDuration {
				assert.NotNil(t, view.Duration, "Expected non-nil Duration")
				assert.NotEmpty(t, *view.Duration, "Expected non-empty Duration string")
			} else {
				assert.Nil(t, view.Duration, "Expected nil Duration")
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "Hours and Minutes",
			duration: 2*time.Hour + 30*time.Minute,
			expected: "2 hours, 30 minutes",
		},
		{
			name:     "Single Hour",
			duration: 1*time.Hour + 5*time.Minute,
			expected: "1 hour, 5 minutes",
		},
		{
			name:     "Minutes and Seconds",
			duration: 45*time.Minute + 15*time.Second,
			expected: "45 minutes, 15 seconds",
		},
		{
			name:     "Single Minute",
			duration: 1*time.Minute + 30*time.Second,
			expected: "1 minute, 30 seconds",
		},
		{
			name:     "Seconds Only",
			duration: 45 * time.Second,
			expected: "45 seconds",
		},
		{
			name:     "Single Second",
			duration: 1 * time.Second,
			expected: "1 second",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatted := formatDuration(tc.duration)
			assert.Equal(t, tc.expected, formatted, "Expected formatted duration to match")
		})
	}
}

func TestFormatTimePart(t *testing.T) {
	testCases := []struct {
		name     string
		value    time.Duration
		unit     string
		expected string
	}{
		{
			name:     "Plural Hours",
			value:    2,
			unit:     "hour",
			expected: "2 hours",
		},
		{
			name:     "Single Hour",
			value:    1,
			unit:     "hour",
			expected: "1 hour",
		},
		{
			name:     "Plural Minutes",
			value:    30,
			unit:     "minute",
			expected: "30 minutes",
		},
		{
			name:     "Single Minute",
			value:    1,
			unit:     "minute",
			expected: "1 minute",
		},
		{
			name:     "Plural Seconds",
			value:    45,
			unit:     "second",
			expected: "45 seconds",
		},
		{
			name:     "Single Second",
			value:    1,
			unit:     "second",
			expected: "1 second",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			formatted := formatTimePart(tc.value, tc.unit)
			assert.Equal(t, tc.expected, formatted, "Expected formatted time part to match")
		})
	}
}