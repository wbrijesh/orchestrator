package server

import (
	"api-server/internal/database"
	"context"

	"github.com/google/uuid"
)

// MockDB is a mock implementation of database.Service for testing
type MockDB struct {
	CreateUserFunc        func(ctx context.Context, u *database.User) (uuid.UUID, error)
	GetUserByEmailFunc    func(ctx context.Context, email string) (*database.User, error)
	HealthFunc            func() map[string]string
	CloseFunc             func() error
	CreateSessionFunc     func(ctx context.Context, userID uuid.UUID, name string) (*database.Session, error)
	GetSessionsByUserIDFunc func(ctx context.Context, userID uuid.UUID) ([]*database.Session, error)
	GetSessionByIDFunc    func(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*database.Session, error)
	StopSessionFunc       func(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*database.Session, error)
	DeleteSessionFunc     func(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
}

// User methods
func (m *MockDB) CreateUser(ctx context.Context, u *database.User) (uuid.UUID, error) {
	if m.CreateUserFunc != nil {
		return m.CreateUserFunc(ctx, u)
	}
	return uuid.UUID{}, nil
}

func (m *MockDB) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	if m.GetUserByEmailFunc != nil {
		return m.GetUserByEmailFunc(ctx, email)
	}
	return &database.User{}, nil
}

// Health and Close methods
func (m *MockDB) Health() map[string]string {
	if m.HealthFunc != nil {
		return m.HealthFunc()
	}
	return map[string]string{"status": "up"}
}

func (m *MockDB) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

// Session methods
func (m *MockDB) CreateSession(ctx context.Context, userID uuid.UUID, name string) (*database.Session, error) {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(ctx, userID, name)
	}
	return &database.Session{}, nil
}

func (m *MockDB) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*database.Session, error) {
	if m.GetSessionsByUserIDFunc != nil {
		return m.GetSessionsByUserIDFunc(ctx, userID)
	}
	return []*database.Session{}, nil
}

func (m *MockDB) GetSessionByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*database.Session, error) {
	if m.GetSessionByIDFunc != nil {
		return m.GetSessionByIDFunc(ctx, id, userID)
	}
	return &database.Session{}, nil
}

func (m *MockDB) StopSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*database.Session, error) {
	if m.StopSessionFunc != nil {
		return m.StopSessionFunc(ctx, id, userID)
	}
	return &database.Session{}, nil
}

func (m *MockDB) DeleteSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	if m.DeleteSessionFunc != nil {
		return m.DeleteSessionFunc(ctx, id, userID)
	}
	return nil
}