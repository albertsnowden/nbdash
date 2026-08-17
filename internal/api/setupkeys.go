package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

type setupKeysResponse struct {
	SetupKeys []nbapi.SetupKey `json:"setup_keys"`
	Total     int              `json:"total"`
	Valid     int              `json:"valid"`
}

func (s *Server) listSetupKeys(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	keys, err := s.api.ListSetupKeys(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	resp := setupKeysResponse{SetupKeys: keys, Total: len(keys)}
	for _, k := range keys {
		if k.Valid {
			resp.Valid++
		}
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// createSetupKeyRequest collects ExpiresInDays, not seconds — the API
// itself bounds expires_in to [86400, 31536000] (1-365 days), and asking
// for days is what an operator actually thinks in, same as the htmx form.
type createSetupKeyRequest struct {
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	ExpiresInDays int      `json:"expires_in_days"`
	AutoGroups    []string `json:"auto_groups"`
	UsageLimit    int      `json:"usage_limit"`
	Ephemeral     bool     `json:"ephemeral"`
}

// createSetupKey returns the plaintext key from a successful create — the
// only response from this API that ever contains it (see
// nbapi.SetupKey's doc comment). The frontend must display it once and
// never persist it beyond that render.
func (s *Server) createSetupKey(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	var req createSetupKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Name cannot be empty.")
		return
	}
	if req.Type != "one-off" && req.Type != "reusable" {
		s.writeError(w, http.StatusUnprocessableEntity, "Type must be one-off or reusable.")
		return
	}
	if req.ExpiresInDays < 1 || req.ExpiresInDays > 365 {
		s.writeError(w, http.StatusUnprocessableEntity, "Expiry must be between 1 and 365 days.")
		return
	}
	if req.UsageLimit < 0 {
		s.writeError(w, http.StatusUnprocessableEntity, "Usage limit must be zero or a positive number.")
		return
	}

	autoGroups := req.AutoGroups
	if autoGroups == nil {
		autoGroups = []string{}
	}

	created, err := s.api.CreateSetupKey(r.Context(), token, nbapi.CreateSetupKeyRequest{
		Name:       name,
		Type:       req.Type,
		ExpiresIn:  req.ExpiresInDays * 86400,
		AutoGroups: autoGroups,
		UsageLimit: req.UsageLimit,
		Ephemeral:  req.Ephemeral,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, created)
}

// revokeSetupKey marks a key revoked. The API replaces the whole editable
// set (revoked + auto_groups) on every PUT and rejects a nil auto_groups
// outright, so this fetches the key's current groups first and resends them
// — same full-replace discipline internal/handlers/setupkeys.go's
// revokeSetupKey documents.
func (s *Server) revokeSetupKey(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")

	current, err := s.api.GetSetupKey(r.Context(), token, id)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	autoGroups := current.AutoGroups
	if autoGroups == nil {
		autoGroups = []string{}
	}

	updated, err := s.api.UpdateSetupKey(r.Context(), token, id, nbapi.SetupKeyRequest{
		Revoked:    true,
		AutoGroups: autoGroups,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteSetupKey(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	if err := s.api.DeleteSetupKey(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
