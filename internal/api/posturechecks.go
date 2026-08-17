package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

type postureChecksResponse struct {
	PostureChecks []nbapi.PostureCheck `json:"posture_checks"`
	Total         int                  `json:"total"`
}

func (s *Server) listPostureChecks(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	checks, err := s.api.ListPostureChecks(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, postureChecksResponse{PostureChecks: checks, Total: len(checks)})
}

func (s *Server) getPostureCheck(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	check, err := s.api.GetPostureCheck(r.Context(), token, r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, check)
}

func decodePostureCheckRequest(r *http.Request) (nbapi.PostureCheckRequest, bool, string) {
	var req nbapi.PostureCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return req, false, "invalid JSON body"
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return req, false, "Name cannot be empty."
	}
	return req, true, ""
}

func (s *Server) createPostureCheck(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	req, valid, msg := decodePostureCheckRequest(r)
	if !valid {
		status := http.StatusUnprocessableEntity
		if msg == "invalid JSON body" {
			status = http.StatusBadRequest
		}
		s.writeError(w, status, msg)
		return
	}

	check, err := s.api.CreatePostureCheck(r.Context(), token, req)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, check)
}

func (s *Server) updatePostureCheck(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")

	req, valid, msg := decodePostureCheckRequest(r)
	if !valid {
		status := http.StatusUnprocessableEntity
		if msg == "invalid JSON body" {
			status = http.StatusBadRequest
		}
		s.writeError(w, status, msg)
		return
	}

	check, err := s.api.UpdatePostureCheck(r.Context(), token, id, req)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, check)
}

func (s *Server) deletePostureCheck(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	if err := s.api.DeletePostureCheck(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
