package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

func TestGetCurrentAccountReturnsFirstAccount(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/accounts" {
			t.Errorf("path = %q, want /api/accounts", r.URL.Path)
		}
		_, _ = io.WriteString(w, `[{"id":"acc1","domain":"example.com"}]`)
	})

	rec := httptest.NewRecorder()
	srv.getCurrentAccount(rec, authed(http.MethodGet, "/api/bff/account", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got nbapi.Account
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.ID != "acc1" || got.Domain != "example.com" {
		t.Errorf("got = %+v, want id=acc1 domain=example.com", got)
	}
}

func TestDeleteAccountUsesResolvedAccountID(t *testing.T) {
	var gotMethod, gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/accounts":
			_, _ = io.WriteString(w, `[{"id":"acc1","domain":"example.com"}]`)
		case "/api/accounts/acc1":
			gotMethod, gotPath = r.Method, r.URL.Path
			w.WriteHeader(http.StatusOK)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	})

	rec := httptest.NewRecorder()
	srv.deleteAccount(rec, authed(http.MethodDelete, "/api/bff/account", nil))

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", rec.Code, rec.Body.String())
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/accounts/acc1" {
		t.Errorf("method/path = %s %s, want DELETE /api/accounts/acc1", gotMethod, gotPath)
	}
}
