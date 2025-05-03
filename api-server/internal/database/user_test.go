package database

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

// setupUserTable creates the users table for testing
func setupUserTable(t *testing.T, s *service) {
	ctx := context.Background()
	
	// Create schema (use public for testing)
	schema = "public"
	
	// First ensure we can connect to the database
	if err := s.db.PingContext(ctx); err != nil {
		t.Skipf("Skipping test: database connection failed: %v", err)
		return
	}
	
	// Set search_path explicitly
	_, err := s.db.ExecContext(ctx, `SET search_path TO public`)
	if err != nil {
		t.Fatalf("failed to set search path: %v", err)
	}
	
	// Create users table with UUID extension
	_, err = s.db.ExecContext(ctx, `CREATE EXTENSION IF NOT EXISTS pgcrypto`)
	if err != nil {
		t.Fatalf("failed to create pgcrypto extension: %v", err)
	}
	
	// Create users table
	_, err = s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			email VARCHAR(255) UNIQUE NOT NULL,
			first_name VARCHAR(255) NOT NULL,
			last_name VARCHAR(255) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("failed to create users table: %v", err)
	}
}

func TestCreateUser(t *testing.T) {
	// Reset dbInstance to get fresh connection
	originalInstance := dbInstance
	dbInstance = nil
	defer func() {
		dbInstance = originalInstance
	}()
	
	srv := New()
	s, ok := srv.(*service)
	if !ok {
		t.Fatal("expected srv to be of type *service")
	}
	
	// Setup test table
	setupUserTable(t, s)
	
	// Create test user
	ctx := context.Background()
	testUser := &User{
		Email:        "test@example.com",
		FirstName:    "Test",
		LastName:     "User",
		PasswordHash: "hashedpassword",
	}
	
	// Test user creation
	id, err := s.CreateUser(ctx, testUser)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	
	// Verify ID is returned
	if id == uuid.Nil {
		t.Fatal("expected non-nil UUID")
	}
	
	// Verify CreatedAt and UpdatedAt are set
	if testUser.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if testUser.UpdatedAt.IsZero() {
		t.Error("expected UpdatedAt to be set")
	}
	
	// Test duplicate email error
	duplicateUser := &User{
		Email:        "test@example.com", // Same email
		FirstName:    "Another",
		LastName:     "User",
		PasswordHash: "differenthash",
	}
	
	_, err = s.CreateUser(ctx, duplicateUser)
	if err == nil {
		t.Fatal("expected error when creating user with duplicate email")
	}
}

func TestGetUserByEmail(t *testing.T) {
	// Reset dbInstance to get fresh connection
	originalInstance := dbInstance
	dbInstance = nil
	defer func() {
		dbInstance = originalInstance
	}()
	
	srv := New()
	s, ok := srv.(*service)
	if !ok {
		t.Fatal("expected srv to be of type *service")
	}
	
	// Setup test table
	setupUserTable(t, s)
	
	// Create a user first
	ctx := context.Background()
	expectedUser := &User{
		Email:        "find@example.com",
		FirstName:    "Find",
		LastName:     "Me",
		PasswordHash: "findmepassword",
	}
	
	id, err := s.CreateUser(ctx, expectedUser)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	expectedUser.ID = id
	
	// Test finding the user
	foundUser, err := s.GetUserByEmail(ctx, "find@example.com")
	if err != nil {
		t.Fatalf("failed to get user by email: %v", err)
	}
	
	// Verify the user data
	if foundUser.ID != id {
		t.Errorf("expected ID %v, got %v", id, foundUser.ID)
	}
	if foundUser.Email != expectedUser.Email {
		t.Errorf("expected Email %s, got %s", expectedUser.Email, foundUser.Email)
	}
	if foundUser.FirstName != expectedUser.FirstName {
		t.Errorf("expected FirstName %s, got %s", expectedUser.FirstName, foundUser.FirstName)
	}
	if foundUser.LastName != expectedUser.LastName {
		t.Errorf("expected LastName %s, got %s", expectedUser.LastName, foundUser.LastName)
	}
	if foundUser.PasswordHash != expectedUser.PasswordHash {
		t.Errorf("expected PasswordHash %s, got %s", expectedUser.PasswordHash, foundUser.PasswordHash)
	}
	
	// Test non-existent user
	_, err = s.GetUserByEmail(ctx, "nonexistent@example.com")
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound for non-existent user, got %v", err)
	}
}

func TestUserOperationsSequence(t *testing.T) {
	// Reset dbInstance to get fresh connection
	originalInstance := dbInstance
	dbInstance = nil
	defer func() {
		dbInstance = originalInstance
	}()
	
	srv := New()
	s, ok := srv.(*service)
	if !ok {
		t.Fatal("expected srv to be of type *service")
	}
	
	// Setup test table
	setupUserTable(t, s)
	
	ctx := context.Background()
	
	// Create multiple users and verify each one
	for i, userData := range []struct {
		email     string
		firstName string
		lastName  string
	}{
		{"user1@example.com", "User", "One"},
		{"user2@example.com", "User", "Two"},
		{"user3@example.com", "User", "Three"},
	} {
		user := &User{
			Email:        userData.email,
			FirstName:    userData.firstName,
			LastName:     userData.lastName,
			PasswordHash: "password123",
		}
		
		// Create the user
		id, err := s.CreateUser(ctx, user)
		if err != nil {
			t.Fatalf("failed to create user %d: %v", i, err)
		}
		
		// Retrieve the user by email
		retrieved, err := s.GetUserByEmail(ctx, userData.email)
		if err != nil {
			t.Fatalf("failed to get user %d by email: %v", i, err)
		}
		
		// Verify ID matches
		if retrieved.ID != id {
			t.Errorf("user %d: expected ID %v, got %v", i, id, retrieved.ID)
		}
		
		// Verify other fields
		if retrieved.Email != userData.email {
			t.Errorf("user %d: expected Email %s, got %s", i, userData.email, retrieved.Email)
		}
		if retrieved.FirstName != userData.firstName {
			t.Errorf("user %d: expected FirstName %s, got %s", i, userData.firstName, retrieved.FirstName)
		}
		if retrieved.LastName != userData.lastName {
			t.Errorf("user %d: expected LastName %s, got %s", i, userData.lastName, retrieved.LastName)
		}
	}
}