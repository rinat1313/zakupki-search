package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/rinat1313/zakupki-search/internal/models"
)

const tenderSelect = `
	id::text, user_id::text,
	CASE WHEN profile_id IS NULL THEN NULL ELSE profile_id::text END,
	reg_number, law, notice_url, notice_guid, source_site,
	object_title, status, price_raw, org_name,
	published_at, updated_on_site, application_end,
	payload, found_at, created_at, updated_at`

func (s *Store) ListTenders(ctx context.Context, userID string, f models.TenderFilter) ([]models.Tender, int, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}

	where := []string{"user_id = $1"}
	args := []any{userID}
	n := 2
	if f.ProfileID != "" {
		where = append(where, fmt.Sprintf("profile_id = $%d", n))
		args = append(args, f.ProfileID)
		n++
	}
	if f.Law != "" {
		where = append(where, fmt.Sprintf("law = $%d", n))
		args = append(args, f.Law)
		n++
	}
	if q := strings.TrimSpace(f.Q); q != "" {
		where = append(where, fmt.Sprintf(
			"(reg_number ILIKE $%d OR object_title ILIKE $%d OR org_name ILIKE $%d)", n, n, n,
		))
		args = append(args, "%"+q+"%")
		n++
	}
	w := strings.Join(where, " AND ")

	var total int
	if err := s.Pool.QueryRow(ctx, `SELECT count(*) FROM tenders WHERE `+w, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, f.Limit, f.Offset)
	rows, err := s.Pool.Query(ctx, `
		SELECT `+tenderSelect+`
		FROM tenders
		WHERE `+w+`
		ORDER BY found_at DESC, created_at DESC
		LIMIT $`+fmt.Sprint(n)+` OFFSET $`+fmt.Sprint(n+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]models.Tender, 0)
	for rows.Next() {
		t, err := scanTender(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, t)
	}
	return out, total, rows.Err()
}

func (s *Store) GetTender(ctx context.Context, userID, id string) (models.Tender, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT `+tenderSelect+`
		FROM tenders WHERE id = $1 AND user_id = $2`, id, userID)
	t, err := scanTender(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

func (s *Store) GetTenderByReg(ctx context.Context, userID, sourceSite, regNumber string) (models.Tender, error) {
	row := s.Pool.QueryRow(ctx, `
		SELECT `+tenderSelect+`
		FROM tenders
		WHERE user_id = $1 AND source_site = $2 AND reg_number = $3`,
		userID, sourceSite, regNumber)
	t, err := scanTender(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return t, ErrNotFound
	}
	return t, err
}

func (s *Store) CreateTender(ctx context.Context, userID string, t models.Tender) (models.Tender, error) {
	normalizeTender(&t)
	raw, err := marshalPayload(t.Payload)
	if err != nil {
		return t, err
	}
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO tenders (
			user_id, profile_id, reg_number, law, notice_url, notice_guid, source_site,
			object_title, status, price_raw, org_name,
			published_at, updated_on_site, application_end, payload
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15::jsonb
		)
		RETURNING `+tenderSelect,
		userID, t.ProfileID, t.RegNumber, t.Law, t.NoticeURL, t.NoticeGUID, t.SourceSite,
		t.ObjectTitle, t.Status, t.PriceRaw, t.OrgName,
		t.PublishedAt, t.UpdatedOnSite, t.ApplicationEnd, raw,
	)
	return scanTender(row)
}

