package auth

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	_ "github.com/joho/godotenv/autoload"
)

var (
	secretKey = []byte(os.Getenv("JWT_SECRET"))
	expiryDur = parseExpiry(os.Getenv("JWT_EXPIRY"))
)

func parseExpiry(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 24 * time.Hour
	}
	return d
}

// TokenGeneratorFunc is a function type for token generation
type TokenGeneratorFunc func(userID string) (string, error) 

// GenerateToken creates a JWT containing the user ID.
var GenerateToken TokenGeneratorFunc = func(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiryDur)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// TokenValidatorFunc is a function type for token validation
type TokenValidatorFunc func(tkn string) (*jwt.RegisteredClaims, error)

// ValidateToken parses and validates the token string.
var ValidateToken TokenValidatorFunc = func(tkn string) (*jwt.RegisteredClaims, error) {
	parsed, err := jwt.ParseWithClaims(tkn, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := parsed.Claims.(*jwt.RegisteredClaims); ok && parsed.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}