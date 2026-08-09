package models

import "time"

// SearchProfile — поисковая настройка (внутреннее хранилище / legacy /search-profiles).
//
// ID (UUID) — стабильный идентификатор. В UI/gateway это Searcher.id;
// в zakupki-core — search_config_id / search_profile_id.
type SearchProfile struct {
	ID            string         `json:"id"`
	UserID        string         `json:"user_id"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	Source        string         `json:"source"`
	Config        SearcherConfig `json:"config"`
	EISConfig     SearcherConfig `json:"eis_config"` // alias for older clients
	Enabled       bool           `json:"enabled"`
	AutoAI        bool           `json:"auto_ai"`
	ConfigVersion int64          `json:"config_version"`
	TendersCount  int            `json:"tenders_count,omitempty"`
	LastRunAt     *time.Time     `json:"last_run_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// Searcher — DTO ответа для gateway UI (/api/v1/searchers).
type Searcher struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Config       SearcherConfig `json:"config"`
	AutoAI       bool           `json:"auto_ai"`
	TendersCount int            `json:"tenders_count"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	LastRunAt    *time.Time     `json:"last_run_at"`
}

func (p SearchProfile) AsSearcher() Searcher {
	cfg := p.Config
	if cfg.RecordsPerPage == 0 {
		cfg = p.EISConfig
	}
	return Searcher{
		ID:           p.ID,
		Name:         p.Name,
		Config:       cfg,
		AutoAI:       p.AutoAI,
		TendersCount: p.TendersCount,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
		LastRunAt:    p.LastRunAt,
	}
}
