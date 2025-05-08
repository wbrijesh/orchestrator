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

	db    database.Service
	nrApp *newrelic.Application // New Relic application
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

// NewServer is the default implementation for creating a new HTTP server
var NewServer = func() *http.Server {
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
		// Continue without New Relic if it fails to initialize
		nrApp = nil
	}

	// Create database service with New Relic instrumentation if available
	dbService := database.New()
	if nrApp != nil {
		dbService = &DatabaseInstrumentation{
			db:    dbService,
			nrApp: nrApp,
		}
	}

	NewServer := &Server{
		port:  port,
		db:    dbService,
		nrApp: nrApp,
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", NewServer.port), // Listen on all interfaces
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
