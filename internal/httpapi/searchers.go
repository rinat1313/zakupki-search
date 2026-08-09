package httpapi

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rinat1313/zakupki-search/internal/coreclient"
	"github.com/rinat1313/zakupki-search/internal/db"
	"github.com/rinat1313/zakupki-search/internal/models"
)

type searcherWriteRequest struct {
	Name   string                 `json:"name"`
	Config *models.SearcherConfig `json:"config"`
	AutoAI *bool                  `json:"auto_ai"`
}

type autoAIRequest struct {
	Enabled *bool `json:"enabled"`
	AutoAI  *bool `json:"auto_ai"`
}

func (s *Server) handleListSearchers(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	list, err := s.Store.ListSearchProfiles(r.Context(), u.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	out := make([]models.Searcher, 0, len(list))
	for _, p := range list {
		sr := p.AsSearcher()
		if s.Core != nil && s.Core.Enabled() {
			sr.TendersCount = s.Core.TendersCount(r.Context(), p.ID)
		}
		out = append(out, sr)
	}
	// UI accepts either a bare array or {items:[]}.
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGetSearcher(w http.ResponseWriter, r *http.Request) {
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
	sr := p.AsSearcher()
	if s.Core != nil && s.Core.Enabled() {
		sr.TendersCount = s.Core.TendersCount(r.Context(), p.ID)
	}
	writeJSON(w, http.StatusOK, sr)
}

func (s *Server) handleCreateSearcher(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	var req searcherWriteRequest
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
	}
	autoAI := false
	if req.AutoAI != nil {
		autoAI = *req.AutoAI
	}
	p, err := s.Store.CreateSearchProfile(r.Context(), u.ID, models.SearchProfile{
		Name:    name,
		Source:  "eis",
		Config:  cfg,
		Enabled: true,
		AutoAI:  autoAI,
	})
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "profile name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	if s.Core != nil && s.Core.Enabled() {
		_ = s.Core.EnsureCategory(r.Context(), p.ID, p.Name)
	}
	writeJSON(w, http.StatusCreated, p.AsSearcher())
}

func (s *Server) handleUpdateSearcher(w http.ResponseWriter, r *http.Request) {
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
	var req searcherWriteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if name := strings.TrimSpace(req.Name); name != "" {
		existing.Name = name
	}
	if req.Config != nil {
		existing.Config = *req.Config
		existing.EISConfig = *req.Config
	}
	if req.AutoAI != nil {
		existing.AutoAI = *req.AutoAI
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
	if s.Core != nil && s.Core.Enabled() {
		_ = s.Core.EnsureCategory(r.Context(), out.ID, out.Name)
	}
	sr := out.AsSearcher()
	if s.Core != nil && s.Core.Enabled() {
		sr.TendersCount = s.Core.TendersCount(r.Context(), out.ID)
	}
	writeJSON(w, http.StatusOK, sr)
}

func (s *Server) handleDeleteSearcher(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) handleSetSearcherAutoAI(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	var req autoAIRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	var enabled *bool
	if req.Enabled != nil {
		enabled = req.Enabled
	} else if req.AutoAI != nil {
		enabled = req.AutoAI
	}
	if enabled == nil {
		writeError(w, http.StatusBadRequest, "нужно поле enabled или auto_ai")
		return
	}
	out, err := s.Store.SetSearcherAutoAI(r.Context(), u.ID, r.PathValue("id"), *enabled)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	sr := out.AsSearcher()
	if s.Core != nil && s.Core.Enabled() {
		sr.TendersCount = s.Core.TendersCount(r.Context(), out.ID)
	}
	writeJSON(w, http.StatusOK, sr)
}

func (s *Server) handleListSearcherTenders(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	id := r.PathValue("id")
	if _, err := s.Store.GetSearchProfile(r.Context(), u.ID, id); errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "get failed")
		return
	}
	if s.Core == nil || !s.Core.Enabled() {
		writeJSON(w, http.StatusOK, map[string]any{"items": []any{}, "total": 0})
		return
	}
	raw, err := s.Core.ListTendersBySearchConfig(r.Context(), id, r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, http.StatusBadGateway, "core tenders: "+err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

func (s *Server) handleRunSearcher(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	id := r.PathValue("id")
	p, err := s.Store.GetSearchProfile(r.Context(), u.ID, id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get failed")
		return
	}

	runID := uuid.NewString()
	now := time.Now().UTC()
	if _, err := s.Store.TouchSearcherRun(r.Context(), u.ID, id, now); err != nil {
		writeError(w, http.StatusInternalServerError, "touch run failed")
		return
	}
	if s.Core != nil && s.Core.Enabled() {
		_ = s.Core.EnsureCategory(r.Context(), p.ID, p.Name)
	}

	// Synchronous first-pass fetch so UI gets found_count; remaining pages can expand later.
	found := 0
	msg := "Поиск выполнен. Новые тендеры сохраняются в СУБД и обрабатываются."
	status := "done"
	if s.EIS != nil {
		hits, ferr := s.EIS.FetchFirstPages(r.Context(), p.Config, 3)
		if ferr != nil {
			log.Printf("eis fetch searcher=%s: %v", id, ferr)
			status = "error"
			msg = "Ошибка запроса к ЕИС: " + ferr.Error()
		} else {
			found = len(hits)
			if s.Core != nil && s.Core.Enabled() && len(hits) > 0 {
				items := make([]coreclient.SyncItem, 0, len(hits))
				for _, h := range hits {
					items = append(items, coreclient.SyncItem{
						RegNumber:  h.RegNumber,
						SourceSite: "https://zakupki.gov.ru",
						NoticeURL:  h.NoticeURL,
						Law:        h.Law,
						ObjectName: h.ObjectName,
					})
				}
				if err := s.Core.SyncSearchConfig(r.Context(), p.ID, items, true); err != nil {
					log.Printf("core sync searcher=%s: %v", id, err)
					status = "error"
					msg = "ЕИС ок, но sync в core: " + err.Error()
				}
			} else if s.Core == nil || !s.Core.Enabled() {
				msg = "Найдено в ЕИС, но CORE_URL не задан — список в core не обновлён."
				if found == 0 {
					msg = "В ЕИС ничего не найдено (или пустая выдача). CORE_URL не задан."
				}
			} else if found == 0 {
				msg = "В ЕИС по текущим фильтрам ничего не найдено."
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"run_id":      runID,
		"searcher_id": id,
		"status":      status,
		"found_count": found,
		"message":     msg,
		"auto_ai":     p.AutoAI,
		"config_version": p.ConfigVersion,
	})
}
