package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/Nico-Csk/socialflow/internal/domain"
)

// CreateUser inserts a new user and returns the fully populated domain object.
// The caller is responsible for hashing the password beforehand.
func (s *Store) CreateUser(ctx context.Context, db DB, email, passwordHash, name string) (*domain.User, error) {
	u := &domain.User{}
	err := db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name)
		 VALUES ($1, $2, $3)
		 RETURNING id, email, password_hash, name, created_at, updated_at`,
		email, passwordHash, name,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// GetUserByEmail looks up a user by their unique email address.
func (s *Store) GetUserByEmail(ctx context.Context, db DB, email string) (*domain.User, error) {
	u := &domain.User{}
	err := db.QueryRow(ctx,
		`SELECT id, email, password_hash, name, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

// GetUserByID looks up a user by primary key.
func (s *Store) GetUserByID(ctx context.Context, db DB, id string) (*domain.User, error) {
	u := &domain.User{}
	err := db.QueryRow(ctx,
		`SELECT id, email, password_hash, name, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}
