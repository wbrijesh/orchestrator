package server

import (
	"api-server/internal/database"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMockDB(t *testing.T) {
	// Test each method in the MockDB implementation to ensure it properly
	// delegates to the provided function and returns default values when no function is provided

	t.Run("CreateUser", func(t *testing.T) {
		// Test with function provided
		expectedUser := &database.User{
			Email:     "test@example.com",
			FirstName: "Test",
			LastName:  "User",
		}
		expectedID := uuid.New()
		expectedError := errors.New("test error")
		
		mockDB := &MockDB{
			CreateUserFunc: func(ctx context.Context, u *database.User) (uuid.UUID, error) {
				// Verify the user passed to the function matches what we expect
				assert.Equal(t, expectedUser, u, "Expected matching user")
				return expectedID, expectedError
			},
		}
		
		// Call the method and verify it returns what we expect
		id, err := mockDB.CreateUser(context.Background(), expectedUser)
		assert.Equal(t, expectedID, id, "Expected ID to match")
		assert.Equal(t, expectedError, err, "Expected error to match")
		
		// Test with no function provided
		mockDB = &MockDB{}
		id, err = mockDB.CreateUser(context.Background(), expectedUser)
		assert.Equal(t, uuid.UUID{}, id, "Expected empty UUID")
		assert.Nil(t, err, "Expected nil error")
	})
	
	t.Run("GetUserByEmail", func(t *testing.T) {
		// Test with function provided
		expectedEmail := "test@example.com"
		expectedUser := &database.User{
			Email:     expectedEmail,
			FirstName: "Test",
			LastName:  "User",
		}
		expectedError := errors.New("test error")
		
		mockDB := &MockDB{
			GetUserByEmailFunc: func(ctx context.Context, email string) (*database.User, error) {
				// Verify the email passed to the function matches what we expect
				assert.Equal(t, expectedEmail, email, "Expected matching email")
				return expectedUser, expectedError
			},
		}
		
		// Call the method and verify it returns what we expect
		user, err := mockDB.GetUserByEmail(context.Background(), expectedEmail)
		assert.Equal(t, expectedUser, user, "Expected user to match")
		assert.Equal(t, expectedError, err, "Expected error to match")
		
		// Test with no function provided
		mockDB = &MockDB{}
		user, err = mockDB.GetUserByEmail(context.Background(), expectedEmail)
		assert.Equal(t, &database.User{}, user, "Expected empty user")
		assert.Nil(t, err, "Expected nil error")
	})
	
	t.Run("Health", func(t *testing.T) {
		// Test with function provided
		expectedHealth := map[string]string{
			"status": "up",
			"message": "test message",
		}
		
		mockDB := &MockDB{
			HealthFunc: func() map[string]string {
				return expectedHealth
			},
		}
		
		// Call the method and verify it returns what we expect
		health := mockDB.Health()
		assert.Equal(t, expectedHealth, health, "Expected health to match")
		
		// Test with no function provided
		mockDB = &MockDB{}
		health = mockDB.Health()
		assert.Equal(t, map[string]string{"status": "up"}, health, "Expected default health")
	})
	
	t.Run("Close", func(t *testing.T) {
		// Test with function provided
		expectedError := errors.New("test error")
		
		mockDB := &MockDB{
			CloseFunc: func() error {
				return expectedError
			},
		}
		
		// Call the method and verify it returns what we expect
		err := mockDB.Close()
		assert.Equal(t, expectedError, err, "Expected error to match")
		
		// Test with no function provided
		mockDB = &MockDB{}
		err = mockDB.Close()
		assert.Nil(t, err, "Expected nil error")
	})
	
	t.Run("CreateSession", func(t *testing.T) {
		// Test with function provided
		expectedUserID := uuid.New()
		expectedName := "test session"
		expectedBrowserID := "browser-123"
		expectedBrowserType := "firefox"
		expectedCdpURL := "ws://localhost:1234"
		expectedHeadless := false
		expectedViewportW := 1280
		expectedViewportH := 720
		expectedUserAgent := "Mozilla/5.0"
		
		expectedSession := &database.Session{
			ID:          uuid.New(),
			UserID:      expectedUserID,
			Name:        expectedName,
			BrowserID:   expectedBrowserID,
			BrowserType: expectedBrowserType,
			CdpURL:      expectedCdpURL,
			Headless:    expectedHeadless,
			ViewportW:   expectedViewportW,
			ViewportH:   expectedViewportH,
			UserAgent:   sql.NullString{String: expectedUserAgent, Valid: true},
		}
		expectedError := errors.New("test error")
		
		mockDB := &MockDB{
			CreateSessionFunc: func(ctx context.Context, userID uuid.UUID, name string, 
				browserID, browserType, cdpURL string, headless bool, 
				viewportW, viewportH int, userAgent *string) (*database.Session, error) {
				// Verify the parameters passed to the function match what we expect
				assert.Equal(t, expectedUserID, userID, "Expected matching user ID")
				assert.Equal(t, expectedName, name, "Expected matching session name")
				assert.Equal(t, expectedBrowserID, browserID, "Expected matching browser ID")
				assert.Equal(t, expectedBrowserType, browserType, "Expected matching browser type")
				assert.Equal(t, expectedCdpURL, cdpURL, "Expected matching CDP URL")
				assert.Equal(t, expectedHeadless, headless, "Expected matching headless value")
				assert.Equal(t, expectedViewportW, viewportW, "Expected matching viewport width")
				assert.Equal(t, expectedViewportH, viewportH, "Expected matching viewport height")
				assert.Equal(t, &expectedUserAgent, userAgent, "Expected matching user agent")
				
				return expectedSession, expectedError
			},
		}
		
		// Call the method and verify it returns what we expect
		session, err := mockDB.CreateSession(context.Background(), 
			expectedUserID, expectedName, 
			expectedBrowserID, expectedBrowserType, expectedCdpURL, 
			expectedHeadless, expectedViewportW, expectedViewportH,
			&expectedUserAgent)
			
		assert.Equal(t, expectedSession, session, "Expected session to match")
		assert.Equal(t, expectedError, err, "Expected error to match")
		
		// Test with no function provided
		mockDB = &MockDB{}
		session, err = mockDB.CreateSession(context.Background(), 
			expectedUserID, expectedName,
			expectedBrowserID, expectedBrowserType, expectedCdpURL, 
			expectedHeadless, expectedViewportW, expectedViewportH,
			&expectedUserAgent)
			
		assert.Equal(t, &database.Session{}, session, "Expected empty session")
		assert.Nil(t, err, "Expected nil error")
	})
	
	t.Run("GetSessionsByUserID", func(t *testing.T) {
		// Test with function provided
		expectedUserID := uuid.New()
		expectedSessions := []*database.Session{
			{
				ID:     uuid.New(),
				UserID: expectedUserID,
				Name:   "session 1",
			},
			{
				ID:     uuid.New(),
				UserID: expectedUserID,
				Name:   "session 2",
			},
		}
		expectedError := errors.New("test error")
		
		mockDB := &MockDB{
			GetSessionsByUserIDFunc: func(ctx context.Context, userID uuid.UUID) ([]*database.Session, error) {
				// Verify the user ID passed to the function matches what we expect
				assert.Equal(t, expectedUserID, userID, "Expected matching user ID")
				return expectedSessions, expectedError
			},
		}
		
		// Call the method and verify it returns what we expect
		sessions, err := mockDB.GetSessionsByUserID(context.Background(), expectedUserID)
		assert.Equal(t, expectedSessions, sessions, "Expected sessions to match")
		assert.Equal(t, expectedError, err, "Expected error to match")
		
		// Test with no function provided
		mockDB = &MockDB{}
		sessions, err = mockDB.GetSessionsByUserID(context.Background(), expectedUserID)
		assert.Equal(t, []*database.Session{}, sessions, "Expected empty sessions")
		assert.Nil(t, err, "Expected nil error")
	})
	
	t.Run("GetSessionByID", func(t *testing.T) {
		// Test with function provided
		expectedSessionID := uuid.New()
		expectedUserID := uuid.New()
		expectedSession := &database.Session{
			ID:     expectedSessionID,
			UserID: expectedUserID,
			Name:   "test session",
		}
		expectedError := errors.New("test error")
		
		mockDB := &MockDB{
			GetSessionByIDFunc: func(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*database.Session, error) {
				// Verify the parameters passed to the function match what we expect
				assert.Equal(t, expectedSessionID, id, "Expected matching session ID")
				assert.Equal(t, expectedUserID, userID, "Expected matching user ID")
				return expectedSession, expectedError
			},
		}
		
		// Call the method and verify it returns what we expect
		session, err := mockDB.GetSessionByID(context.Background(), expectedSessionID, expectedUserID)
		assert.Equal(t, expectedSession, session, "Expected session to match")
		assert.Equal(t, expectedError, err, "Expected error to match")
		
		// Test with no function provided
		mockDB = &MockDB{}
		session, err = mockDB.GetSessionByID(context.Background(), expectedSessionID, expectedUserID)
		assert.Equal(t, &database.Session{}, session, "Expected empty session")
		assert.Nil(t, err, "Expected nil error")
	})
	
	t.Run("StopSession", func(t *testing.T) {
		// Test with function provided
		expectedSessionID := uuid.New()
		expectedUserID := uuid.New()
		now := time.Now()
		expectedSession := &database.Session{
			ID:        expectedSessionID,
			UserID:    expectedUserID,
			Name:      "test session",
			StoppedAt: sql.NullTime{Valid: true, Time: now},
		}
		expectedError := errors.New("test error")
		
		mockDB := &MockDB{
			StopSessionFunc: func(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*database.Session, error) {
				// Verify the parameters passed to the function match what we expect
				assert.Equal(t, expectedSessionID, id, "Expected matching session ID")
				assert.Equal(t, expectedUserID, userID, "Expected matching user ID")
				return expectedSession, expectedError
			},
		}
		
		// Call the method and verify it returns what we expect
		session, err := mockDB.StopSession(context.Background(), expectedSessionID, expectedUserID)
		assert.Equal(t, expectedSession, session, "Expected session to match")
		assert.Equal(t, expectedError, err, "Expected error to match")
		
		// Test with no function provided
		mockDB = &MockDB{}
		session, err = mockDB.StopSession(context.Background(), expectedSessionID, expectedUserID)
		assert.Equal(t, &database.Session{}, session, "Expected empty session")
		assert.Nil(t, err, "Expected nil error")
	})
	
	t.Run("DeleteSession", func(t *testing.T) {
		// Test with function provided
		expectedSessionID := uuid.New()
		expectedUserID := uuid.New()
		expectedError := errors.New("test error")
		
		mockDB := &MockDB{
			DeleteSessionFunc: func(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
				// Verify the parameters passed to the function match what we expect
				assert.Equal(t, expectedSessionID, id, "Expected matching session ID")
				assert.Equal(t, expectedUserID, userID, "Expected matching user ID")
				return expectedError
			},
		}
		
		// Call the method and verify it returns what we expect
		err := mockDB.DeleteSession(context.Background(), expectedSessionID, expectedUserID)
		assert.Equal(t, expectedError, err, "Expected error to match")
		
		// Test with no function provided
		mockDB = &MockDB{}
		err = mockDB.DeleteSession(context.Background(), expectedSessionID, expectedUserID)
		assert.Nil(t, err, "Expected nil error")
	})
}