package models

import "time"

// SearchProfile — именованная поисковая настройка пользователя.
type SearchProfile struct {
	ID          string         `json:"id"`
	UserID      string         `json:"user_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Source      string         `json:"source"` // пока только "eis"
	EISConfig   EISSearchConfig `json:"eis_config"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
