package api

import (
	"net/http"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

type eventsResponse struct {
	Events []nbapi.Event `json:"events"`
	Total  int           `json:"total"`
}

func (s *Server) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	events, err := s.api.ListAuditEvents(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	s.writeJSON(w, http.StatusOK, eventsResponse{Events: events, Total: len(events)})
}
