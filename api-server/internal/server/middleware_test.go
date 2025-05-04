package server

import (
	"api-server/internal/auth"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	// Create a test server
	s := &Server{}

	// Create a mock handler that will be wrapped by the middleware
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract userID from context to verify it was set correctly
		userID, err := GetUserIDFromContext(r.Context())
		if err != nil {
			http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
			return
		}
		// Write the user ID to the response for testing
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(userID.String()))
	})

	// Wrap the mock handler with the auth middleware
	handler := s.AuthMiddleware(mockHandler)

	// Set up test cases
	testCases := []struct {
		name           string
		setupRequest   func() *http.Request
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "Valid token",
			setupRequest: func() *http.Request {
				// Create a valid user ID
				userID := uuid.New()

				// Store the original token validator
				originalValidator := auth.ValidateToken

				// Replace with a mock validator that always returns valid claims
				auth.ValidateToken = func(tkn string) (*jwt.RegisteredClaims, error) {
					return &jwt.RegisteredClaims{
						Subject: userID.String(),
					}, nil
				}

				// Restore the original validator after the test
				t.Cleanup(func() {
					auth.ValidateToken = originalValidator
				})

				// Create request with Authorization header
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer valid-token-doesnt-matter")
				
				// Store the user ID in the request for later verification
				req = req.WithContext(context.WithValue(req.Context(), "expected_user_id", userID))
				return req
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "", // Will be filled with the user ID
		},
		{
			name: "Missing Authorization header",
			setupRequest: func() *http.Request {
				return httptest.NewRequest("GET", "/test", nil)
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Authorization header required\n",
		},
		{
			name: "Invalid Authorization format",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "InvalidFormat")
				return req
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Authorization header must be in the format 'Bearer {token}'\n",
		},
		{
			name: "Invalid token",
			setupRequest: func() *http.Request {
				// Store the original token validator
				originalValidator := auth.ValidateToken

				// Replace with a mock validator that returns an error
				auth.ValidateToken = func(tkn string) (*jwt.RegisteredClaims, error) {
					return nil, jwt.ErrTokenInvalidClaims
				}

				// Restore the original validator after the test
				t.Cleanup(func() {
					auth.ValidateToken = originalValidator
				})

				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer invalidtoken")
				return req
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid token: ",
		},
		{
			name: "Invalid user ID in token",
			setupRequest: func() *http.Request {
				// Store the original token validator
				originalValidator := auth.ValidateToken

				// Replace with a mock validator that returns an invalid user ID
				auth.ValidateToken = func(tkn string) (*jwt.RegisteredClaims, error) {
					return &jwt.RegisteredClaims{
						Subject: "not-a-valid-uuid",
					}, nil
				}

				// Restore the original validator after the test
				t.Cleanup(func() {
					auth.ValidateToken = originalValidator
				})

				req := httptest.NewRequest("GET", "/test", nil)
				req.Header.Set("Authorization", "Bearer dummytoken")
				return req
			},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Invalid user ID in token\n",
		},
	}

	// Run test cases
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a response recorder
			rec := httptest.NewRecorder()

			// Set up the request
			req := tc.setupRequest()

			// Call the handler
			handler.ServeHTTP(rec, req)

			// Check the status code
			assert.Equal(t, tc.expectedStatus, rec.Code)

			// For the valid token case, we need to check that the user ID matches
			if tc.name == "Valid token" {
				// Get the expected user ID from the context
				expectedUserID, ok := req.Context().Value("expected_user_id").(uuid.UUID)
				if ok {
					assert.Equal(t, expectedUserID.String(), rec.Body.String())
				} else {
					// If we can't extract the expected user ID, just check that the response is not empty
					assert.NotEmpty(t, rec.Body.String())
				}
			} else {
				// For other cases, check that the response body contains the expected error message
				assert.Contains(t, rec.Body.String(), tc.expectedBody)
			}
		})
	}
}

func TestGetUserIDFromContext(t *testing.T) {
	// Test with valid user ID in context
	t.Run("Valid user ID in context", func(t *testing.T) {
		expectedUserID := uuid.New()
		ctx := context.WithValue(context.Background(), userIDContextKey, expectedUserID)

		userID, err := GetUserIDFromContext(ctx)
		assert.NoError(t, err)
		assert.Equal(t, expectedUserID, userID)
	})

	// Test with missing user ID in context
	t.Run("Missing user ID in context", func(t *testing.T) {
		ctx := context.Background()

		userID, err := GetUserIDFromContext(ctx)
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, userID)
		assert.Contains(t, err.Error(), "user ID not found in context")
	})

	// Test with invalid type in context
	t.Run("Invalid type in context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDContextKey, "not-a-uuid")

		userID, err := GetUserIDFromContext(ctx)
		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, userID)
		assert.Contains(t, err.Error(), "user ID not found in context")
	})
}
