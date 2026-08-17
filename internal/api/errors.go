package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

// userMessage picks the text to send the client. Mirrors
// internal/handlers/errors.go's function of the same name — kept as its own
// small copy rather than an import from a package this migration retires at
// cutover (see the package doc comment).
func userMessage(err error) string {
	var apiErr *nbapi.Error
	if errors.As(err, &apiErr) && apiErr.Message != "" {
		return apiErr.Message
	}
	return "Could not reach the NetBird Management API. Please try again."
}

// accountLockoutReason recognizes the two account-level lockout messages the
// Management API sends on every request once they apply — not just the one
// action that triggered them (see management/server/permissions/manager.go's
// ValidateUserPermissions and shared/management/status/error.go's
// NewUserPendingApprovalError/NewUserBlockedError in the netbird server
// repo). Matched by substring, not exact equality: several call sites wrap
// the bare message before it reaches this client — e.g.
// management/server/account.go wraps it as "failed to validate user
// permissions: user is pending approval" — and which wrapper applies
// depends on which endpoint the request hit, not something worth pinning
// down call-site by call-site when the inner message is the stable part.
func accountLockoutReason(message string) string {
	switch {
	case strings.Contains(message, "user is pending approval"):
		return "pending_approval"
	case strings.Contains(message, "user is blocked"):
		return "blocked"
	default:
		return ""
	}
}

// fail turns an nbapi call's error into a JSON error response, the JSON
// equivalent of internal/handlers' fail(w,r,title,err). An expired or
// rejected token gets the same 401 treatment token() already gives a
// missing session, so the frontend's single 401-means-go-to-/login handling
// in apiFetch() covers both cases uniformly. A pending-approval or blocked
// account gets its Reason set so the frontend can redirect to one
// dedicated "account restricted" screen instead of leaking a raw 403
// message into whatever page happened to make the request — the user
// hasn't done anything wrong on that page, their whole account is locked.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	if nbapi.IsUnauthorized(err) {
		s.writeError(w, http.StatusUnauthorized, "unauthenticated")
		return
	}

	s.log.Error("request failed", "path", r.URL.Path, "error", err)

	status := http.StatusInternalServerError
	var apiErr *nbapi.Error
	if errors.As(err, &apiErr) && apiErr.Code >= 400 && apiErr.Code < 600 {
		status = apiErr.Code
	}

	msg := userMessage(err)
	s.writeErrorWithReason(w, status, msg, accountLockoutReason(msg))
}
