package models

import "time"

// Tender — лёгкая запись о найденной закупке (каталог для search / handoff в parser).
// Не хранит документы и полный HTML — только идентификаторы и сниппет из ленты.
type Tender struct {
	ID             string         `json:"id"`
	UserID         string         `json:"user_id"`
	ProfileID      *string        `json:"profile_id,omitempty"`
	RegNumber      string         `json:"reg_number"`
	Law            string         `json:"law"`
	NoticeURL      string         `json:"notice_url"`
	NoticeGUID     string         `json:"notice_guid"`
	SourceSite     string         `json:"source_site"`
	ObjectTitle    string         `json:"object_title"`
	Status         string         `json:"status"`
	PriceRaw       string         `json:"price_raw"`
	OrgName        string         `json:"org_name"`
	PublishedAt    string         `json:"published_at"`
	UpdatedOnSite  string         `json:"updated_on_site"`
	ApplicationEnd string         `json:"application_end"`
	Payload        map[string]any `json:"payload,omitempty"`
	FoundAt        time.Time      `json:"found_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// TenderFilter — фильтры списка.
type TenderFilter struct {
	ProfileID string
	Law       string
	Q         string // поиск по reg_number / object_title / org_name
	Limit     int
	Offset    int
}
