package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Session represents a user session
type Session struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Name      string
	StartedAt time.Time
	StoppedAt sql.NullTime
	// Browser-specific fields
	BrowserID   string
	BrowserType string
	CdpURL      string
	Headless    bool
	ViewportW   int
	ViewportH   int
	UserAgent   sql.NullString
}

// SessionView is the public representation of a Session
type SessionView struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	StartedAt time.Time `json:"started_at"`
	StoppedAt *time.Time `json:"stopped_at"`
	Active    bool      `json:"active"`
	Duration  *string   `json:"duration"`
	// Browser details
	BrowserID   string  `json:"browser_id"`
	BrowserType string  `json:"browser_type"`
	CdpURL      string  `json:"cdp_url"`
	Headless    bool    `json:"headless"`
	ViewportW   int     `json:"viewport_width"`
	ViewportH   int     `json:"viewport_height"`
	UserAgent   *string `json:"user_agent,omitempty"`
}

// CreateSession inserts a new session
func (s *service) CreateSession(ctx context.Context, userID uuid.UUID, name string, browserID, browserType, cdpURL string, headless bool, viewportW, viewportH int, userAgent *string) (*Session, error) {
	q := `
		INSERT INTO sessions (
			user_id, name, browser_id, browser_type, cdp_url, 
			headless, viewport_w, viewport_h, user_agent
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, name, started_at, stopped_at, browser_id, 
		browser_type, cdp_url, headless, viewport_w, viewport_h, user_agent
	`
	session := &Session{
		UserID:      userID,
		Name:        name,
		BrowserID:   browserID,
		BrowserType: browserType,
		CdpURL:      cdpURL,
		Headless:    headless,
		ViewportW:   viewportW,
		ViewportH:   viewportH,
	}
	
	// Set user agent if provided
	if userAgent != nil {
		session.UserAgent = sql.NullString{String: *userAgent, Valid: true}
	}

	row := s.db.QueryRowContext(ctx, q, 
		userID, name, browserID, browserType, cdpURL, 
		headless, viewportW, viewportH, 
		sql.NullString{String: session.UserAgent.String, Valid: session.UserAgent.Valid},
	)
	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.Name,
		&session.StartedAt,
		&session.StoppedAt,
		&session.BrowserID,
		&session.BrowserType,
		&session.CdpURL,
		&session.Headless,
		&session.ViewportW,
		&session.ViewportH,
		&session.UserAgent,
	)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetSessionsByUserID retrieves all sessions for a specific user
func (s *service) GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]*Session, error) {
	q := `
		SELECT id, user_id, name, started_at, stopped_at, 
		       browser_id, browser_type, cdp_url, headless, 
		       viewport_w, viewport_h, user_agent
		FROM sessions
		WHERE user_id = $1
		ORDER BY started_at DESC
	`

	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []*Session
	for rows.Next() {
		session := &Session{}
		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.Name,
			&session.StartedAt,
			&session.StoppedAt,
			&session.BrowserID,
			&session.BrowserType,
			&session.CdpURL,
			&session.Headless,
			&session.ViewportW,
			&session.ViewportH,
			&session.UserAgent,
		)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return sessions, nil
}

// GetSessionByID retrieves a specific session
func (s *service) GetSessionByID(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Session, error) {
	q := `
		SELECT id, user_id, name, started_at, stopped_at, 
		       browser_id, browser_type, cdp_url, headless, 
		       viewport_w, viewport_h, user_agent
		FROM sessions
		WHERE id = $1 AND user_id = $2
	`

	session := &Session{}
	row := s.db.QueryRowContext(ctx, q, id, userID)
	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.Name,
		&session.StartedAt,
		&session.StoppedAt,
		&session.BrowserID,
		&session.BrowserType,
		&session.CdpURL,
		&session.Headless,
		&session.ViewportW,
		&session.ViewportH,
		&session.UserAgent,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("session not found")
		}
		return nil, err
	}

	return session, nil
}

// StopSession updates a session by setting its stopped_at time
func (s *service) StopSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*Session, error) {
	q := `
		UPDATE sessions
		SET stopped_at = NOW()
		WHERE id = $1 AND user_id = $2 AND stopped_at IS NULL
		RETURNING id, user_id, name, started_at, stopped_at, 
		          browser_id, browser_type, cdp_url, headless, 
		          viewport_w, viewport_h, user_agent
	`

	session := &Session{}
	row := s.db.QueryRowContext(ctx, q, id, userID)
	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.Name,
		&session.StartedAt,
		&session.StoppedAt,
		&session.BrowserID,
		&session.BrowserType,
		&session.CdpURL,
		&session.Headless,
		&session.ViewportW,
		&session.ViewportH,
		&session.UserAgent,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("session not found or already stopped")
		}
		return nil, err
	}

	return session, nil
}

// DeleteSession permanently removes a session
func (s *service) DeleteSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	q := `
		DELETE FROM sessions
		WHERE id = $1 AND user_id = $2
	`

	result, err := s.db.ExecContext(ctx, q, id, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("session not found or already deleted")
	}

	return nil
}

// ToView converts a Session to a SessionView
func (s *Session) ToView() *SessionView {
	view := &SessionView{
		ID:          s.ID.String(),
		UserID:      s.UserID.String(),
		Name:        s.Name,
		StartedAt:   s.StartedAt,
		Active:      !s.StoppedAt.Valid,
		BrowserID:   s.BrowserID,
		BrowserType: s.BrowserType,
		CdpURL:      s.CdpURL,
		Headless:    s.Headless,
		ViewportW:   s.ViewportW,
		ViewportH:   s.ViewportH,
	}

	if s.StoppedAt.Valid {
		stoppedAt := s.StoppedAt.Time
		view.StoppedAt = &stoppedAt
		
		// Calculate duration
		duration := stoppedAt.Sub(s.StartedAt)
		durationStr := formatDuration(duration)
		view.Duration = &durationStr
	}
	
	// Set user agent if valid
	if s.UserAgent.Valid {
		userAgent := s.UserAgent.String
		view.UserAgent = &userAgent
	}

	return view
}

// formatDuration formats a duration in a human-readable way
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	
	if h > 0 {
		return formatTimePart(h, "hour") + ", " + formatTimePart(m, "minute")
	}
	if m > 0 {
		return formatTimePart(m, "minute") + ", " + formatTimePart(s, "second")
	}
	return formatTimePart(s, "second")
}

// formatTimePart formats a time part (hours, minutes, seconds)
func formatTimePart(value time.Duration, unit string) string {
	if value == 1 {
		return "1 " + unit
	}
	return fmt.Sprintf("%d %s", value, unit+"s")
}