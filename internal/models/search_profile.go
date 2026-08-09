package models

import "time"

// SearchProfile — именованная поисковая настройка пользователя.
//
// ID (UUID) — стабильный идентификатор конфигурации поиска.
// Его же zakupki-core сохраняет у тендера как search_profile_id,
// чтобы связать найденную закупку с исходным поисковым запросом.
//
// ConfigVersion увеличивается при изменении eis_config. Worker/core
// по новой версии пересобирают список тендеров профиля (upsert + delete stale).
type SearchProfile struct {
	ID            string          `json:"id"`
	UserID        string          `json:"user_id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Source        string          `json:"source"` // пока только "eis"
	EISConfig     EISSearchConfig `json:"eis_config"`
	Enabled       bool            `json:"enabled"`
	ConfigVersion int64           `json:"config_version"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
