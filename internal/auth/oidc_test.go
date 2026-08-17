package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/albertsnowden/nbdash/internal/config"
)

// The ?next= parameter is attacker-controllable: anyone can hand a user a link
// to /login?next=... So it must only ever produce a same-site path.
func TestIsSafeRedirect(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"/peers", true},
		{"/peers/p1?q=x", true},
		{"/", true},

		{"", false},
		{"//evil.example/phish", false}, // protocol-relative
		{"https://evil.example", false}, // absolute
		{"http://evil.example", false},  //
		{"javascript:alert(1)", false},  //
		{"peers", false},                // relative, would resolve oddly
		{"\\\\evil.example", false},     // backslash form

		// Browsers normalise backslashes to forward slashes inside the
		// authority component (WHATWG URL), so these reach the same off-origin
		// destination as "//evil.example" despite starting with a single "/".
		{`/\evil.example`, false},
		{`/\/evil.example`, false},
		{`/%5Cevil.example`, false}, // percent-encoded backslash

		// Control characters are stripped by browsers before parsing, so a tab
		// hides the second slash from a naive check; CR/LF additionally split
		// the Location header.
		{"/\tevil.example", false},
		{"/\r\nSet-Cookie: x=y", false},
	}

	for _, tc := range tests {
		if got := isSafeRedirect(tc.in); got != tc.want {
			t.Errorf("isSafeRedirect(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestTakePendingIsSingleUse(t *testing.T) {
	store := NewStore()
	store.savePending("state-1", &pending{verifier: "v", nonce: "n", redirect: "/peers"})

	if _, ok := store.takePending("state-1"); !ok {
		t.Fatal("first take should succeed")
	}
	// Replaying the same state must fail, so a captured callback URL cannot be
	// used twice.
	if _, ok := store.takePending("state-1"); ok {
		t.Error("second take should fail")
	}
}

// RequestToken is the shared entry point internal/api's JSON handlers use
// instead of duplicating internal/handlers' token(w,r) — this pins its two
// outcomes independent of which package ends up calling it.
func TestRequestTokenReturnsErrNoSessionWithoutOne(t *testing.T) {
	var refreshes atomic.Int64
	authenticator, err := New(context.Background(), &config.Config{
		AuthAuthority:    countingIssuer(t, &refreshes),
		AuthClientID:     "client",
		AuthClientSecret: "secret",
		RedirectURI:      "http://localhost:8080/auth/callback",
		TokenSource:      config.AccessToken,
	}, NewStore(), testLogger())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := authenticator.RequestToken(context.Background()); !errors.Is(err, ErrNoSession) {
		t.Errorf("RequestToken with no session in context = %v, want ErrNoSession", err)
	}
}

func TestRequestTokenResolvesAnAttachedSession(t *testing.T) {
	var refreshes atomic.Int64
	issuer := countingIssuer(t, &refreshes)

	authenticator, err := New(context.Background(), &config.Config{
		AuthAuthority:    issuer,
		AuthClientID:     "client",
		AuthClientSecret: "secret",
		RedirectURI:      "http://localhost:8080/auth/callback",
		TokenSource:      config.AccessToken,
	}, NewStore(), testLogger())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	sess := &Session{
		ID: "s1",
		Token: &oauth2.Token{
			AccessToken: "still-valid",
			TokenType:   "Bearer",
			Expiry:      time.Now().Add(time.Hour),
		},
	}
	ctx := NewContext(context.Background(), sess)

	token, err := authenticator.RequestToken(ctx)
	if err != nil {
		t.Fatalf("RequestToken: %v", err)
	}
	if token != "still-valid" {
		t.Errorf("token = %q, want the session's unexpired access token", token)
	}
	if got := refreshes.Load(); got != 0 {
		t.Errorf("refreshes = %d, want 0 — the token wasn't expired", got)
	}
}

func TestSessionLifecycle(t *testing.T) {
	store := NewStore()
	sess := &Session{ID: "abc", Email: "ada@example.com"}

	store.save(sess)
	got, ok := store.get("abc")
	if !ok || got.Email != "ada@example.com" {
		t.Fatalf("get returned %+v, %v", got, ok)
	}

	store.delete("abc")
	if _, ok := store.get("abc"); ok {
		t.Error("session should be gone after delete")
	}
}

// countingIssuer serves OIDC discovery plus a token endpoint that tallies how
// many refreshes it is asked to perform.
func countingIssuer(t *testing.T, refreshes *atomic.Int64) string {
	t.Helper()

	var issuer string
	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/oauth/token",
			"jwks_uri":                              issuer + "/.well-known/jwks.json",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})

	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
		n := refreshes.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"access_token":"access-%d","token_type":"Bearer",`+
			`"expires_in":3600,"refresh_token":"refresh-%d","id_token":"id-%d"}`, n, n, n)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	issuer = srv.URL
	return srv.URL
}

// BearerToken mutates the shared *Session that store.get hands to every
// concurrent request. The SPA's React Query hooks routinely fire several
// /api/bff/... calls in parallel on one page load, so two handlers
// routinely refresh the same session at the same instant.
func TestBearerTokenIsRaceFree(t *testing.T) {
	var refreshes atomic.Int64
	issuer := countingIssuer(t, &refreshes)

	store := NewStore()
	authenticator, err := New(context.Background(), &config.Config{
		AuthAuthority:    issuer,
		AuthClientID:     "client",
		AuthClientSecret: "secret",
		RedirectURI:      "http://localhost:8080/auth/callback",
		TokenSource:      config.AccessToken,
	}, store, testLogger())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// An already-expired token means every caller arrives wanting a refresh.
	sess := &Session{
		ID:      "s1",
		IDToken: "id-0",
		Token: &oauth2.Token{
			AccessToken:  "stale",
			TokenType:    "Bearer",
			RefreshToken: "refresh-0",
			Expiry:       time.Now().Add(-time.Hour),
		},
	}
	store.save(sess)

	const n = 64
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, n)

	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start // release together, to overlap as tightly as possible
			if _, err := authenticator.BearerToken(context.Background(), sess); err != nil {
				errs <- err
			}
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("BearerToken: %v", err)
	}

	// Providers that rotate refresh tokens invalidate the previous one, so a
	// second concurrent refresh would spend an already-consumed token and log
	// the user out mid-session.
	if got := refreshes.Load(); got != 1 {
		t.Errorf("refreshes = %d, want exactly 1", got)
	}
}
