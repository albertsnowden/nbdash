package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/albertsnowden/nbdash/internal/auth"
	"github.com/albertsnowden/nbdash/internal/config"
	"github.com/albertsnowden/nbdash/internal/nbapi"
)

// fakeIssuer serves just enough OIDC discovery for auth.New to succeed. No
// token endpoint is needed because the sessions below carry unexpired
// tokens — same helper shape as internal/handlers/peers_test.go's, kept as
// its own copy since test helpers aren't exported across packages.
func fakeIssuer(t *testing.T) string {
	t.Helper()

	mux := http.NewServeMux()
	var issuer string
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

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	issuer = srv.URL
	return srv.URL
}

func newTestServer(t *testing.T, apiHandler http.HandlerFunc) *Server {
	t.Helper()

	apiSrv := httptest.NewServer(apiHandler)
	t.Cleanup(apiSrv.Close)

	cfg := &config.Config{
		APIOrigin:        apiSrv.URL,
		AuthAuthority:    fakeIssuer(t),
		AuthClientID:     "client",
		AuthClientSecret: "secret",
		RedirectURI:      "http://localhost:8080/auth/callback",
		TokenSource:      config.AccessToken,
	}

	authenticator, err := auth.New(context.Background(), cfg, auth.NewStore(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("auth.New: %v", err)
	}

	return New(cfg, authenticator, nbapi.New(apiSrv.URL), slog.New(slog.DiscardHandler))
}

// authed returns a request carrying a session with an unexpired token,
// attached directly to the context the way auth.Middleware would after a
// real cookie round-trip — same shape as
// internal/handlers/peers_test.go's authed().
func authed(method, target string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, target, body)
	sess := &auth.Session{
		ID:    "sess-1",
		Email: "ada@example.com",
		Name:  "Ada Lovelace",
		Token: &oauth2.Token{
			AccessToken: "test-token",
			TokenType:   "Bearer",
			Expiry:      time.Now().Add(time.Hour),
		},
	}
	return req.WithContext(auth.NewContext(req.Context(), sess))
}

const twoPeers = `[
	{"id":"p1","name":"gateway-01","ip":"100.92.0.1","hostname":"gw01","connected":true,"os":"Ubuntu 22.04"},
	{"id":"p2","name":"laptop-02","ip":"100.92.0.2","hostname":"lt02","connected":false,"os":"Darwin 14.5"}
]`

const onePeer = `{"id":"p1","name":"gateway-01","ip":"100.92.0.1","hostname":"gw01","connected":true,"os":"Ubuntu 22.04","ssh_enabled":true}`

func TestListPeersReturnsCountsAndFilteredList(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/peers" {
			t.Errorf("path = %q, want /api/peers", r.URL.Path)
		}
		_, _ = io.WriteString(w, twoPeers)
	})

	rec := httptest.NewRecorder()
	srv.listPeers(rec, authed(http.MethodGet, "/api/bff/peers", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got peersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 2 || got.ConnectedCount != 1 {
		t.Errorf("Total/ConnectedCount = %d/%d, want 2/1", got.Total, got.ConnectedCount)
	}
	if len(got.Peers) != 2 {
		t.Fatalf("Peers = %d, want 2", len(got.Peers))
	}
}

func TestListPeersFiltersByQuery(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, twoPeers)
	})

	rec := httptest.NewRecorder()
	srv.listPeers(rec, authed(http.MethodGet, "/api/bff/peers?q=laptop", nil))

	var got peersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 2 {
		t.Errorf("Total = %d, want 2 (account-wide, unaffected by the search)", got.Total)
	}
	if len(got.Peers) != 1 || got.Peers[0].Name != "laptop-02" {
		t.Errorf("Peers = %+v, want just laptop-02", got.Peers)
	}
}

func TestGetPeerReturnsDecodedPeer(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/peers/p1" {
			t.Errorf("path = %q, want /api/peers/p1", r.URL.Path)
		}
		_, _ = io.WriteString(w, onePeer)
	})

	req := authed(http.MethodGet, "/api/bff/peers/p1", nil)
	req.SetPathValue("id", "p1")

	rec := httptest.NewRecorder()
	srv.getPeer(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got nbapi.Peer
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Name != "gateway-01" || !got.SSHEnabled {
		t.Errorf("got = %+v, want gateway-01 with SSH enabled", got)
	}
}

func TestUpdatePeerSendsFieldsAndReturnsResult(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody nbapi.PeerRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"p1","name":"renamed","ip":"100.92.0.1","hostname":"gw01"}`)
	})

	req := authed(http.MethodPut, "/api/bff/peers/p1",
		strings.NewReader(`{"name":"renamed","ssh_enabled":true,"login_expiration_enabled":true,"inactivity_expiration_enabled":false}`))
	req.SetPathValue("id", "p1")

	rec := httptest.NewRecorder()
	srv.updatePeer(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if gotMethod != http.MethodPut || gotPath != "/api/peers/p1" {
		t.Errorf("method/path = %s %s, want PUT /api/peers/p1", gotMethod, gotPath)
	}
	if gotBody.Name != "renamed" || !gotBody.SSHEnabled || !gotBody.LoginExpirationEnabled {
		t.Errorf("unexpected upstream body: %+v", gotBody)
	}

	var got nbapi.Peer
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Name != "renamed" {
		t.Errorf("Name = %q, want renamed", got.Name)
	}
}

func TestUpdatePeerRejectsEmptyName(t *testing.T) {
	var putCalled bool
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		putCalled = true
	})

	req := authed(http.MethodPut, "/api/bff/peers/p1", strings.NewReader(`{"name":"   "}`))
	req.SetPathValue("id", "p1")

	rec := httptest.NewRecorder()
	srv.updatePeer(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
	if putCalled {
		t.Error("an empty name must not reach the Management API")
	}

	var env errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if env.Error.Message != "Name cannot be empty." {
		t.Errorf("error message = %q, want the validation message", env.Error.Message)
	}
}

func TestUpdatePeerRejectsInvalidJSON(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("invalid JSON must not reach the Management API")
	})

	req := authed(http.MethodPut, "/api/bff/peers/p1", strings.NewReader(`not json`))
	req.SetPathValue("id", "p1")

	rec := httptest.NewRecorder()
	srv.updatePeer(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestDeletePeerReturnsNoContent(t *testing.T) {
	var gotMethod, gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	req := authed(http.MethodDelete, "/api/bff/peers/p1", nil)
	req.SetPathValue("id", "p1")

	rec := httptest.NewRecorder()
	srv.deletePeer(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/peers/p1" {
		t.Errorf("method/path = %s %s, want DELETE /api/peers/p1", gotMethod, gotPath)
	}
}

func TestUnauthorizedRequestGetsJSON401(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("should never reach the Management API without a session")
	})

	rec := httptest.NewRecorder()
	// No session attached — httptest.NewRequest alone, not authed().
	srv.listPeers(rec, httptest.NewRequest(http.MethodGet, "/api/bff/peers", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

func TestUnauthorizedManagementAPIResponseGetsJSON401(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"code":401,"message":"token invalid"}`)
	})

	rec := httptest.NewRecorder()
	srv.listPeers(rec, authed(http.MethodGet, "/api/bff/peers", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
