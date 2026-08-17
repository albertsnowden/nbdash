package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/albertsnowden/nbdash/internal/config"
)

// rpInitiatedIssuer serves OIDC discovery advertising end_session_endpoint, so
// Logout has something to redirect to.
func rpInitiatedIssuer(t *testing.T) string {
	t.Helper()

	var issuer string
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/oauth/token",
			"end_session_endpoint":                  issuer + "/endsession",
			"jwks_uri":                              issuer + "/.well-known/jwks.json",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	issuer = srv.URL
	return srv.URL
}

// TestLogoutEndsProviderSession pins the fix for "sign out doesn't work": a
// local-only logout leaves the provider's own SSO session alive, so the very
// next protected page load silently re-authenticates through it with no
// prompt — indistinguishable, to the user, from sign-out doing nothing. When
// the provider advertises end_session_endpoint, Logout must send the browser
// there (with the session identified via id_token_hint) rather than straight
// back to the dashboard.
func TestLogoutEndsProviderSession(t *testing.T) {
	issuer := rpInitiatedIssuer(t)
	store := NewStore()
	a, err := New(context.Background(), &config.Config{
		AuthAuthority:    issuer,
		AuthClientID:     "client",
		AuthClientSecret: "secret",
		RedirectURI:      "http://localhost:8080/auth/callback",
		TokenSource:      config.AccessToken,
	}, store, testLogger())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	store.save(&Session{ID: "sess-1", IDToken: "raw-id-token"})

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: a.cookieName(), Value: "sess-1"})
	rec := httptest.NewRecorder()

	a.Logout(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}

	loc, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatalf("Location %q did not parse: %v", rec.Header().Get("Location"), err)
	}
	if got := loc.Scheme + "://" + loc.Host + loc.Path; got != issuer+"/endsession" {
		t.Errorf("redirected to %q, want the provider's end_session_endpoint (%s/endsession)", got, issuer)
	}
	q := loc.Query()
	if got := q.Get("id_token_hint"); got != "raw-id-token" {
		t.Errorf("id_token_hint = %q, want %q", got, "raw-id-token")
	}
	if got := q.Get("client_id"); got != "client" {
		t.Errorf("client_id = %q, want %q", got, "client")
	}
	if got := q.Get("post_logout_redirect_uri"); got != "http://localhost:8080/" {
		t.Errorf("post_logout_redirect_uri = %q, want %q", got, "http://localhost:8080/")
	}

	if _, ok := store.get("sess-1"); ok {
		t.Error("session still present in the store after logout")
	}
	assertCookieCleared(t, rec, a.cookieName())
}

// TestLogoutFallsBackToLocalWhenProviderHasNoEndSession covers providers that
// do not advertise end_session_endpoint: Logout must keep behaving exactly as
// it did before RP-Initiated Logout support existed, dropping only the local
// session and sending the browser back to the dashboard's own front page.
func TestLogoutFallsBackToLocalWhenProviderHasNoEndSession(t *testing.T) {
	var refreshes atomic.Int64
	issuer := countingIssuer(t, &refreshes) // discovery with no end_session_endpoint

	store := NewStore()
	a, err := New(context.Background(), &config.Config{
		AuthAuthority:    issuer,
		AuthClientID:     "client",
		AuthClientSecret: "secret",
		RedirectURI:      "http://localhost:8080/auth/callback",
		TokenSource:      config.AccessToken,
	}, store, testLogger())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	store.save(&Session{ID: "sess-1", IDToken: "raw-id-token"})

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: a.cookieName(), Value: "sess-1"})
	rec := httptest.NewRecorder()

	a.Logout(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if got := rec.Header().Get("Location"); got != "/" {
		t.Errorf("Location = %q, want %q", got, "/")
	}
	if _, ok := store.get("sess-1"); ok {
		t.Error("session still present in the store after logout")
	}
	assertCookieCleared(t, rec, a.cookieName())
}

func assertCookieCleared(t *testing.T, rec *httptest.ResponseRecorder, name string) {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			if c.MaxAge >= 0 {
				t.Errorf("cookie %s MaxAge = %d, want negative (expired)", name, c.MaxAge)
			}
			return
		}
	}
	t.Errorf("cookie %s was not set on the logout response", name)
}
