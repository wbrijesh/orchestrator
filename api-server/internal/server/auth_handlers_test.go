package server

import (
	"api-server/internal/auth"
	"api-server/internal/database"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Using the MockDB from mock_db.go

// Save the original functions so we can restore them after tests
var originalGenerateToken = auth.GenerateToken

func restoreAuthFunctions() {
	auth.GenerateToken = originalGenerateToken
}

func TestHealthHandler(t *testing.T) {
	// Create a mock DB that returns a health map
	MockDB := &MockDB{
		HealthFunc: func() map[string]string {
			return map[string]string{
				"status":  "up",
				"message": "It's healthy",
			}
		},
	}

	s := &Server{db: MockDB}
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.healthHandler)
	handler.ServeHTTP(rr, req)

	// Check the status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	var response map[string]string
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "up" {
		t.Errorf("Expected status to be up, got %s", response["status"])
	}
}

func TestRegisterHandler(t *testing.T) {
	// Setup
	defer restoreAuthFunctions()

	testCases := []struct {
		name           string
		requestBody    interface{} // Using interface{} to test malformed JSON
		mockSetup      func() *MockDB
		tokenSetup     func()
		expectedStatus int
		expectedError  string
		checkData      func(t *testing.T, data interface{})
	}{
		{
			name: "Success",
			requestBody: database.AuthRequest{
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
				Password:  "password123",
			},
			mockSetup: func() *MockDB {
				return &MockDB{
					CreateUserFunc: func(ctx context.Context, u *database.User) (uuid.UUID, error) {
						id := uuid.New()
						return id, nil
					},
				}
			},
			tokenSetup: func() {
				auth.GenerateToken = func(userID string) (string, error) {
					return "valid.test.token", nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
			checkData: func(t *testing.T, data interface{}) {
				resp, ok := data.(map[string]interface{})
				if !ok {
					t.Errorf("Expected map response, got %T", data)
					return
				}
				if token, ok := resp["token"].(string); !ok || token != "valid.test.token" {
					t.Errorf("Expected token 'valid.test.token', got %v", token)
				}
				if user, ok := resp["user"].(map[string]interface{}); !ok {
					t.Error("Expected user object in response")
				} else {
					if email, ok := user["email"].(string); !ok || email != "test@example.com" {
						t.Errorf("Expected email 'test@example.com', got %v", email)
					}
				}
			},
		},
		{
			name:        "Invalid JSON",
			requestBody: `{"email": "bad-json"`, // Malformed JSON
			mockSetup: func() *MockDB {
				return &MockDB{}
			},
			tokenSetup:     func() {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "bad request",
			checkData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Error("Expected nil data for error response")
				}
			},
		},
		{
			name:        "Empty Request",
			requestBody: "",
			mockSetup: func() *MockDB {
				return &MockDB{}
			},
			tokenSetup:     func() {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "bad request",
			checkData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Error("Expected nil data for error response")
				}
			},
		},
		{
			name: "Password Hashing Error",
			requestBody: database.AuthRequest{
				Email: "test@example.com",
				// Create a very long password (>72 bytes) that will cause bcrypt to fail
				Password:  string(make([]byte, 100)),
				FirstName: "Test",
				LastName:  "User",
			},
			mockSetup: func() *MockDB {
				return &MockDB{
					// We need to set this even though it shouldn't be called
					CreateUserFunc: func(ctx context.Context, u *database.User) (uuid.UUID, error) {
						return uuid.Nil, nil
					},
				}
			},
			tokenSetup:     func() {},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "server error",
			checkData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Error("Expected nil data for error response")
				}
			},
		},
		{
			name: "Database Error",
			requestBody: database.AuthRequest{
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
				Password:  "password123",
			},
			mockSetup: func() *MockDB {
				return &MockDB{
					CreateUserFunc: func(ctx context.Context, u *database.User) (uuid.UUID, error) {
						return uuid.Nil, errors.New("database error")
					},
				}
			},
			tokenSetup:     func() {},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "could not create user",
			checkData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Error("Expected nil data for error response")
				}
			},
		},
		{
			name: "Duplicate Email",
			requestBody: database.AuthRequest{
				Email:     "existing@example.com",
				FirstName: "Test",
				LastName:  "User",
				Password:  "password123",
			},
			mockSetup: func() *MockDB {
				return &MockDB{
					CreateUserFunc: func(ctx context.Context, u *database.User) (uuid.UUID, error) {
						return uuid.Nil, errors.New("duplicate key value violates unique constraint")
					},
				}
			},
			tokenSetup:     func() {},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "email already exists",
			checkData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Error("Expected nil data for error response")
				}
			},
		},
		{
			name: "Token Generation Error",
			requestBody: database.AuthRequest{
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
				Password:  "password123",
			},
			mockSetup: func() *MockDB {
				return &MockDB{
					CreateUserFunc: func(ctx context.Context, u *database.User) (uuid.UUID, error) {
						id := uuid.New()
						return id, nil
					},
				}
			},
			tokenSetup: func() {
				auth.GenerateToken = func(userID string) (string, error) {
					return "", errors.New("token generation error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "could not sign token",
			checkData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Error("Expected nil data for error response")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			MockDB := tc.mockSetup()
			tc.tokenSetup()

			s := &Server{db: MockDB}

			// Prepare request
			var reqBody []byte
			var err error

			switch body := tc.requestBody.(type) {
			case string:
				reqBody = []byte(body)
			case database.AuthRequest:
				reqBody, err = json.Marshal(body)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			default:
				reqBody, err = json.Marshal(body)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req, err := http.NewRequest("POST", "/register", bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatal(err)
			}

			// Execute request
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(s.RegisterHandler)
			handler.ServeHTTP(rr, req)

			// Check status code
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectedStatus)
			}

			// Check response
			var response database.APIResponse
			if rr.Body.Len() > 0 {
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal response: %v", err)
					return
				}

				if response.Error != tc.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tc.expectedError, response.Error)
				}

				tc.checkData(t, response.Data)
			} else {
				t.Errorf("Expected non-empty response body")
			}
		})
	}
}

