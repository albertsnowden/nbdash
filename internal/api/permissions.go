package api

import (
	"net/http"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

type permissionsResponse struct {
	Role        string                `json:"role"`
	Permissions nbapi.UserPermissions `json:"permissions"`
}

func (s *Server) getCurrentUserPermissions(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	user, err := s.api.CurrentUser(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, permissionsResponse{Role: user.Role, Permissions: user.Permissions})
}
