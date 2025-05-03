package database

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func mustStartPostgresContainer() (func(context.Context, ...testcontainers.TerminateOption) error, error) {
	var (
		dbName = "database"
		dbPwd  = "password"
		dbUser = "user"
	)

	dbContainer, err := postgres.Run(
		context.Background(),
		"postgres:latest",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPwd),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, err
	}

	database = dbName
	password = dbPwd
	username = dbUser

	dbHost, err := dbContainer.Host(context.Background())
	if err != nil {
		return dbContainer.Terminate, err
	}

	dbPort, err := dbContainer.MappedPort(context.Background(), "5432/tcp")
	if err != nil {
		return dbContainer.Terminate, err
	}

	host = dbHost
	port = dbPort.Port()

	return dbContainer.Terminate, err
}

func TestMain(m *testing.M) {
	teardown, err := mustStartPostgresContainer()
	if err != nil {
		log.Fatalf("could not start postgres container: %v", err)
	}

	m.Run()

	if teardown != nil && teardown(context.Background()) != nil {
		log.Fatalf("could not teardown postgres container: %v", err)
	}
}

func TestNew(t *testing.T) {
	srv := New()
	if srv == nil {
		t.Fatal("New() returned nil")
	}
}

func TestHealth(t *testing.T) {
	srv := New()

	stats := srv.Health()

	// Basic health checks
	if stats["status"] != "up" {
		t.Fatalf("expected status to be up, got %s", stats["status"])
	}

	if _, ok := stats["error"]; ok {
		t.Fatalf("expected error not to be present")
	}

	if stats["message"] != "It's healthy" {
		t.Fatalf("expected message to be 'It's healthy', got %s", stats["message"])
	}
	
	// Check for presence of DB stats
	expectedStats := []string{
		"open_connections",
		"in_use",
		"idle",
		"wait_count",
		"wait_duration",
		"max_idle_closed",
		"max_lifetime_closed",
	}
	
	for _, stat := range expectedStats {
		if _, ok := stats[stat]; !ok {
			t.Errorf("expected %s stat to be present", stat)
		}
	}
}

func TestClose(t *testing.T) {
	srv := New()

	if srv.Close() != nil {
		t.Fatalf("expected Close() to return nil")
	}
}

// TestDatabaseFailure tests health reporting when database is down
func TestDatabaseFailure(t *testing.T) {
	// Save the original dbInstance
	originalInstance := dbInstance
	defer func() {
		// Restore the original dbInstance after test
		dbInstance = originalInstance
	}()
	
	// Create a service with a closed connection to simulate failure
	srv := New()
	s := srv.(*service)
	
	// Close the database connection
	s.db.Close()
	
	// Now check health - should report down status
	stats := s.Health()
	
	if stats["status"] != "down" {
		t.Errorf("expected status to be down when database is closed, got %s", stats["status"])
	}
	
	if _, ok := stats["error"]; !ok {
		t.Errorf("expected error to be present when database is down")
	}
	
	// Reset dbInstance to nil so next call to New() creates a fresh connection
	dbInstance = nil
}
