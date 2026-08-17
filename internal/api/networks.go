// Networks group one or more routers (routing peers or peer groups) with one
// or more resources (hosts, subnets or domains) reachable through them.
// Unlike Policies' rules, a network's resources and routers are independent
// REST sub-resources with their own IDs and their own endpoints — a create,
// edit or delete of one never needs to fetch the parent network and resend
// anything else. See nbapi/networks.go and nbapi/types.go for the schema
// notes this leans on.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

type networksResponse struct {
	Networks []nbapi.Network `json:"networks"`
	Total    int             `json:"total"`
}

func (s *Server) listNetworks(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	networks, err := s.api.ListNetworks(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, networksResponse{Networks: networks, Total: len(networks)})
}

func (s *Server) getNetwork(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	network, err := s.api.GetNetwork(r.Context(), token, r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, network)
}

type networkRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *Server) createNetwork(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req networkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Name cannot be empty.")
		return
	}

	created, err := s.api.CreateNetwork(r.Context(), token, nbapi.NetworkRequest{
		Name:        name,
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, created)
}

func (s *Server) updateNetworkMeta(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req networkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Name cannot be empty.")
		return
	}

	updated, err := s.api.UpdateNetwork(r.Context(), token, r.PathValue("id"), nbapi.NetworkRequest{
		Name:        name,
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteNetwork(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteNetwork(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type resourcesResponse struct {
	Resources []nbapi.NetworkResource `json:"resources"`
}

func (s *Server) listResources(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	resources, err := s.api.ListNetworkResources(r.Context(), token, r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, resourcesResponse{Resources: resources})
}

// resourceRequest mirrors internal/handlers/networks.go's parseResourceForm
// field for field; the server derives Type from Address itself, so this
// never sets or validates one.
type resourceRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Address     string   `json:"address"`
	Enabled     bool     `json:"enabled"`
	Groups      []string `json:"groups"`
}

func validateResource(req resourceRequest) (nbapi.NetworkResourceRequest, string) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nbapi.NetworkResourceRequest{}, "Name cannot be empty."
	}
	address := strings.TrimSpace(req.Address)
	if address == "" {
		return nbapi.NetworkResourceRequest{}, "Address cannot be empty."
	}
	return nbapi.NetworkResourceRequest{
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Address:     address,
		Enabled:     req.Enabled,
		Groups:      req.Groups,
	}, ""
}

func (s *Server) createResource(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req resourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	nbReq, msg := validateResource(req)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	created, err := s.api.CreateNetworkResource(r.Context(), token, r.PathValue("id"), nbReq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, created)
}

// updateResource replaces one resource's editable fields, including its
// whole group membership — req.Groups must carry every group the resource
// should end up belonging to, not just changes (see
// nbapi.NetworkResourceRequest's doc comment).
func (s *Server) updateResource(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req resourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	nbReq, msg := validateResource(req)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	updated, err := s.api.UpdateNetworkResource(r.Context(), token, r.PathValue("id"), r.PathValue("resourceID"), nbReq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteResource(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteNetworkResource(r.Context(), token, r.PathValue("id"), r.PathValue("resourceID")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type routersResponse struct {
	Routers []nbapi.NetworkRouter `json:"routers"`
}

func (s *Server) listRouters(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	routers, err := s.api.ListNetworkRouters(r.Context(), token, r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, routersResponse{Routers: routers})
}

type routerRequest struct {
	Peer       string   `json:"peer"`
	PeerGroups []string `json:"peer_groups"`
	Metric     int      `json:"metric"`
	Masquerade bool     `json:"masquerade"`
	Enabled    bool     `json:"enabled"`
}

func validateRouter(req routerRequest) (nbapi.NetworkRouterRequest, string) {
	peer := strings.TrimSpace(req.Peer)
	if peer != "" && len(req.PeerGroups) > 0 {
		return nbapi.NetworkRouterRequest{}, "Choose either a single routing peer or peer group(s), not both."
	}
	if peer == "" && len(req.PeerGroups) == 0 {
		return nbapi.NetworkRouterRequest{}, "Choose a routing peer or at least one peer group."
	}
	if req.Metric < 1 || req.Metric > 9999 {
		return nbapi.NetworkRouterRequest{}, "Metric must be a number between 1 and 9999."
	}
	return nbapi.NetworkRouterRequest{
		Peer:       peer,
		PeerGroups: req.PeerGroups,
		Metric:     req.Metric,
		Masquerade: req.Masquerade,
		Enabled:    req.Enabled,
	}, ""
}

// createRouter always sends Enabled true: the real server forces it on
// create regardless of what the request carries (see
// nbapi.NetworkRouterRequest's doc comment), so there's nothing to read
// from the request for it.
func (s *Server) createRouter(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req routerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	nbReq, msg := validateRouter(req)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}
	nbReq.Enabled = true

	created, err := s.api.CreateNetworkRouter(r.Context(), token, r.PathValue("id"), nbReq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, created)
}

// updateRouter honours Enabled from the request — the server's
// force-to-true override only applies on create.
func (s *Server) updateRouter(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req routerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	nbReq, msg := validateRouter(req)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	updated, err := s.api.UpdateNetworkRouter(r.Context(), token, r.PathValue("id"), r.PathValue("routerID"), nbReq)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteRouter(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteNetworkRouter(r.Context(), token, r.PathValue("id"), r.PathValue("routerID")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
