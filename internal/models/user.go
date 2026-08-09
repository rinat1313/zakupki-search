package models

import "time"

type User struct {
	ID          string    `json:"id"`
	Login       string    `json:"login"`
	Name        string    `json:"name"` // UI (gateway) expects "name"
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
