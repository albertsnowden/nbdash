package api

import "net/http"

func (s *Server) getCurrentAccount(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	account, err := s.api.CurrentAccount(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	s.writeJSON(w, http.StatusOK, account)
}

// deleteAccount is the Danger Zone's one action — permanent, whole-account
// deletion. There is deliberately no confirmation text/typed-domain check
// here: that belongs in the UI, not the API, and the Management API itself
// is the actual authority on who's allowed to call this (owner-only).
func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	current, err := s.api.CurrentAccount(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	if err := s.api.DeleteAccount(r.Context(), token, current.ID); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
