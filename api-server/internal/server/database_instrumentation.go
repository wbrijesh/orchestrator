package server

import (
	"api-server/internal/database"
	"context"

	"github.com/google/uuid"
	"github.com/newrelic/go-agent/v3/newrelic"
)

// DatabaseInstrumentation wraps a database.Service to provide New Relic instrumentation
type DatabaseInstrumentation struct {
	db    database.Service
	nrApp *newrelic.Application
}

// NewDatabaseInstrumentation creates a new database instrumentation layer
func NewDatabaseInstrumentation(db database.Service, nrApp *newrelic.Application) database.Service {
	if nrApp == nil {
		return db // If New Relic isn't available, just return the original DB
	}
	return &DatabaseInstrumentation{
		db:    db,
		nrApp: nrApp,
	}
}

// startSegment begins a database segment and returns the segment along with a callback to end it
func (d *DatabaseInstrumentation) startSegment(ctx context.Context, operation string) (*newrelic.DatastoreSegment, func()) {
	// Try to get transaction from context
	txn := newrelic.FromContext(ctx)
	if txn == nil {
		return nil, func() {} // No transaction, return no-op
	}

	segment := &newrelic.DatastoreSegment{
		Product:    newrelic.DatastorePostgres,
		Collection: "", // Would be table name if available
		Operation:  operation,
		StartTime:  txn.StartSegmentNow(),
	}

	return segment, func() { segment.End() }
}

// Health returns a map of health status information
func (d *DatabaseInstrumentation) Health() map[string]string {
	// Not adding New Relic instrumentation to health check as it's internal
	return d.db.Health()
}

// Close closes the database connection
func (d *DatabaseInstrumentation) Close() error {
	// Not adding New Relic instrumentation to close as it's internal
	return d.db.Close()
}

// User methods

// CreateUser creates a new user
func (d *DatabaseInstrumentation) CreateUser(ctx context.Context, u *database.User) (uuid.UUID, error) {
	segment, end := d.startSegment(ctx, "CreateUser")
	defer end()

	id, err := d.db.CreateUser(ctx, u)
	if segment != nil {
		segment.Collection = "users"
	}
	return id, err
}

// GetUserByEmail gets a user by email
func (d *DatabaseInstrumentation) GetUserByEmail(ctx context.Context, email string) (*database.User, error) {
	segment, end := d.startSegment(ctx, "GetUserByEmail")
	defer end()

	user, err := d.db.GetUserByEmail(ctx, email)
	if segment != nil {
		segment.Collection = "users"
	}
	return user, err
}

// Session methods

// CreateSession creates a new session
func (d *DatabaseInstrumentation) CreateSession(ctx context.Context, userID uuid.UUID, name string, browserID, browserType, cdpURL string, headless bool, viewportW, viewportH int, userAgent *string) (*database.Session, error) {
	segment, end := d.startSegment(ctx, "CreateSession")
	defer end()

	session, err := d.db.CreateSession(ctx, userID, name, browserID, browserType, cdpURL, headless, viewportW, viewportH, userAgent)
	if segment != nil {
		segment.Collection = "sessions"
	}
	return session, err
}

// GetSessionsByUserID gets sessions by user ID
func (d *DatabaseInstrumentation) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*database.Session, error) {
	segment, end := d.startSegment(ctx, "GetSessionsByUserID")
	defer end()

	sessions, err := d.db.GetSessionsByUserID(ctx, userID)
	if segment != nil {
		segment.Collection = "sessions"
	}
	return sessions, err
}

// GetSessionByID gets a specific session
func (d *DatabaseInstrumentation) GetSessionByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*database.Session, error) {
	segment, end := d.startSegment(ctx, "GetSessionByID")
	defer end()

	session, err := d.db.GetSessionByID(ctx, id, userID)
	if segment != nil {
		segment.Collection = "sessions"
	}
	return session, err
}

// StopSession stops a session
func (d *DatabaseInstrumentation) StopSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*database.Session, error) {
	segment, end := d.startSegment(ctx, "StopSession")
	defer end()

	session, err := d.db.StopSession(ctx, id, userID)
	if segment != nil {
		segment.Collection = "sessions"
	}
	return session, err
}

// DeleteSession deletes a session
func (d *DatabaseInstrumentation) DeleteSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	segment, end := d.startSegment(ctx, "DeleteSession")
	defer end()

	err := d.db.DeleteSession(ctx, id, userID)
	if segment != nil {
		segment.Collection = "sessions"
	}
	return err
}