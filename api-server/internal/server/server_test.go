package server

import (
	"os"
	"testing"

	_ "github.com/joho/godotenv/autoload"
)

func TestNewServer(t *testing.T) {
	// Save original PORT env var
	originalPort := os.Getenv("PORT")
	defer func() {
		if originalPort != "" {
			os.Setenv("PORT", originalPort)
		} else {
			os.Unsetenv("PORT")
		}
	}()

	// Test with valid port
	os.Setenv("PORT", "8080")
	server := NewServer()

	if server == nil {
		t.Fatal("Expected non-nil server")
	}

	if server.Addr != "0.0.0.0:8080" {
		t.Errorf("Expected server address to be 0.0.0.0:8080, got %s", server.Addr)
	}

	// Check expected timeouts
	if server.IdleTimeout != 60*1000*1000*1000 { // 1 minute in nanoseconds
		t.Errorf("Unexpected IdleTimeout value")
	}

	if server.ReadTimeout != 10*1000*1000*1000 { // 10 seconds in nanoseconds
		t.Errorf("Unexpected ReadTimeout value")
	}

	if server.WriteTimeout != 30*1000*1000*1000 { // 30 seconds in nanoseconds
		t.Errorf("Unexpected WriteTimeout value")
	}

	// Test with invalid port (defaults to 0)
	os.Setenv("PORT", "invalid")
	server = NewServer()

	if server.Addr != "0.0.0.0:0" {
		t.Errorf("Expected server address to be 0.0.0.0:0 when PORT is invalid, got %s", server.Addr)
	}
}
