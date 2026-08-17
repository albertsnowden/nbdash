package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/albertsnowden/nbdash/internal/auth"
)

// writeJSON encodes v as the response body. Errors from Encode can only
// happen after the status line is already written, so there is nothing left
// to do but log — unlike internal/handlers' render(), which buffers first
// specifically to still be able to fall back to a clean 500. JSON encoding
// of the small, hand-shaped DTOs this package emits does not fail in
// practice, so that extra buffering isn't worth carrying here too.
func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	// Every /api/bff/... response goes through here, success and error alike
	// — including the 401 body an unauthenticated request gets. Without this,
	// a caching layer sitting in front of the dashboard (a CDN, a reverse
	// proxy with proxy_cache on) is free to store a response keyed only by
	// URL and hand it back to the next visitor regardless of their own
	// session cookie — an authenticated GET's data leaking to whoever asks
	// next, or a stale 401 masking a since-established session. Neither is
	// hypothetical for a GET-heavy JSON API sitting behind an "edge" domain.
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.log.Error("could not encode JSON response", "error", err)
	}
}

type errorEnvelope struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		// Reason is an optional machine-readable discriminator for a handful
		// of account-level lockout conditions (see errors.go's
		// accountLockoutReason) the frontend needs to react to uniformly
		// across every page, not just show inline — e.g. redirecting to a
		// dedicated "account restricted" screen the same way a 401 redirects
		// to /login. Empty for every other error; do not add new reasons
		// speculatively, only for conditions the frontend actually branches
		// on.
		Reason string `json:"reason,omitempty"`
	} `json:"error"`
}

func (s *Server) writeError(w http.ResponseWriter, status int, message string) {
	s.writeErrorWithReason(w, status, message, "")
}

func (s *Server) writeErrorWithReason(w http.ResponseWriter, status int, message, reason string) {
	var env errorEnvelope
	env.Error.Code = status
	env.Error.Message = message
	env.Error.Reason = reason
	s.writeJSON(w, status, env)
}

// token resolves the bearer credential for the current request, the JSON
// equivalent of internal/handlers' token(w,r). A fetch() has no concept of
// following a custom redirect header the way htmx's HX-Redirect does, so
// the failure response here is a plain 401 body — it's the frontend's
// shared apiFetch() wrapper that turns that into a navigation to /login,
// not this server.
func (s *Server) token(w http.ResponseWriter, r *http.Request) (string, bool) {
	tok, err := s.auth.RequestToken(r.Context())
	if err != nil {
		if !errors.Is(err, auth.ErrNoSession) {
			s.log.Warn("could not obtain bearer token", "error", err)
		}
		s.writeError(w, http.StatusUnauthorized, "unauthenticated")
		return "", false
	}
	return tok, true
}
