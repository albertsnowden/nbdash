package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

// peersResponse is the list endpoint's shape — total/connected counts are
// account-wide while Peers is the (possibly search-filtered) view, mirroring
// internal/handlers/peers.go's loadPeers: the summary line should keep
// reading "3 of 40 connected" even while a search narrows the table.
type peersResponse struct {
	Peers          []nbapi.Peer `json:"peers"`
	Total          int          `json:"total"`
	ConnectedCount int          `json:"connected_count"`
}

func (s *Server) listPeers(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	peers, err := s.api.ListPeers(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	resp := peersResponse{Total: len(peers)}
	for _, p := range peers {
		if p.Connected {
			resp.ConnectedCount++
		}
	}
	resp.Peers = nbapi.FilterPeers(peers, r.URL.Query().Get("q"))

	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) getPeer(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	peer, err := s.api.GetPeer(r.Context(), token, r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, peer)
}

// updatePeerRequest is the PUT body — a subset of nbapi.PeerRequest, the
// same subset internal/handlers/peers.go's updatePeer form exposes.
// ApprovalRequired, IP and IPv6 stay unset (nil), leaving them untouched on
// every save, for the same reason the htmx form never exposes them either.
type updatePeerRequest struct {
	Name                        string `json:"name"`
	SSHEnabled                  bool   `json:"ssh_enabled"`
	LoginExpirationEnabled      bool   `json:"login_expiration_enabled"`
	InactivityExpirationEnabled bool   `json:"inactivity_expiration_enabled"`
}

func (s *Server) updatePeer(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")

	var req updatePeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Name cannot be empty.")
		return
	}

	peer, err := s.api.UpdatePeer(r.Context(), token, id, nbapi.PeerRequest{
		Name:                        name,
		SSHEnabled:                  req.SSHEnabled,
		LoginExpirationEnabled:      req.LoginExpirationEnabled,
		InactivityExpirationEnabled: req.InactivityExpirationEnabled,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, peer)
}

func (s *Server) deletePeer(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	if err := s.api.DeletePeer(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
