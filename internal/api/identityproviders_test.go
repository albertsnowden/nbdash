package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

const twoIdentityProviders = `[
	{"id":"idp-1","type":"oidc","name":"Corp SSO","issuer":"https://login.example.com","client_id":"abc"},
	{"id":"idp-2","type":"google","name":"Google","client_id":"xyz"}
]`

func identityProviderBody(overrides map[string]string) string {
	fields := map[string]string{
		"type":          "oidc",
		"name":          "Corp SSO",
		"issuer":        "https://login.example.com",
		"client_id":     "abc",
		"client_secret": "shh",
	}
	for k, v := range overrides {
		fields[k] = v
	}
	b, _ := json.Marshal(fields)
	return string(b)
}

func TestListIdentityProvidersReturnsAll(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, twoIdentityProviders)
	})

	rec := httptest.NewRecorder()
	srv.listIdentityProviders(rec, authed(http.MethodGet, "/api/bff/settings/identity-providers", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got identityProvidersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 2 {
		t.Errorf("Total = %d, want 2", got.Total)
	}
}

func TestCreateIdentityProviderSendsFields(t *testing.T) {
	var gotBody nbapi.IdentityProviderRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"idp-1","type":"oidc","name":"Corp SSO","client_id":"abc"}`)
	})

	rec := httptest.NewRecorder()
	srv.createIdentityProvider(rec, authed(http.MethodPost, "/api/bff/settings/identity-providers",
		strings.NewReader(identityProviderBody(nil))))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	if gotBody.Type != nbapi.IdentityProviderOIDC || gotBody.ClientSecret != "shh" {
		t.Errorf("unexpected request body: %+v", gotBody)
	}
}

func TestCreateIdentityProviderRejectsInvalidType(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("an invalid type must not reach the Management API")
	})

	body := identityProviderBody(map[string]string{"type": "not-a-real-type"})
	rec := httptest.NewRecorder()
	srv.createIdentityProvider(rec, authed(http.MethodPost, "/api/bff/settings/identity-providers", strings.NewReader(body)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestCreateIdentityProviderRejectsMissingIssuerForOIDC(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("a missing issuer must not reach the Management API for a type without a built-in one")
	})

	body := identityProviderBody(map[string]string{"issuer": ""})
	rec := httptest.NewRecorder()
	srv.createIdentityProvider(rec, authed(http.MethodPost, "/api/bff/settings/identity-providers", strings.NewReader(body)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

// Google and Microsoft use built-in issuer endpoints — the form must not
// require one for these types.
func TestCreateIdentityProviderAllowsMissingIssuerForGoogle(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":"idp-2","type":"google","name":"Google","client_id":"xyz"}`)
	})

	body := identityProviderBody(map[string]string{"type": "google", "issuer": "", "client_id": "xyz"})
	rec := httptest.NewRecorder()
	srv.createIdentityProvider(rec, authed(http.MethodPost, "/api/bff/settings/identity-providers", strings.NewReader(body)))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
}

func TestCreateIdentityProviderRejectsMissingSecret(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("a missing secret must not reach the Management API on create")
	})

	body := identityProviderBody(map[string]string{"client_secret": ""})
	rec := httptest.NewRecorder()
	srv.createIdentityProvider(rec, authed(http.MethodPost, "/api/bff/settings/identity-providers", strings.NewReader(body)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

// A blank secret on update must still reach the API — that's exactly what
// preserves the currently stored secret server-side.
func TestUpdateIdentityProviderAllowsBlankSecret(t *testing.T) {
	var putCalled bool
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		putCalled = true
		_, _ = io.WriteString(w, `{"id":"idp-1","type":"oidc","name":"Corp SSO","client_id":"abc"}`)
	})

	body := identityProviderBody(map[string]string{"client_secret": ""})
	req := authed(http.MethodPut, "/api/bff/settings/identity-providers/idp-1", strings.NewReader(body))
	req.SetPathValue("id", "idp-1")

	rec := httptest.NewRecorder()
	srv.updateIdentityProvider(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if !putCalled {
		t.Error("a blank secret on update should still reach the Management API")
	}
}

func TestDeleteIdentityProviderReturnsNoContent(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	req := authed(http.MethodDelete, "/api/bff/settings/identity-providers/idp-1", nil)
	req.SetPathValue("id", "idp-1")

	rec := httptest.NewRecorder()
	srv.deleteIdentityProvider(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotPath != "/api/identity-providers/idp-1" {
		t.Errorf("path = %q, want /api/identity-providers/idp-1", gotPath)
	}
}
