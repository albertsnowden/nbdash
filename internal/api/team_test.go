package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const twoUsers = `[
	{"id":"u1","name":"Ada","email":"ada@example.com","role":"admin","status":"active","auto_groups":[]},
	{"id":"u2","name":"Bea","email":"bea@example.com","role":"user","status":"invited","auto_groups":[]}
]`

func TestListUsersFiltersServiceUserFalse(t *testing.T) {
	var gotQuery string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = io.WriteString(w, twoUsers)
	})

	rec := httptest.NewRecorder()
	srv.listUsers(rec, authed(http.MethodGet, "/api/bff/team/users", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if gotQuery != "service_user=false" {
		t.Errorf("query = %q, want service_user=false", gotQuery)
	}
	var got usersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 2 {
		t.Errorf("Total = %d, want 2", got.Total)
	}
}

func TestListServiceUsersFiltersServiceUserTrue(t *testing.T) {
	var gotQuery string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = io.WriteString(w, `[]`)
	})

	rec := httptest.NewRecorder()
	srv.listServiceUsers(rec, authed(http.MethodGet, "/api/bff/team/service-users", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if gotQuery != "service_user=true" {
		t.Errorf("query = %q, want service_user=true", gotQuery)
	}
}

func TestInviteUserRequiresEmail(t *testing.T) {
	var posted bool
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		posted = true
	})

	body := `{"name":"Ada","email":"","role":"admin"}`
	rec := httptest.NewRecorder()
	srv.inviteUser(rec, authed(http.MethodPost, "/api/bff/team/users", strings.NewReader(body)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
	if posted {
		t.Error("a missing email must not reach the Management API")
	}
}

func TestInviteUserRejectsInvalidRole(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("an invalid role must not reach the Management API")
	})

	body := `{"name":"Ada","email":"ada@example.com","role":"owner"}`
	rec := httptest.NewRecorder()
	srv.inviteUser(rec, authed(http.MethodPost, "/api/bff/team/users", strings.NewReader(body)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422 (owner is never a valid invite role)", rec.Code)
	}
}

func TestCreateServiceUserDoesNotRequireEmail(t *testing.T) {
	var gotBody map[string]any
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"su1","name":"ci-bot","role":"admin","is_service_user":true}`)
	})

	body := `{"name":"ci-bot","role":"admin"}`
	rec := httptest.NewRecorder()
	srv.createServiceUser(rec, authed(http.MethodPost, "/api/bff/team/service-users", strings.NewReader(body)))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	if isService, _ := gotBody["is_service_user"].(bool); !isService {
		t.Errorf("is_service_user sent = %v, want true", gotBody["is_service_user"])
	}
	if _, hasEmail := gotBody["email"]; hasEmail {
		t.Errorf("email should not be sent for a service user, got %v", gotBody["email"])
	}
}

func TestUpdateUserAllowsOwnerRole(t *testing.T) {
	// The server, not this handler, is the authority on who may actually set
	// "owner" — client-side validation must not pre-emptively block it.
	var gotBody map[string]any
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"u1","name":"Ada","role":"owner"}`)
	})

	req := authed(http.MethodPut, "/api/bff/team/users/u1", strings.NewReader(`{"role":"owner","auto_groups":[]}`))
	req.SetPathValue("id", "u1")

	rec := httptest.NewRecorder()
	srv.updateUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if gotBody["role"] != "owner" {
		t.Errorf("role sent = %v, want owner", gotBody["role"])
	}
}

func TestUpdateUserRejectsGarbageRole(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("a nonsense role must not reach the Management API")
	})

	req := authed(http.MethodPut, "/api/bff/team/users/u1", strings.NewReader(`{"role":"superuser"}`))
	req.SetPathValue("id", "u1")

	rec := httptest.NewRecorder()
	srv.updateUser(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestApproveUserReturnsUpdated(t *testing.T) {
	var gotMethod, gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = io.WriteString(w, `{"id":"u1","name":"Ada","pending_approval":false}`)
	})

	req := authed(http.MethodPost, "/api/bff/team/users/u1/approve", nil)
	req.SetPathValue("id", "u1")

	rec := httptest.NewRecorder()
	srv.approveUser(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if gotMethod != http.MethodPost || gotPath != "/api/users/u1/approve" {
		t.Errorf("method/path = %s %s, want POST /api/users/u1/approve", gotMethod, gotPath)
	}
}

func TestRejectUserReturnsNoContent(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	req := authed(http.MethodDelete, "/api/bff/team/users/u1/reject", nil)
	req.SetPathValue("id", "u1")

	rec := httptest.NewRecorder()
	srv.rejectUser(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotPath != "/api/users/u1/reject" {
		t.Errorf("path = %q, want /api/users/u1/reject", gotPath)
	}
}

func TestCreatePATReturnsPlaintext(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/users/su1/tokens" {
			t.Errorf("path = %q, want /api/users/su1/tokens", r.URL.Path)
		}
		_, _ = io.WriteString(w, `{"plain_token":"nbp_PLAINTEXT","personal_access_token":{"id":"pat1","name":"ci"}}`)
	})

	req := authed(http.MethodPost, "/api/bff/team/service-users/su1/tokens",
		strings.NewReader(`{"name":"ci","expires_in":30}`))
	req.SetPathValue("id", "su1")

	rec := httptest.NewRecorder()
	srv.createPAT(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "nbp_PLAINTEXT") {
		t.Errorf("response missing plaintext token: %s", rec.Body.String())
	}
}

func TestCreatePATRejectsBadExpiry(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("an out-of-range expiry must not reach the Management API")
	})

	req := authed(http.MethodPost, "/api/bff/team/service-users/su1/tokens",
		strings.NewReader(`{"name":"ci","expires_in":9999}`))
	req.SetPathValue("id", "su1")

	rec := httptest.NewRecorder()
	srv.createPAT(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestDeletePATUsesNestedPath(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	req := authed(http.MethodDelete, "/api/bff/team/service-users/su1/tokens/pat1", nil)
	req.SetPathValue("id", "su1")
	req.SetPathValue("tokenID", "pat1")

	rec := httptest.NewRecorder()
	srv.deletePAT(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotPath != "/api/users/su1/tokens/pat1" {
		t.Errorf("path = %q, want /api/users/su1/tokens/pat1", gotPath)
	}
}
