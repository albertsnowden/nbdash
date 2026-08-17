// Identity Providers is a Settings sub-page: SSO connectors configured on
// the account, backed by the Management API's /identity-providers CRUD
// endpoints. Mirrors internal/handlers/identityproviders.go's validation
// exactly — see nbapi.IdentityProviderRequest's doc comment for the
// server-side quirks (blank secret preserves the stored one on update,
// issuer is ignored entirely for google/microsoft) this leans on.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

type identityProvidersResponse struct {
	Providers []nbapi.IdentityProvider `json:"providers"`
	Total     int                      `json:"total"`
}

func (s *Server) listIdentityProviders(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	providers, err := s.api.ListIdentityProviders(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, identityProvidersResponse{Providers: providers, Total: len(providers)})
}

func (s *Server) getIdentityProvider(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	provider, err := s.api.GetIdentityProvider(r.Context(), token, r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, provider)
}

type identityProviderRequest struct {
	Type         nbapi.IdentityProviderType `json:"type"`
	Name         string                     `json:"name"`
	Issuer       string                     `json:"issuer"`
	ClientID     string                     `json:"client_id"`
	ClientSecret string                     `json:"client_secret"`
}

// validateIdentityProvider mirrors internal/handlers/identityproviders.go's
// parseIdentityProviderForm. requireSecret is true only for create — on
// update a blank secret means "keep the one already stored," not "clear
// it," so it's optional there.
func validateIdentityProvider(req identityProviderRequest, requireSecret bool) (nbapi.IdentityProviderRequest, string) {
	valid := false
	for _, t := range nbapi.IdentityProviderTypes {
		if t == req.Type {
			valid = true
			break
		}
	}
	if !valid {
		return nbapi.IdentityProviderRequest{}, "Choose a valid provider type."
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nbapi.IdentityProviderRequest{}, "Name cannot be empty."
	}

	clientID := strings.TrimSpace(req.ClientID)
	if clientID == "" {
		return nbapi.IdentityProviderRequest{}, "Client ID cannot be empty."
	}

	issuer := strings.TrimSpace(req.Issuer)
	if issuer == "" && !req.Type.HasBuiltInIssuer() {
		return nbapi.IdentityProviderRequest{}, "Issuer URL cannot be empty for this provider type."
	}

	if req.ClientSecret == "" && requireSecret {
		return nbapi.IdentityProviderRequest{}, "Client secret cannot be empty."
	}

	return nbapi.IdentityProviderRequest{
		Type:         req.Type,
		Name:         name,
		Issuer:       issuer,
		ClientID:     clientID,
		ClientSecret: req.ClientSecret,
	}, ""
}

func (s *Server) createIdentityProvider(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req identityProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	nbReq, msg := validateIdentityProvider(req, true)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	created, err := s.api.CreateIdentityProvider(r.Context(), token, nbReq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, created)
}

func (s *Server) updateIdentityProvider(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req identityProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	nbReq, msg := validateIdentityProvider(req, false)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	updated, err := s.api.UpdateIdentityProvider(r.Context(), token, r.PathValue("id"), nbReq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteIdentityProvider(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteIdentityProvider(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
