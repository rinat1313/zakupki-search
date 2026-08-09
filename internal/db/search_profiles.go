package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rinat1313/zakupki-search/internal/models"
)

const profileSelect = `
	id::text, user_id::text, name, description, source, eis_config, enabled,
	auto_ai, config_version, last_run_at, created_at, updated_at`

func (s *Store) ListSearchProfiles(ctx context.Context, userID string) ([]models.SearchProfile, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT `+profileSelect+`
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
		SELECT `+profileSelect+`
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
	cfg := p.Config
	raw, err := json.Marshal(cfg)
	if err != nil {
		return p, err
	}
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO search_profiles (user_id, name, description, source, eis_config, enabled, auto_ai, config_version)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, 1)
		RETURNING `+profileSelect,
		userID, p.Name, p.Description, p.Source, string(raw), p.Enabled, p.AutoAI,
	)
	return scanProfile(row)
}

func (s *Store) UpdateSearchProfile(ctx context.Context, userID, id string, p models.SearchProfile) (models.SearchProfile, error) {
	if p.Source == "" {
		p.Source = "eis"
	}
	raw, err := json.Marshal(p.Config)
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
		    auto_ai = $8,
		    config_version = CASE
		      WHEN eis_config IS DISTINCT FROM $6::jsonb THEN config_version + 1
		      ELSE config_version
		    END,
		    updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING `+profileSelect,
		id, userID, p.Name, p.Description, p.Source, string(raw), p.Enabled, p.AutoAI,
	)
	out, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	return out, err
}

func (s *Store) SetSearcherAutoAI(ctx context.Context, userID, id string, enabled bool) (models.SearchProfile, error) {
	row := s.Pool.QueryRow(ctx, `
		UPDATE search_profiles
		SET auto_ai = $3, updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING `+profileSelect, id, userID, enabled)
	out, err := scanProfile(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	return out, err
}

func (s *Store) TouchSearcherRun(ctx context.Context, userID, id string, at time.Time) (models.SearchProfile, error) {
	row := s.Pool.QueryRow(ctx, `
		UPDATE search_profiles
		SET last_run_at = $3, updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING `+profileSelect, id, userID, at)
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
	var lastRun *time.Time
	err := row.Scan(
		&p.ID, &p.UserID, &p.Name, &p.Description, &p.Source,
		&raw, &p.Enabled, &p.AutoAI, &p.ConfigVersion, &lastRun, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return p, err
	}
	p.LastRunAt = lastRun
	p.Config = models.DefaultSearcherConfig()
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &p.Config); err != nil {
			return p, fmt.Errorf("decode eis_config: %w", err)
		}
	}
	p.EISConfig = p.Config
	return p, nil
}
