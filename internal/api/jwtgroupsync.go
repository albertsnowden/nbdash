package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

// jwtGroupSync is the three AccountSettings fields (out of 20+) this
// dashboard's JWT Group Sync page manages — mirrors the wire names of
// AccountSettings.jwt_groups_enabled/jwt_groups_claim_name/jwt_allow_groups
// in openapi.yml.
type jwtGroupSync struct {
	Enabled       bool     `json:"jwt_groups_enabled"`
	ClaimName     string   `json:"jwt_groups_claim_name"`
	AllowedGroups []string `json:"jwt_allow_groups"`
}

func (s *Server) getJWTGroupSync(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	account, err := s.api.CurrentAccount(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	var settings jwtGroupSync
	if err := json.Unmarshal(account.Settings, &settings); err != nil {
		s.writeError(w, http.StatusBadGateway, "could not parse account settings")
		return
	}
	s.writeJSON(w, http.StatusOK, settings)
}

// updateJWTGroupSync fetches the account's current settings, patches only
// the three JWT fields into that JSON object, and PUTs the whole thing
// back — AccountSettings has 20+ other fields (peer expiration, DNS
// domain, dashboard feature flags, ...) that must round-trip untouched,
// same discipline as this package's other full-replace-on-PUT handlers.
func (s *Server) updateJWTGroupSync(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	var req jwtGroupSync
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.ClaimName = strings.TrimSpace(req.ClaimName)
	if req.Enabled && req.ClaimName == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Claim name is required when JWT group sync is enabled.")
		return
	}

	account, err := s.api.CurrentAccount(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	var settingsMap map[string]json.RawMessage
	if err := json.Unmarshal(account.Settings, &settingsMap); err != nil {
		s.writeError(w, http.StatusBadGateway, "could not parse account settings")
		return
	}
	if settingsMap == nil {
		settingsMap = map[string]json.RawMessage{}
	}

	enabledJSON, _ := json.Marshal(req.Enabled)
	claimJSON, _ := json.Marshal(req.ClaimName)
	groupsJSON, _ := json.Marshal(req.AllowedGroups)
	settingsMap["jwt_groups_enabled"] = enabledJSON
	settingsMap["jwt_groups_claim_name"] = claimJSON
	settingsMap["jwt_allow_groups"] = groupsJSON

	patched, err := json.Marshal(settingsMap)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, "could not encode account settings")
		return
	}

	updated, err := s.api.UpdateAccountSettings(r.Context(), token, account.ID, patched)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	var settings jwtGroupSync
	if err := json.Unmarshal(updated.Settings, &settings); err != nil {
		s.writeError(w, http.StatusBadGateway, "could not parse account settings")
		return
	}
	s.writeJSON(w, http.StatusOK, settings)
}
