package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

type groupsResponse struct {
	Groups []nbapi.Group `json:"groups"`
	Total  int           `json:"total"`
}

func (s *Server) listGroups(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	groups, err := s.api.ListGroups(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, groupsResponse{Groups: groups, Total: len(groups)})
}

func (s *Server) getGroup(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	group, err := s.api.GetGroup(r.Context(), token, r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, group)
}

// groupRequest is the create/update body — a subset of nbapi.GroupRequest's
// own shape (Resources is never set here either, same as the htmx form; see
// nbapi.GroupRequest's doc comment for why).
type groupRequest struct {
	Name  string   `json:"name"`
	Peers []string `json:"peers"`
}

func (s *Server) createGroup(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	var req groupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Name cannot be empty.")
		return
	}

	group, err := s.api.CreateGroup(r.Context(), token, nbapi.GroupRequest{Name: name, Peers: req.Peers})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, group)
}

// updateGroup replaces a group's whole peer membership with whatever the
// request carries — the frontend fetches the group fresh before editing
// (usePeer/useGroup), so its checkbox state already represents the complete
// desired membership by the time it submits, the same discipline
// internal/handlers/groups.go's updateGroup documents.
func (s *Server) updateGroup(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")

	var req groupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Name cannot be empty.")
		return
	}

	group, err := s.api.UpdateGroup(r.Context(), token, id, nbapi.GroupRequest{Name: name, Peers: req.Peers})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, group)
}

func (s *Server) deleteGroup(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	if err := s.api.DeleteGroup(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
