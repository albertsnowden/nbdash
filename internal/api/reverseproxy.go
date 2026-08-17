package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

// --- Clusters ---

type proxyClustersResponse struct {
	Clusters []nbapi.ProxyCluster `json:"clusters"`
	Total    int                  `json:"total"`
}

func (s *Server) listProxyClusters(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	clusters, err := s.api.ListProxyClusters(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, proxyClustersResponse{Clusters: clusters, Total: len(clusters)})
}

func (s *Server) deleteProxyCluster(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteProxyCluster(r.Context(), token, r.PathValue("address")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Proxy tokens ---

type proxyTokensResponse struct {
	Tokens []nbapi.ProxyToken `json:"tokens"`
	Total  int                `json:"total"`
}

func (s *Server) listProxyTokens(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	tokens, err := s.api.ListProxyTokens(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, proxyTokensResponse{Tokens: tokens, Total: len(tokens)})
}

func (s *Server) createProxyToken(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	var req nbapi.ProxyTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Name cannot be empty.")
		return
	}

	created, err := s.api.CreateProxyToken(r.Context(), token, req)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, created)
}

func (s *Server) deleteProxyToken(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteProxyToken(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- Custom domains ---

type reverseProxyDomainsResponse struct {
	Domains []nbapi.ReverseProxyDomain `json:"domains"`
	Total   int                        `json:"total"`
}

func (s *Server) listReverseProxyDomains(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	domains, err := s.api.ListReverseProxyDomains(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, reverseProxyDomainsResponse{Domains: domains, Total: len(domains)})
}

func (s *Server) createReverseProxyDomain(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	var req nbapi.ReverseProxyDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	req.Domain = strings.TrimSpace(req.Domain)
	if req.Domain == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Domain cannot be empty.")
		return
	}

	if err := s.api.CreateReverseProxyDomain(r.Context(), token, req); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (s *Server) deleteReverseProxyDomain(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteReverseProxyDomain(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) validateReverseProxyDomain(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.ValidateReverseProxyDomain(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// --- Services ---

type servicesResponse struct {
	Services []nbapi.Service `json:"services"`
	Total    int             `json:"total"`
}

func (s *Server) listServices(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	services, err := s.api.ListServices(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, servicesResponse{Services: services, Total: len(services)})
}

func (s *Server) getService(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	service, err := s.api.GetService(r.Context(), token, r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, service)
}

// serviceRequest is the create/edit body this dashboard's form actually
// exposes — Auth/AccessRestrictions/Private/AccessGroups are handled
// separately (see updateService) so an edit here can never silently
// clear them.
type serviceRequest struct {
	Name             string                `json:"name"`
	Domain           string                `json:"domain"`
	Mode             string                `json:"mode"`
	ListenPort       int                   `json:"listen_port"`
	Targets          []nbapi.ServiceTarget `json:"targets"`
	Enabled          bool                  `json:"enabled"`
	PassHostHeader   bool                  `json:"pass_host_header"`
	RewriteRedirects bool                  `json:"rewrite_redirects"`
}

func decodeServiceRequest(r *http.Request) (serviceRequest, bool, string) {
	var req serviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, false, "invalid JSON body"
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return req, false, "Name cannot be empty."
	}
	req.Domain = strings.TrimSpace(req.Domain)
	if req.Domain == "" {
		return req, false, "Domain cannot be empty."
	}
	if len(req.Targets) == 0 {
		return req, false, "At least one target is required."
	}
	return req, true, ""
}

func (s *Server) createService(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	req, valid, msg := decodeServiceRequest(r)
	if !valid {
		status := http.StatusUnprocessableEntity
		if msg == "invalid JSON body" {
			status = http.StatusBadRequest
		}
		s.writeError(w, status, msg)
		return
	}

	service, err := s.api.CreateService(r.Context(), token, nbapi.ServiceRequest{
		Name:             req.Name,
		Domain:           req.Domain,
		Mode:             req.Mode,
		ListenPort:       req.ListenPort,
		Targets:          req.Targets,
		Enabled:          req.Enabled,
		PassHostHeader:   req.PassHostHeader,
		RewriteRedirects: req.RewriteRedirects,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, service)
}

// updateService fetches the current service first and carries Auth,
// AccessRestrictions, Private and AccessGroups forward untouched — this
// dashboard's form doesn't expose editing those yet, and PUT is a full
// replace, so skipping the fetch would silently disable whatever auth or
// access restrictions were configured through another client. Same
// discipline as updateGroup/updateZoneMeta elsewhere in this package.
func (s *Server) updateService(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")

	req, valid, msg := decodeServiceRequest(r)
	if !valid {
		status := http.StatusUnprocessableEntity
		if msg == "invalid JSON body" {
			status = http.StatusBadRequest
		}
		s.writeError(w, status, msg)
		return
	}

	current, err := s.api.GetService(r.Context(), token, id)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	service, err := s.api.UpdateService(r.Context(), token, id, nbapi.ServiceRequest{
		Name:               req.Name,
		Domain:             req.Domain,
		Mode:               req.Mode,
		ListenPort:         req.ListenPort,
		Targets:            req.Targets,
		Enabled:            req.Enabled,
		PassHostHeader:     req.PassHostHeader,
		RewriteRedirects:   req.RewriteRedirects,
		Auth:               current.Auth,
		AccessRestrictions: current.AccessRestrictions,
		Private:            current.Private,
		AccessGroups:       current.AccessGroups,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, service)
}

func (s *Server) deleteService(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteService(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
