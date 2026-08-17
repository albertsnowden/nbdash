package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPendingApprovalAndBlockedGetReason pins the two account-lockout
// messages the Management API sends verbatim (see accountLockoutReason's
// doc comment) to a stable Reason the frontend redirects on — any other
// 403/permission-denied message must NOT set Reason, since only these two
// conditions apply account-wide rather than to the one action that
// triggered them.
func TestPendingApprovalAndBlockedGetReason(t *testing.T) {
	cases := []struct {
		upstreamMessage string
		wantReason      string
	}{
		{"user is pending approval", "pending_approval"},
		{"user is blocked", "blocked"},
		// The real regression this pins: several Management API call sites
		// wrap the bare message (management/server/account.go's
		// "failed to validate user permissions: %w"), so an exact-equality
		// match against the bare string alone missed every one of these in
		// production — accountLockoutReason must match by substring.
		{"failed to validate user permissions: user is pending approval", "pending_approval"},
		{"failed to validate user permissions: user is blocked", "blocked"},
		{"insufficient permissions", ""},
		{"policy not found", ""},
	}

	for _, tc := range cases {
		t.Run(tc.upstreamMessage, func(t *testing.T) {
			srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = io.WriteString(w, `{"code":403,"message":"`+tc.upstreamMessage+`"}`)
			})

			rec := httptest.NewRecorder()
			srv.listPeers(rec, authed(http.MethodGet, "/api/bff/peers", nil))

			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403, body=%s", rec.Code, rec.Body.String())
			}
			var env errorEnvelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("decode error envelope: %v", err)
			}
			if env.Error.Reason != tc.wantReason {
				t.Errorf("Reason = %q, want %q", env.Error.Reason, tc.wantReason)
			}
			if env.Error.Message != tc.upstreamMessage {
				t.Errorf("Message = %q, want the upstream message preserved verbatim", env.Error.Message)
			}
		})
	}
}
