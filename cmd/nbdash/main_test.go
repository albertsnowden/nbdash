package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/albertsnowden/nbdash/internal/auth"
	"github.com/albertsnowden/nbdash/internal/config"
)

// discoveryStub serves just the OIDC discovery document, with a configurable
// PKCE advertisement — enough for auth.New's discovery + PKCE-support check.
// Nothing here ever reaches token exchange or ID token verification, so no
// signing key or jwks endpoint is needed.
func discoveryStub(t *testing.T, codeChallengeMethods []string) string {
	t.Helper()

	var issuer string
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		doc := map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/oauth/token",
			"jwks_uri":                              issuer + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}
		if codeChallengeMethods != nil {
			doc["code_challenge_methods_supported"] = codeChallengeMethods
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(doc)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	issuer = srv.URL
	return srv.URL
}

// A static misconfiguration — a public client against a provider with no PKCE
// support — must fail on the first attempt, not after the retry loop's full
// ~31s backoff (1+2+4+8+16s across 5 attempts). Asserting elapsed time makes
// that failure mode explicit rather than relying on an unrelated test timeout
// to eventually catch a regression.
func TestDiscoverWithRetryDoesNotRetryStaticPKCEMisconfiguration(t *testing.T) {
	issuer := discoveryStub(t, nil) // no code_challenge_methods_supported at all

	cfg := &config.Config{
		AuthAuthority:    issuer,
		AuthClientID:     "test-client",
		AuthClientSecret: "", // public mode
		RedirectURI:      "http://localhost:8080/auth/callback",
	}

	log := slog.New(slog.DiscardHandler)
	start := time.Now()
	_, err := discoverWithRetry(context.Background(), cfg, auth.NewStore(), log)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error: provider does not support PKCE")
	}
	if !errors.Is(err, auth.ErrPublicClientPKCEUnsupported) {
		t.Errorf("error = %v, want it to wrap auth.ErrPublicClientPKCEUnsupported", err)
	}
	// The retry loop's first backoff alone is 1s; failing well under that
	// confirms no retry was attempted.
	if elapsed > 500*time.Millisecond {
		t.Errorf("discoverWithRetry took %v — looks like it retried a static error", elapsed)
	}
}

// A transient failure (provider simply not reachable yet) is exactly what
// this loop exists to survive — confirm that path still retries rather than
// bailing on the first attempt, which the fix above must not have broken.
func TestDiscoverWithRetryStillRetriesUnreachableProvider(t *testing.T) {
	// Bind a listener, then close it immediately: the address now refuses
	// connections, indistinguishable from "the provider container hasn't
	// started listening yet" from the client's side.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	unreachable := "http://" + listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	cfg := &config.Config{
		AuthAuthority:    unreachable,
		AuthClientID:     "test-client",
		AuthClientSecret: "secret",
		RedirectURI:      "http://localhost:8080/auth/callback",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	log := slog.New(slog.DiscardHandler)
	start := time.Now()
	_, err = discoverWithRetry(ctx, cfg, auth.NewStore(), log)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected an error: provider is unreachable")
	}
	// The loop should still be backing off (1s, then 2s) when the context
	// deadline cuts it short — i.e. it did not bail immediately like the
	// static-misconfiguration case above.
	if elapsed < 900*time.Millisecond {
		t.Errorf("discoverWithRetry returned after %v — expected it to retry at least once", elapsed)
	}
}
