package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetCurrentUserPermissionsReturnsRoleAndModules(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users/current" {
			t.Errorf("path = %q, want /api/users/current", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{
			"id":"u1","name":"Ada","role":"network_admin","status":"active","auto_groups":[],
			"is_blocked":false,"pending_approval":false,
			"permissions":{"is_restricted":true,"modules":{"peers":{"read":true,"create":false,"update":false,"delete":false}}}
		}`)
	})

	rec := httptest.NewRecorder()
	srv.getCurrentUserPermissions(rec, authed(http.MethodGet, "/api/bff/permissions", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got permissionsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Role != "network_admin" {
		t.Errorf("Role = %q, want network_admin", got.Role)
	}
	if !got.Permissions.IsRestricted {
		t.Error("IsRestricted = false, want true")
	}
	if !got.Permissions.Modules["peers"]["read"] {
		t.Error("Modules[peers][read] = false, want true")
	}
}