func TestLoginHandler(t *testing.T) {
	// Setup
	defer restoreAuthFunctions()

	// Create a real bcrypt hash for testing
	const testPassword = "password123"
	testHash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("Failed to generate password hash for testing: %v", err)
	}

	testCases := []struct {
		name           string
		requestBody    interface{} // Using interface{} to test malformed JSON
		mockSetup      func() *MockDB
		tokenSetup     func()
		expectedStatus int
		expectedError  string
		checkData      func(t *testing.T, data interface{})
	}{
		{
			name: "Success",
			requestBody: database.AuthRequest{
				Email:    "test@example.com",
				Password: testPassword,
			},
			mockSetup: func() *MockDB {
				return &MockDB{
					GetUserByEmailFunc: func(ctx context.Context, email string) (*database.User, error) {
						return &database.User{
							ID:           uuid.New(),
							Email:        email,
							FirstName:    "Test",
							LastName:     "User",
							PasswordHash: string(testHash),
						}, nil
					},
				}
			},
			tokenSetup: func() {
				auth.GenerateToken = func(userID string) (string, error) {
					return "valid.test.token", nil
				}
			},
			expectedStatus: http.StatusOK,
			expectedError:  "",
			checkData: func(t *testing.T, data interface{}) {
				resp, ok := data.(map[string]interface{})
				if !ok {
					t.Errorf("Expected map response, got %T", data)
					return
				}
				if token, ok := resp["token"].(string); !ok || token != "valid.test.token" {
					t.Errorf("Expected token 'valid.test.token', got %v", token)
				}
				if user, ok := resp["user"].(map[string]interface{}); !ok {
					t.Error("Expected user object in response")
				} else {
					if email, ok := user["email"].(string); !ok || email != "test@example.com" {
						t.Errorf("Expected email 'test@example.com', got %v", email)
					}
				}
			},
		},
		{
			name:        "Invalid JSON",
			requestBody: `{"email": "bad-json"`, // Malformed JSON
			mockSetup: func() *MockDB {
				return &MockDB{}
			},
			tokenSetup:     func() {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "bad request",
			checkData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Error("Expected nil data for error response")
				}
			},
		},
		{
			name:        "Empty Request",
			requestBody: "",
			mockSetup: func() *MockDB {
				return &MockDB{}
			},
			tokenSetup:     func() {},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "bad request",
			checkData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Error("Expected nil data for error response")
				}
			},
		},
		{
			name: "User Not Found",
			requestBody: database.AuthRequest{
				Email:    "nonexistent@example.com",
				Password: testPassword,
			},
			mockSetup: func() *MockDB {
				return &MockDB{
					GetUserByEmailFunc: func(ctx context.Context, email string) (*database.User, error) {
						return nil, database.ErrUserNotFound
					},
				}
			},
			tokenSetup:     func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid credentials",
			checkData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Error("Expected nil data for error response")
				}
			},
		},
		{
			name: "Database Error",
			requestBody: database.AuthRequest{
				Email:    "test@example.com",
				Password: testPassword,
			},
			mockSetup: func() *MockDB {
				return &MockDB{
					GetUserByEmailFunc: func(ctx context.Context, email string) (*database.User, error) {
						return nil, errors.New("database connection error")
					},
				}
			},
			tokenSetup:     func() {},
			expectedStatus: http.StatusUnauthorized, // Note: Currently all DB errors in Login return 401
			expectedError:  "invalid credentials",
			checkData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Error("Expected nil data for error response")
				}
			},
		},
		{
			name: "Invalid Password",
			requestBody: database.AuthRequest{
				Email:    "test@example.com",
				Password: "wrongpassword", // Doesn't match the hash
			},
			mockSetup: func() *MockDB {
				return &MockDB{
					GetUserByEmailFunc: func(ctx context.Context, email string) (*database.User, error) {
						return &database.User{
							ID:           uuid.New(),
							Email:        email,
							FirstName:    "Test",
							LastName:     "User",
							PasswordHash: string(testHash), // Hash for "password123"
						}, nil
					},
				}
			},
			tokenSetup:     func() {},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "invalid credentials",
			checkData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Error("Expected nil data for error response")
				}
			},
		},
		{
			name: "Token Generation Error",
			requestBody: database.AuthRequest{
				Email:    "test@example.com",
				Password: testPassword,
			},
			mockSetup: func() *MockDB {
				return &MockDB{
					GetUserByEmailFunc: func(ctx context.Context, email string) (*database.User, error) {
						return &database.User{
							ID:           uuid.New(),
							Email:        email,
							FirstName:    "Test",
							LastName:     "User",
							PasswordHash: string(testHash),
						}, nil
					},
				}
			},
			tokenSetup: func() {
				auth.GenerateToken = func(userID string) (string, error) {
					return "", errors.New("token generation error")
				}
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "could not sign token",
			checkData: func(t *testing.T, data interface{}) {
				if data != nil {
					t.Error("Expected nil data for error response")
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mocks
			MockDB := tc.mockSetup()
			tc.tokenSetup()

			s := &Server{db: MockDB}

			// Prepare request
			var reqBody []byte
			var err error

			switch body := tc.requestBody.(type) {
			case string:
				reqBody = []byte(body)
			case database.AuthRequest:
				reqBody, err = json.Marshal(body)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			default:
				reqBody, err = json.Marshal(body)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
			}

			req, err := http.NewRequest("POST", "/login", bytes.NewBuffer(reqBody))
			if err != nil {
				t.Fatal(err)
			}

			// Execute request
			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(s.LoginHandler)
			handler.ServeHTTP(rr, req)

			// Check status code
			if status := rr.Code; status != tc.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v", status, tc.expectedStatus)
			}

			// Check response
			var response database.APIResponse
			if rr.Body.Len() > 0 {
				err = json.Unmarshal(rr.Body.Bytes(), &response)
				if err != nil {
					t.Errorf("Failed to unmarshal response: %v\nBody: %s", err, rr.Body.String())
					return
				}

				if response.Error != tc.expectedError {
					t.Errorf("Expected error '%s', got '%s'", tc.expectedError, response.Error)
				}

				tc.checkData(t, response.Data)
			} else {
				t.Errorf("Expected non-empty response body")
			}
		})
	}
}

// Test for HelloWorldHandler
func TestHelloWorldHandler(t *testing.T) {
	s := &Server{}
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(s.HelloWorldHandler)
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check response body
	expected := `{"message":"Hello World"}`
	body, err := io.ReadAll(rr.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if string(body) != expected {
		t.Errorf("Expected response body '%s', got '%s'", expected, string(body))
	}
}
