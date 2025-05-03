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

// GenerateToken creates a JWT containing the user ID.
func GenerateToken(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiryDur)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// ValidateToken parses and validates the token string.
func ValidateToken(tkn string) (*jwt.RegisteredClaims, error) {
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
