package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rinat1313/zakupki-search/internal/db"
	"github.com/rinat1313/zakupki-search/internal/models"
)

type profileRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Source      string                 `json:"source"`
	EISConfig   *models.SearcherConfig `json:"eis_config"`
	Config      *models.SearcherConfig `json:"config"`
	Enabled     *bool                  `json:"enabled"`
	AutoAI      *bool                  `json:"auto_ai"`
}

func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	list, err := s.Store.ListSearchProfiles(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": list})
}

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	p, err := s.Store.GetSearchProfile(r.Context(), u.ID, r.PathValue("id"))
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get failed")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	var req profileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	cfg := models.DefaultSearcherConfig()
	if req.Config != nil {
		cfg = *req.Config
	} else if req.EISConfig != nil {
		cfg = *req.EISConfig
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	autoAI := false
	if req.AutoAI != nil {
		autoAI = *req.AutoAI
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = "eis"
	}
	if source != "eis" {
		writeError(w, http.StatusBadRequest, "only source=eis is supported for now")
		return
	}

	p, err := s.Store.CreateSearchProfile(r.Context(), u.ID, models.SearchProfile{
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Source:      source,
		Config:      cfg,
		Enabled:     enabled,
		AutoAI:      autoAI,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "profile name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	id := r.PathValue("id")
	existing, err := s.Store.GetSearchProfile(r.Context(), u.ID, id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get failed")
		return
	}

	var req profileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		existing.Name = name
	}
	if req.Description != "" || req.Name != "" {
		existing.Description = strings.TrimSpace(req.Description)
	}
	if req.Config != nil {
		existing.Config = *req.Config
		existing.EISConfig = *req.Config
	} else if req.EISConfig != nil {
		existing.Config = *req.EISConfig
		existing.EISConfig = *req.EISConfig
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.AutoAI != nil {
		existing.AutoAI = *req.AutoAI
	}
	if src := strings.TrimSpace(req.Source); src != "" {
		if src != "eis" {
			writeError(w, http.StatusBadRequest, "only source=eis is supported for now")
			return
		}
		existing.Source = src
	}

	out, err := s.Store.UpdateSearchProfile(r.Context(), u.ID, id, existing)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "profile name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	if err := s.Store.DeleteSearchProfile(r.Context(), u.ID, r.PathValue("id")); errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleProfileEISURL(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	p, err := s.Store.GetSearchProfile(r.Context(), u.ID, r.PathValue("id"))
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"profile_id":     p.ID,
		"config_version": p.ConfigVersion,
		"name":           p.Name,
		"url":            p.Config.ResultsURL(s.Cfg.EISBaseURL),
		"query":          p.Config.QueryValues(),
		"auto_ai":        p.AutoAI,
	})
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint")
}
