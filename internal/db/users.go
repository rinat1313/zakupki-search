package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/rinat1313/zakupki-search/internal/models"
)

var ErrNotFound = errors.New("not found")

type userRow struct {
	ID           string
	Login        string
	PasswordHash string
	DisplayName  string
}

func (s *Store) GetUserByLogin(ctx context.Context, login string) (userRow, error) {
	var u userRow
	err := s.Pool.QueryRow(ctx, `
		SELECT id::text, login, password_hash, display_name
		FROM users WHERE login = $1`, login,
	).Scan(&u.ID, &u.Login, &u.PasswordHash, &u.DisplayName)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *Store) GetUserByID(ctx context.Context, id string) (models.User, error) {
	var u models.User
	err := s.Pool.QueryRow(ctx, `
		SELECT id::text, login, display_name, created_at, updated_at
		FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Login, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, ErrNotFound
	}
	return u, err
}

func (s *Store) CreateUser(ctx context.Context, login, passwordHash, displayName string) (models.User, error) {
	var u models.User
	err := s.Pool.QueryRow(ctx, `
		INSERT INTO users (login, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING id::text, login, display_name, created_at, updated_at`,
		login, passwordHash, displayName,
	).Scan(&u.ID, &u.Login, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return u, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (u userRow) Public() models.User {
	return models.User{ID: u.ID, Login: u.Login, DisplayName: u.DisplayName}
}
