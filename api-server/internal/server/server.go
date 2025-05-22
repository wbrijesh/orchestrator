package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"
	"github.com/newrelic/go-agent/v3/newrelic"

	"api-server/internal/database"
)

type Server struct {
	port int
	db   database.Service
	nrApp *newrelic.Application // New Relic application
	*http.Server
}

// NewServerFunc defines function type for server creation
type NewServerFunc func() *http.Server

// getEnvOrDefaultString gets environment string variables with default values
func getEnvOrDefaultString(key, defaultVal string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return defaultVal
}

// NewServer creates a new HTTP server with the given database service
func NewServer(db database.Service) (*Server, error) {
	if db == nil {
		var err error
		db, err = database.New()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize database for server: %w", err)
		}
	}
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	// Initialize New Relic
	nrApp, err := newrelic.NewApplication(
		newrelic.ConfigAppName(getEnvOrDefaultString("NEW_RELIC_APP_NAME", "Orchestrator API")),
		newrelic.ConfigLicense("eu01xxe1e5cc9a5442065e9bc419758bFFFFNRAL"),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	if err != nil {
		log.Printf("Warning: failed to initialize New Relic: %v", err)
		// Decide if this should be a fatal error or just a warning
		// For now, continuing without New Relic if it fails
		nrApp = nil // Ensure nrApp is nil if initialization failed
	}

	// Wrap database service with New Relic instrumentation if available
	if nrApp != nil {
		db = &DatabaseInstrumentation{
			db:    db,
			nrApp: nrApp,
		}
	}

	s := &Server{
		port:  port,
		db:    db,
		nrApp: nrApp,
	}

	// Set up the HTTP server
	s.Server = &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", port),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	s.Server.Handler = s.RegisterRoutes()

	return s, nil
}
