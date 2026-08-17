// Team covers Users and Service Users — both the same nbapi.User shape and
// mostly the same underlying /users REST endpoint (see nbapi.User's doc
// comment), split into separate route groups here the same way
// internal/handlers/team.go splits them into separate tabs, since inviting
// a regular user and creating a service user are different operations with
// different required fields.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

var validUserRoles = map[string]bool{
	"admin": true, "user": true, "billing_admin": true, "auditor": true, "network_admin": true,
}

type usersResponse struct {
	Users []nbapi.User `json:"users"`
	Total int          `json:"total"`
}

func (s *Server) listUsersFiltered(w http.ResponseWriter, r *http.Request, serviceUser bool) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	users, err := s.api.ListUsers(r.Context(), token, &serviceUser)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, usersResponse{Users: users, Total: len(users)})
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	s.listUsersFiltered(w, r, false)
}

func (s *Server) listServiceUsers(w http.ResponseWriter, r *http.Request) {
	s.listUsersFiltered(w, r, true)
}

type inviteUserRequest struct {
	Name       string   `json:"name"`
	Email      string   `json:"email"`
	Role       string   `json:"role"`
	AutoGroups []string `json:"auto_groups"`
}

// inviteUser sends a real invite through the configured IdP — Name and
// Email are both required (see nbapi.UserCreateRequest's doc comment).
func (s *Server) inviteUser(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req inviteUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Name cannot be empty.")
		return
	}
	email := strings.TrimSpace(req.Email)
	if email == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Email cannot be empty.")
		return
	}
	if !validUserRoles[req.Role] {
		s.writeError(w, http.StatusUnprocessableEntity, "Invalid role.")
		return
	}

	created, err := s.api.CreateUser(r.Context(), token, nbapi.UserCreateRequest{
		Email:      email,
		Name:       name,
		Role:       req.Role,
		AutoGroups: req.AutoGroups,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, created)
}

type createServiceUserRequest struct {
	Name       string   `json:"name"`
	Role       string   `json:"role"`
	AutoGroups []string `json:"auto_groups"`
}

func (s *Server) createServiceUser(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req createServiceUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Name cannot be empty.")
		return
	}
	if !validUserRoles[req.Role] {
		s.writeError(w, http.StatusUnprocessableEntity, "Invalid role.")
		return
	}

	created, err := s.api.CreateUser(r.Context(), token, nbapi.UserCreateRequest{
		Name:          name,
		Role:          req.Role,
		AutoGroups:    req.AutoGroups,
		IsServiceUser: true,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, created)
}

type updateUserRequest struct {
	Role       string   `json:"role"`
	AutoGroups []string `json:"auto_groups"`
	IsBlocked  bool     `json:"is_blocked"`
}

// updateUser handles both Users and Service Users — the server accepts the
// exact same PUT body for either (see nbapi.UserRequest's doc comment: only
// role, auto_groups and blocked status can ever change here). "owner" is
// accepted client-side — the server itself is the actual authority on who
// may set it and rejects an unauthorized attempt with a normal API error.
func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if !validUserRoles[req.Role] && req.Role != "owner" {
		s.writeError(w, http.StatusUnprocessableEntity, "Invalid role.")
		return
	}

	updated, err := s.api.UpdateUser(r.Context(), token, r.PathValue("id"), nbapi.UserRequest{
		Role:       req.Role,
		AutoGroups: req.AutoGroups,
		IsBlocked:  req.IsBlocked,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

// deleteUser handles both Users and Service Users. The server refuses to
// delete the account owner or let a user delete themselves; both surface as
// a normal API error.
func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeleteUser(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) resendInvite(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.InviteUser(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) approveUser(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	updated, err := s.api.ApproveUser(r.Context(), token, r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

func (s *Server) rejectUser(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.RejectUser(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type patsResponse struct {
	Tokens []nbapi.PersonalAccessToken `json:"tokens"`
}

func (s *Server) listPATs(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	pats, err := s.api.ListPATs(r.Context(), token, r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, patsResponse{Tokens: pats})
}

type createPATRequest struct {
	Name      string `json:"name"`
	ExpiresIn int    `json:"expires_in"`
}

// createPAT returns the plaintext token from a successful create — the
// only response from this API that ever contains it (see
// nbapi.PersonalAccessTokenGenerated's doc comment).
func (s *Server) createPAT(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	userID := r.PathValue("id")

	var req createPATRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Name cannot be empty.")
		return
	}
	if req.ExpiresIn < 1 || req.ExpiresIn > 365 {
		s.writeError(w, http.StatusUnprocessableEntity, "Expiration must be a number of days between 1 and 365.")
		return
	}

	generated, err := s.api.CreatePAT(r.Context(), token, userID, nbapi.PersonalAccessTokenRequest{
		Name: name, ExpiresIn: req.ExpiresIn,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, generated)
}

func (s *Server) deletePAT(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	if err := s.api.DeletePAT(r.Context(), token, r.PathValue("id"), r.PathValue("tokenID")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
