package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

// --- Providers ---

type agentNetworkProvidersResponse struct {
	Providers []nbapi.AgentNetworkProvider `json:"providers"`
	Total     int                          `json:"total"`
}

func (s *Server) listAgentNetworkProviders(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	providers, err := s.api.ListAgentNetworkProviders(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, agentNetworkProvidersResponse{Providers: providers, Total: len(providers)})
}

// providerRequest is the create/edit body this dashboard's form exposes —
// Models/ExtraValues/IdentityHeader* stay untouched on update (see
// updateAgentNetworkProvider) so a save through this UI can't silently
// clear per-model pricing or catalog header values set through another
// client.
type providerRequest struct {
	ProviderID          string `json:"provider_id"`
	Name                string `json:"name"`
	UpstreamURL         string `json:"upstream_url"`
	BootstrapCluster    string `json:"bootstrap_cluster"`
	APIKey              string `json:"api_key"`
	Enabled             bool   `json:"enabled"`
	SkipTLSVerification bool   `json:"skip_tls_verification"`
	MetadataDisabled    bool   `json:"metadata_disabled"`
}

func decodeProviderRequest(r *http.Request) (providerRequest, bool, string) {
	var req providerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, false, "invalid JSON body"
	}
	req.ProviderID = strings.TrimSpace(req.ProviderID)
	req.Name = strings.TrimSpace(req.Name)
	req.UpstreamURL = strings.TrimSpace(req.UpstreamURL)
	if req.ProviderID == "" || req.Name == "" || req.UpstreamURL == "" {
		return req, false, "Provider type, name and upstream URL are all required."
	}
	return req, true, ""
}

func (s *Server) createAgentNetworkProvider(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	req, valid, msg := decodeProviderRequest(r)
	if !valid {
		status := http.StatusUnprocessableEntity
		if msg == "invalid JSON body" {
			status = http.StatusBadRequest
		}
		s.writeError(w, status, msg)
		return
	}
	if req.APIKey == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "API key is required when creating a provider.")
		return
	}

	provider, err := s.api.CreateAgentNetworkProvider(r.Context(), token, nbapi.AgentNetworkProviderRequest{
		ProviderID:          req.ProviderID,
		Name:                req.Name,
		UpstreamURL:         req.UpstreamURL,
		BootstrapCluster:    req.BootstrapCluster,
		APIKey:              req.APIKey,
		Enabled:             req.Enabled,
		SkipTLSVerification: req.SkipTLSVerification,
		MetadataDisabled:    req.MetadataDisabled,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, provider)
}

// updateAgentNetworkProvider fetches the current provider so Models,
// ExtraValues and the identity header fields carry forward unedited — see
// providerRequest's doc comment.
func (s *Server) updateAgentNetworkProvider(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")

	req, valid, msg := decodeProviderRequest(r)
	if !valid {
		status := http.StatusUnprocessableEntity
		if msg == "invalid JSON body" {
			status = http.StatusBadRequest
		}
		s.writeError(w, status, msg)
		return
	}

	current, err := s.api.GetAgentNetworkProvider(r.Context(), token, id)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	provider, err := s.api.UpdateAgentNetworkProvider(r.Context(), token, id, nbapi.AgentNetworkProviderRequest{
		ProviderID:           req.ProviderID,
		Name:                 req.Name,
		UpstreamURL:          req.UpstreamURL,
		APIKey:               req.APIKey, // blank keeps the existing sealed key, same as identity providers' client_secret
		Models:               current.Models,
		ExtraValues:          current.ExtraValues,
		IdentityHeaderUserID: current.IdentityHeaderUserID,
		IdentityHeaderGroups: current.IdentityHeaderGroups,
		Enabled:              req.Enabled,
		SkipTLSVerification:  req.SkipTLSVerification,
		MetadataDisabled:     req.MetadataDisabled,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, provider)
}

func (s *Server) deleteAgentNetworkProvider(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteAgentNetworkProvider(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Policies ---

type agentNetworkPoliciesResponse struct {
	Policies []nbapi.AgentNetworkPolicy `json:"policies"`
	Total    int                        `json:"total"`
}

func (s *Server) listAgentNetworkPolicies(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	policies, err := s.api.ListAgentNetworkPolicies(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, agentNetworkPoliciesResponse{Policies: policies, Total: len(policies)})
}

// agentNetworkPolicyRequest omits GuardrailIDs/Limits for the same reason
// providerRequest omits Models/ExtraValues — no guardrail/rate-limit UI yet.
type agentNetworkPolicyRequest struct {
	Name                   string   `json:"name"`
	Description            string   `json:"description"`
	Enabled                bool     `json:"enabled"`
	SourceGroups           []string `json:"source_groups"`
	DestinationProviderIDs []string `json:"destination_provider_ids"`
}

func decodeAgentNetworkPolicyRequest(r *http.Request) (agentNetworkPolicyRequest, bool, string) {
	var req agentNetworkPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, false, "invalid JSON body"
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return req, false, "Name cannot be empty."
	}
	if len(req.SourceGroups) == 0 {
		return req, false, "At least one source group is required."
	}
	if len(req.DestinationProviderIDs) == 0 {
		return req, false, "At least one destination provider is required."
	}
	return req, true, ""
}

func (s *Server) createAgentNetworkPolicy(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	req, valid, msg := decodeAgentNetworkPolicyRequest(r)
	if !valid {
		status := http.StatusUnprocessableEntity
		if msg == "invalid JSON body" {
			status = http.StatusBadRequest
		}
		s.writeError(w, status, msg)
		return
	}

	policy, err := s.api.CreateAgentNetworkPolicy(r.Context(), token, nbapi.AgentNetworkPolicyRequest{
		Name:                   req.Name,
		Description:            req.Description,
		Enabled:                req.Enabled,
		SourceGroups:           req.SourceGroups,
		DestinationProviderIDs: req.DestinationProviderIDs,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, policy)
}

func (s *Server) updateAgentNetworkPolicy(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")

	req, valid, msg := decodeAgentNetworkPolicyRequest(r)
	if !valid {
		status := http.StatusUnprocessableEntity
		if msg == "invalid JSON body" {
			status = http.StatusBadRequest
		}
		s.writeError(w, status, msg)
		return
	}

	current, err := s.api.GetAgentNetworkPolicy(r.Context(), token, id)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	policy, err := s.api.UpdateAgentNetworkPolicy(r.Context(), token, id, nbapi.AgentNetworkPolicyRequest{
		Name:                   req.Name,
		Description:            req.Description,
		Enabled:                req.Enabled,
		SourceGroups:           req.SourceGroups,
		DestinationProviderIDs: req.DestinationProviderIDs,
		GuardrailIDs:           current.GuardrailIDs,
		Limits:                 current.Limits,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, policy)
}

func (s *Server) deleteAgentNetworkPolicy(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteAgentNetworkPolicy(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