func (s *Store) UpdateTender(ctx context.Context, userID, id string, t models.Tender) (models.Tender, error) {
	normalizeTender(&t)
	raw, err := marshalPayload(t.Payload)
	if err != nil {
		return t, err
	}
	row := s.Pool.QueryRow(ctx, `
		UPDATE tenders SET
			profile_id = $3,
			reg_number = $4,
			law = $5,
			notice_url = $6,
			notice_guid = $7,
			source_site = $8,
			object_title = $9,
			status = $10,
			price_raw = $11,
			org_name = $12,
			published_at = $13,
			updated_on_site = $14,
			application_end = $15,
			payload = $16::jsonb,
			updated_at = now()
		WHERE id = $1 AND user_id = $2
		RETURNING `+tenderSelect,
		id, userID, t.ProfileID, t.RegNumber, t.Law, t.NoticeURL, t.NoticeGUID, t.SourceSite,
		t.ObjectTitle, t.Status, t.PriceRaw, t.OrgName,
		t.PublishedAt, t.UpdatedOnSite, t.ApplicationEnd, raw,
	)
	out, err := scanTender(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	return out, err
}

// UpsertTender inserts or updates by (user_id, source_site, reg_number).
func (s *Store) UpsertTender(ctx context.Context, userID string, t models.Tender) (models.Tender, error) {
	normalizeTender(&t)
	raw, err := marshalPayload(t.Payload)
	if err != nil {
		return t, err
	}
	row := s.Pool.QueryRow(ctx, `
		INSERT INTO tenders (
			user_id, profile_id, reg_number, law, notice_url, notice_guid, source_site,
			object_title, status, price_raw, org_name,
			published_at, updated_on_site, application_end, payload
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14, $15::jsonb
		)
		ON CONFLICT (user_id, source_site, reg_number) DO UPDATE SET
			profile_id = COALESCE(EXCLUDED.profile_id, tenders.profile_id),
			law = CASE WHEN EXCLUDED.law <> '' THEN EXCLUDED.law ELSE tenders.law END,
			notice_url = CASE WHEN EXCLUDED.notice_url <> '' THEN EXCLUDED.notice_url ELSE tenders.notice_url END,
			notice_guid = CASE WHEN EXCLUDED.notice_guid <> '' THEN EXCLUDED.notice_guid ELSE tenders.notice_guid END,
			object_title = CASE WHEN EXCLUDED.object_title <> '' THEN EXCLUDED.object_title ELSE tenders.object_title END,
			status = CASE WHEN EXCLUDED.status <> '' THEN EXCLUDED.status ELSE tenders.status END,
			price_raw = CASE WHEN EXCLUDED.price_raw <> '' THEN EXCLUDED.price_raw ELSE tenders.price_raw END,
			org_name = CASE WHEN EXCLUDED.org_name <> '' THEN EXCLUDED.org_name ELSE tenders.org_name END,
			published_at = CASE WHEN EXCLUDED.published_at <> '' THEN EXCLUDED.published_at ELSE tenders.published_at END,
			updated_on_site = CASE WHEN EXCLUDED.updated_on_site <> '' THEN EXCLUDED.updated_on_site ELSE tenders.updated_on_site END,
			application_end = CASE WHEN EXCLUDED.application_end <> '' THEN EXCLUDED.application_end ELSE tenders.application_end END,
			payload = tenders.payload || EXCLUDED.payload,
			found_at = now(),
			updated_at = now()
		RETURNING `+tenderSelect,
		userID, t.ProfileID, t.RegNumber, t.Law, t.NoticeURL, t.NoticeGUID, t.SourceSite,
		t.ObjectTitle, t.Status, t.PriceRaw, t.OrgName,
		t.PublishedAt, t.UpdatedOnSite, t.ApplicationEnd, raw,
	)
	return scanTender(row)
}

func (s *Store) DeleteTender(ctx context.Context, userID, id string) error {
	tag, err := s.Pool.Exec(ctx, `DELETE FROM tenders WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteTendersByProfile(ctx context.Context, userID, profileID string) (int64, error) {
	tag, err := s.Pool.Exec(ctx, `
		DELETE FROM tenders WHERE user_id = $1 AND profile_id = $2`, userID, profileID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (s *Store) ProfileOwnedByUser(ctx context.Context, userID, profileID string) (bool, error) {
	var ok bool
	err := s.Pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM search_profiles WHERE id = $1 AND user_id = $2)`,
		profileID, userID,
	).Scan(&ok)
	return ok, err
}

func normalizeTender(t *models.Tender) {
	t.RegNumber = strings.TrimSpace(strings.TrimPrefix(t.RegNumber, "№"))
	t.RegNumber = strings.TrimSpace(t.RegNumber)
	t.Law = strings.TrimSpace(t.Law)
	t.NoticeURL = strings.TrimSpace(t.NoticeURL)
	t.NoticeGUID = strings.TrimSpace(t.NoticeGUID)
	t.SourceSite = strings.TrimSpace(t.SourceSite)
	if t.SourceSite == "" {
		t.SourceSite = "https://zakupki.gov.ru"
	}
	t.ObjectTitle = strings.TrimSpace(t.ObjectTitle)
	t.Status = strings.TrimSpace(t.Status)
	t.PriceRaw = strings.TrimSpace(t.PriceRaw)
	t.OrgName = strings.TrimSpace(t.OrgName)
	t.PublishedAt = strings.TrimSpace(t.PublishedAt)
	t.UpdatedOnSite = strings.TrimSpace(t.UpdatedOnSite)
	t.ApplicationEnd = strings.TrimSpace(t.ApplicationEnd)
	if t.Payload == nil {
		t.Payload = map[string]any{}
	}
	if t.ProfileID != nil && strings.TrimSpace(*t.ProfileID) == "" {
		t.ProfileID = nil
	}
}

func marshalPayload(p map[string]any) (string, error) {
	if p == nil {
		p = map[string]any{}
	}
	b, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func scanTender(row scannable) (models.Tender, error) {
	var t models.Tender
	var raw []byte
	err := row.Scan(
		&t.ID, &t.UserID, &t.ProfileID,
		&t.RegNumber, &t.Law, &t.NoticeURL, &t.NoticeGUID, &t.SourceSite,
		&t.ObjectTitle, &t.Status, &t.PriceRaw, &t.OrgName,
		&t.PublishedAt, &t.UpdatedOnSite, &t.ApplicationEnd,
		&raw, &t.FoundAt, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return t, err
	}
	t.Payload = map[string]any{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &t.Payload); err != nil {
			return t, fmt.Errorf("decode payload: %w", err)
		}
	}
	return t, nil
}
