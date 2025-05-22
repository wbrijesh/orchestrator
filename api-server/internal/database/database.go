package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

// Service represents a service that interacts with a database.
type Service interface {
	// DB returns the underlying sql.DB connection
	DB() *sql.DB

	// Health returns a map of health status information.
	// The keys and values in the map are service-specific.
	Health() map[string]string

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error

	// User methods
	CreateUser(ctx context.Context, u *User) (uuid.UUID, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	
	// Session methods
	CreateSession(ctx context.Context, userID uuid.UUID, name string, browserID, browserType, cdpURL string, headless bool, viewportW, viewportH int, userAgent *string) (*Session, error)
	GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*Session, error)
	GetSessionByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Session, error)
	StopSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Session, error)
	DeleteSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

type service struct {
	db *sql.DB
}

// DB returns the underlying sql.DB connection
func (s *service) DB() *sql.DB {
	return s.db
}

var (
	dbInstance *service
)

// New creates a new database service using environment variables for configuration.
func New() (Service, error) {
	// Reuse existing connection if already initialised
	if dbInstance != nil {
		return dbInstance, nil
	}

	// Read environment variables at call-time (not package init) so tests can
	// inject values dynamically.
	dbName := os.Getenv("DB_DATABASE")
	pwd := os.Getenv("DB_PASSWORD")
	user := os.Getenv("DB_USERNAME")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	schemaName := os.Getenv("DB_SCHEMA")

	// Provide sensible default for search_path when not supplied
	if schemaName == "" {
		schemaName = "public"
	}

	// Basic sanity check
	if host == "" || port == "" || user == "" || pwd == "" || dbName == "" {
		return nil, fmt.Errorf("database environment variables not set")
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s",
		user, pwd, host, port, dbName, schemaName)

	svc, err := NewWithDSN(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create database service: %w", err)
	}

	dbInstance = svc
	return dbInstance, nil
}

// NewWithDSN creates a new database service with the provided DSN.
// It opens a connection to the database and verifies it's working.
func NewWithDSN(dsn string) (*service, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Verify the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &service{db: db}, nil
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Printf("db down: %v", err) // Log the error but don't terminate
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	log.Printf("Disconnected from database: %s", os.Getenv("DB_DATABASE"))
	return s.db.Close()
}
