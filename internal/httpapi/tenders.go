package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/rinat1313/zakupki-search/internal/db"
	"github.com/rinat1313/zakupki-search/internal/models"
)

type tenderWriteRequest struct {
	ProfileID      *string        `json:"profile_id"`
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
	Payload        map[string]any `json:"payload"`
}

type tenderBatchRequest struct {
	Items []tenderWriteRequest `json:"items"`
}

func (s *Server) handleListTenders(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	list, total, err := s.Store.ListTenders(r.Context(), u.ID, models.TenderFilter{
		ProfileID: strings.TrimSpace(q.Get("profile_id")),
		Law:       strings.TrimSpace(q.Get("law")),
		Q:         strings.TrimSpace(q.Get("q")),
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "list failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":  list,
		"total":  total,
		"limit":  clampLimit(limit),
		"offset": max0(offset),
	})
}

func (s *Server) handleGetTender(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	t, err := s.Store.GetTender(r.Context(), u.ID, r.PathValue("id"))
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get failed")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) handleCreateTender(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	var req tenderWriteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	t, ok := s.tenderFromWrite(w, r, u.ID, req)
	if !ok {
		return
	}
	out, err := s.Store.CreateTender(r.Context(), u.ID, t)
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "tender already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "create failed")
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) handleUpdateTender(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	id := r.PathValue("id")
	existing, err := s.Store.GetTender(r.Context(), u.ID, id)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "get failed")
		return
	}

	var req tenderWriteRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	merged := mergeTenderWrite(existing, req)
	if !s.validateProfile(w, r, u.ID, merged.ProfileID) {
		return
	}
	if merged.RegNumber == "" {
		writeError(w, http.StatusBadRequest, "reg_number required")
		return
	}

	out, err := s.Store.UpdateTender(r.Context(), u.ID, id, merged)
	if errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if err != nil {
		if isUniqueViolation(err) {
			writeError(w, http.StatusConflict, "tender already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleDeleteTender(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	if err := s.Store.DeleteTender(r.Context(), u.ID, r.PathValue("id")); errors.Is(err, db.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleBatchUpsertTenders(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	var req tenderBatchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if len(req.Items) == 0 {
		writeError(w, http.StatusBadRequest, "items required")
		return
	}
	if len(req.Items) > 500 {
		writeError(w, http.StatusBadRequest, "too many items (max 500)")
		return
	}

	created := make([]models.Tender, 0, len(req.Items))
	for i, item := range req.Items {
		t, ok := s.tenderFromWrite(w, r, u.ID, item)
		if !ok {
			return
		}
		out, err := s.Store.UpsertTender(r.Context(), u.ID, t)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "upsert failed at item "+strconv.Itoa(i))
			return
		}
		created = append(created, out)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items": created,
		"count": len(created),
	})
}

func (s *Server) handleDeleteTendersByProfile(w http.ResponseWriter, r *http.Request) {
	u := userFrom(r)
	profileID := strings.TrimSpace(r.URL.Query().Get("profile_id"))
	if profileID == "" {
		writeError(w, http.StatusBadRequest, "profile_id required")
		return
	}
	if !s.validateProfile(w, r, u.ID, &profileID) {
		return
	}
	n, err := s.Store.DeleteTendersByProfile(r.Context(), u.ID, profileID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "deleted": n})
}

func (s *Server) tenderFromWrite(w http.ResponseWriter, r *http.Request, userID string, req tenderWriteRequest) (models.Tender, bool) {
	reg := strings.TrimSpace(strings.TrimPrefix(req.RegNumber, "№"))
	if reg == "" {
		writeError(w, http.StatusBadRequest, "reg_number required")
		return models.Tender{}, false
	}
	if !s.validateProfile(w, r, userID, req.ProfileID) {
		return models.Tender{}, false
	}
	payload := req.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	return models.Tender{
		ProfileID:      req.ProfileID,
		RegNumber:      reg,
		Law:            req.Law,
		NoticeURL:      req.NoticeURL,
		NoticeGUID:     req.NoticeGUID,
		SourceSite:     req.SourceSite,
		ObjectTitle:    req.ObjectTitle,
		Status:         req.Status,
		PriceRaw:       req.PriceRaw,
		OrgName:        req.OrgName,
		PublishedAt:    req.PublishedAt,
		UpdatedOnSite:  req.UpdatedOnSite,
		ApplicationEnd: req.ApplicationEnd,
		Payload:        payload,
	}, true
}

func (s *Server) validateProfile(w http.ResponseWriter, r *http.Request, userID string, profileID *string) bool {
	if profileID == nil || strings.TrimSpace(*profileID) == "" {
		return true
	}
	ok, err := s.Store.ProfileOwnedByUser(r.Context(), userID, strings.TrimSpace(*profileID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profile check failed")
		return false
	}
	if !ok {
		writeError(w, http.StatusBadRequest, "profile_id not found")
		return false
	}
	return true
}

func mergeTenderWrite(existing models.Tender, req tenderWriteRequest) models.Tender {
	out := existing
	if req.ProfileID != nil {
		out.ProfileID = req.ProfileID
	}
	if v := strings.TrimSpace(req.RegNumber); v != "" {
		out.RegNumber = v
	}
	if req.Law != "" {
		out.Law = req.Law
	}
	if req.NoticeURL != "" {
		out.NoticeURL = req.NoticeURL
	}
	if req.NoticeGUID != "" {
		out.NoticeGUID = req.NoticeGUID
	}
	if req.SourceSite != "" {
		out.SourceSite = req.SourceSite
	}
	if req.ObjectTitle != "" {
		out.ObjectTitle = req.ObjectTitle
	}
	if req.Status != "" {
		out.Status = req.Status
	}
	if req.PriceRaw != "" {
		out.PriceRaw = req.PriceRaw
	}
	if req.OrgName != "" {
		out.OrgName = req.OrgName
	}
	if req.PublishedAt != "" {
		out.PublishedAt = req.PublishedAt
	}
	if req.UpdatedOnSite != "" {
		out.UpdatedOnSite = req.UpdatedOnSite
	}
	if req.ApplicationEnd != "" {
		out.ApplicationEnd = req.ApplicationEnd
	}
	if req.Payload != nil {
		out.Payload = req.Payload
	}
	return out
}

func clampLimit(limit int) int {
	if limit <= 0 || limit > 200 {
		return 50
	}
	return limit
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
