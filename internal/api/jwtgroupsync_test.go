package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const accountWithSettings = `{
	"id":"acc1","domain":"example.com",
	"settings":{
		"peer_login_expiration_enabled":true,"peer_login_expiration":43200,
		"jwt_groups_enabled":false,"jwt_groups_claim_name":"","jwt_allow_groups":[],
		"dns_domain":"internal.example.com"
	}
}`

func TestGetJWTGroupSyncReturnsParsedFields(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/accounts" {
			t.Errorf("path = %q, want /api/accounts", r.URL.Path)
		}
		_, _ = io.WriteString(w, "["+accountWithSettings+"]")
	})

	rec := httptest.NewRecorder()
	srv.getJWTGroupSync(rec, authed(http.MethodGet, "/api/bff/settings/jwt-group-sync", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got jwtGroupSync
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Enabled {
		t.Error("Enabled = true, want false")
	}
}

func TestUpdateJWTGroupSyncRequiresClaimNameWhenEnabled(t *testing.T) {
	var putCalled bool
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			putCalled = true
		}
	})

	req := authed(http.MethodPut, "/api/bff/settings/jwt-group-sync",
		strings.NewReader(`{"jwt_groups_enabled":true,"jwt_groups_claim_name":"","jwt_allow_groups":[]}`))
	rec := httptest.NewRecorder()
	srv.updateJWTGroupSync(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body=%s", rec.Code, rec.Body.String())
	}
	if putCalled {
		t.Error("enabling with no claim name must not reach the Management API")
	}
}

// TestUpdateJWTGroupSyncPreservesOtherSettings is the load-bearing test:
// the account has 4+ settings fields, but the request only carries the 3
// JWT ones — updateJWTGroupSync must fetch current settings and merge,
// not replace the whole settings object with just the JWT fields.
func TestUpdateJWTGroupSyncPreservesOtherSettings(t *testing.T) {
	var gotSettings map[string]any
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/accounts":
			_, _ = io.WriteString(w, "["+accountWithSettings+"]")
		case r.Method == http.MethodPut && r.URL.Path == "/api/accounts/acc1":
			var body struct {
				Settings map[string]any `json:"settings"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			gotSettings = body.Settings
			_, _ = io.WriteString(w, `{"id":"acc1","domain":"example.com","settings":`+mustMarshal(body.Settings)+`}`)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	req := authed(http.MethodPut, "/api/bff/settings/jwt-group-sync",
		strings.NewReader(`{"jwt_groups_enabled":true,"jwt_groups_claim_name":"roles","jwt_allow_groups":["Administrators"]}`))
	rec := httptest.NewRecorder()
	srv.updateJWTGroupSync(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if gotSettings["dns_domain"] != "internal.example.com" {
		t.Errorf("upstream dns_domain = %v, want internal.example.com (preserved)", gotSettings["dns_domain"])
	}
	if gotSettings["peer_login_expiration"] != float64(43200) {
		t.Errorf("upstream peer_login_expiration = %v, want 43200 (preserved)", gotSettings["peer_login_expiration"])
	}
	if gotSettings["jwt_groups_enabled"] != true {
		t.Errorf("upstream jwt_groups_enabled = %v, want true (patched)", gotSettings["jwt_groups_enabled"])
	}
	if gotSettings["jwt_groups_claim_name"] != "roles" {
		t.Errorf("upstream jwt_groups_claim_name = %v, want roles (patched)", gotSettings["jwt_groups_claim_name"])
	}
}

func mustMarshal(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
