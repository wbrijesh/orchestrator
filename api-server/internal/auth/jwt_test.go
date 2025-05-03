package auth

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

func TestGenerateAndValidateToken(t *testing.T) {
	// Save original env vars
	originalSecret := os.Getenv("JWT_SECRET")
	originalExpiry := os.Getenv("JWT_EXPIRY")
	
	// Set test values
	os.Setenv("JWT_SECRET", "test-secret-key")
	os.Setenv("JWT_EXPIRY", "1h")
	
	// Reset secretKey and expiryDur for testing
	secretKey = []byte(os.Getenv("JWT_SECRET"))
	expiryDur = parseExpiry(os.Getenv("JWT_EXPIRY"))
	
	// Generate a token
	userID := "user123"
	token, err := GenerateToken(userID)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	
	// Validate the token
	claims, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}
	
	// Check if the claims contain the expected user ID
	if claims.Subject != userID {
		t.Errorf("Expected userID %s, got %s", userID, claims.Subject)
	}
	
	// Restore original env vars
	if originalSecret != "" {
		os.Setenv("JWT_SECRET", originalSecret)
	} else {
		os.Unsetenv("JWT_SECRET")
	}
	
	if originalExpiry != "" {
		os.Setenv("JWT_EXPIRY", originalExpiry)
	} else {
		os.Unsetenv("JWT_EXPIRY")
	}
}

func TestParseExpiry(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"1h", 1 * time.Hour},
		{"30m", 30 * time.Minute},
		{"invalid", 24 * time.Hour}, // Default case
		{"", 24 * time.Hour},        // Default case
	}
	
	for _, test := range tests {
		result := parseExpiry(test.input)
		if result != test.expected {
			t.Errorf("parseExpiry(%s) = %v; want %v", test.input, result, test.expected)
		}
	}
}

func TestInvalidToken(t *testing.T) {
	// Set up test env
	os.Setenv("JWT_SECRET", "test-secret-key")
	secretKey = []byte(os.Getenv("JWT_SECRET"))
	
	// Test invalid token
	invalidToken := "invalid.token.string"
	_, err := ValidateToken(invalidToken)
	if err == nil {
		t.Error("Expected error for invalid token, got nil")
	}
	
	// Test expired token
	claims := jwt.RegisteredClaims{
		Subject:   "user123",
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
		IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	expiredToken, _ := token.SignedString(secretKey)
	
	_, err = ValidateToken(expiredToken)
	if err == nil {
		t.Error("Expected error for expired token, got nil")
	}
}