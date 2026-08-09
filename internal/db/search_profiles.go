package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/rinat1313/zakupki-search/internal/models"
)

func (s *Store) ListSearchProfiles(ctx context.Context, userID string) ([]models.SearchProfile, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id::text, user_id::text, name, description, source, eis_config, enabled, created_at, updated_at
		FROM search_profiles
		WHERE user_id = $1
		ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]models.SearchProfile, 0)
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) GetSearchProfile(ctx context.Context, userID, id string) (models.SearchProfile, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT id::text, user_id::text, name, description, source, eis_config, enabled, created_at, updated_at
		FROM search_profiles
		WHERE id = $1 AND user_id = $2`, id, userID)
	p, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, ErrNotFound
	}
	return p, err
}

func (s *Store) CreateSearchProfile(ctx context.Context, userID string, p models.SearchProfile) (models.SearchProfile, error) {
	if p.Source == "" {
		p.Source = "eis"
	}
	raw, err := json.Marshal(p.EISConfig)
	if err != nil {
		return p, err
	}
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO search_profiles (user_id, name, description, source, eis_config, enabled)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6)
		RETURNING id::text, user_id::text, name, description, source, eis_config, enabled, created_at, updated_at`,
		userID, p.Name, p.Description, p.Source, string(raw), p.Enabled,
	)
	return scanProfile(row)
}

func (s *Store) UpdateSearchProfile(ctx context.Context, userID, id string, p models.SearchProfile) (models.SearchProfile, error) {
	if p.Source == "" {
		p.Source = "eis"
	}
	raw, err := json.Marshal(p.EISConfig)
	if err != nil {
		return p, err
	}
	row := s.Pool.QueryRow(ctx, `
		UPDATE search_profiles
		SET name = $3,
		    description = $4,
		    source = $5,
		    eis_config = $6::jsonb,
		    enabled = $7,
		    updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING id::text, user_id::text, name, description, source, eis_config, enabled, created_at, updated_at`,
		id, userID, p.Name, p.Description, p.Source, string(raw), p.Enabled,
	)
	out, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	return out, err
}

func (s *Store) DeleteSearchProfile(ctx context.Context, userID, id string) error {
	tag, err := s.Pool.Exec(ctx, `
		DELETE FROM search_profiles WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanProfile(row scannable) (models.SearchProfile, error) {
	var p models.SearchProfile
	var raw []byte
	err := row.Scan(
		&p.ID, &p.UserID, &p.Name, &p.Description, &p.Source,
		&raw, &p.Enabled, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return p, err
	}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p.EISConfig); err != nil {
			return p, fmt.Errorf("decode eis_config: %w", err)
		}
	}
	return p, nil
}
