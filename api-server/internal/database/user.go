package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

// ErrUserNotFound is returned when no user is found.
var ErrUserNotFound = errors.New("user not found")

// CreateUser inserts a new user. It returns the generated ID.
func (s *service) CreateUser(ctx context.Context, u *User) (uuid.UUID, error) {
	q := `
    INSERT INTO users
      (email, first_name, last_name, password_hash)
    VALUES ($1,$2,$3,$4)
    RETURNING id, created_at, updated_at
  `
	row := s.db.QueryRowContext(ctx, q,
		u.Email, u.FirstName, u.LastName, u.PasswordHash,
	)
	var id uuid.UUID
	if err := row.Scan(&id, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// GetUserByEmail loads a user by email.
func (s *service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	q := `
    SELECT id, email, first_name, last_name, password_hash,
           created_at, updated_at
    FROM users
    WHERE email = $1
  `
	u := &User{}
	row := s.db.QueryRowContext(ctx, q, email)
	if err := row.Scan(
		&u.ID, &u.Email, &u.FirstName, &u.LastName,
		&u.PasswordHash, &u.CreatedAt, &u.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}
