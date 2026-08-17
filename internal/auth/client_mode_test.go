package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/albertsnowden/nbdash/internal/config"
)

// pkceIssuer is a fake OIDC provider complete enough to drive the real
// authorization-code flow end to end, with two things countingIssuer (in
// oidc_test.go) does not need: a discovery document whose PKCE support is
// configurable, and a token endpoint that records exactly what credential
// material it received, so tests can assert a public client sends none.
type pkceIssuer struct {
	URL      string
	clientID string
	key      *rsa.PrivateKey

	// codeChallengeMethods becomes the discovery document's
	// code_challenge_methods_supported. nil omits the field entirely, which is
	// how a provider that predates PKCE would look.
	codeChallengeMethods []string

	mu    sync.Mutex
	codes map[string]struct{ challenge, method, nonce string }

	// tokenRequests records every /oauth/token request's credential material,
	// in arrival order, so tests can assert on exactly what a given exchange
	// sent — not just "was any request ever missing a secret".
	tokenRequests []tokenRequestRecord
}

type tokenRequestRecord struct {
	clientSecretParam string // from the POST body, "" if absent
	hasBasicAuth      bool
	codeVerifier      string
}

func newPKCEIssuer(t *testing.T, clientID string, codeChallengeMethods []string) *pkceIssuer {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	idp := &pkceIssuer{
		clientID:             clientID,
		key:                  key,
		codeChallengeMethods: codeChallengeMethods,
		codes:                map[string]struct{ challenge, method, nonce string }{},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		doc := map[string]any{
			"issuer":                                idp.URL,
			"authorization_endpoint":                idp.URL + "/authorize",
			"token_endpoint":                        idp.URL + "/oauth/token",
			"jwks_uri":                              idp.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		}
		if idp.codeChallengeMethods != nil {
			doc["code_challenge_methods_supported"] = idp.codeChallengeMethods
		}
		writeJSON(w, doc)
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"keys": []any{map[string]any{
			"kty": "RSA", "kid": "test", "use": "sig", "alg": "RS256",
			"n": base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	})

	// Auto-approving "consent screen": stashes the PKCE challenge and nonce
	// against a code, keyed by state, and bounces straight back.
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		code := "code-" + q.Get("state")

		idp.mu.Lock()
		idp.codes[code] = struct{ challenge, method, nonce string }{
			challenge: q.Get("code_challenge"),
			method:    q.Get("code_challenge_method"),
			nonce:     q.Get("nonce"),
		}
		idp.mu.Unlock()

		http.Redirect(w, r, q.Get("redirect_uri")+"?code="+code+"&state="+q.Get("state"), http.StatusFound)
	})

	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()

		_, _, hasBasic := r.BasicAuth()

		idp.mu.Lock()
		idp.tokenRequests = append(idp.tokenRequests, tokenRequestRecord{
			clientSecretParam: r.PostFormValue("client_secret"),
			hasBasicAuth:      hasBasic,
			codeVerifier:      r.PostFormValue("code_verifier"),
		})
		nonce := idp.codes[r.PostFormValue("code")].nonce
		idp.mu.Unlock()

		now := time.Now()
		writeJSON(w, map[string]any{
			"access_token": "access-token", "token_type": "Bearer", "expires_in": 3600,
			"refresh_token": "refresh-token",
			"id_token": idp.signJWT(t, map[string]any{
				"iss": idp.URL, "aud": clientID, "sub": "user-123",
				"email": "ada@example.com", "nonce": nonce,
				"iat": now.Unix(), "exp": now.Add(time.Hour).Unix(),
			}),
		})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	idp.URL = srv.URL
	return idp
}

func (i *pkceIssuer) signJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	seg := func(v any) string {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		return base64.RawURLEncoding.EncodeToString(b)
	}
	signing := seg(map[string]any{"alg": "RS256", "typ": "JWT", "kid": "test"}) + "." + seg(claims)
	digest := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, i.key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// newTestConfig builds a valid config for the given issuer and secret,
// leaving every other field at the values the rest of the suite already uses.
func newTestConfig(issuer, secret string) *config.Config {
	return &config.Config{
		AuthAuthority:    issuer,
		AuthClientID:     "test-client",
		AuthClientSecret: secret,
		RedirectURI:      "http://localhost:8080/auth/callback",
		AuthScopes:       []string{"openid", "profile", "email"},
		TokenSource:      config.AccessToken,
	}
}

// The whole point of this package's public-client support: the mode is
// decided by one thing, whether a secret was configured, and by nothing else.
func TestOIDCClientMode(t *testing.T) {
	tests := []struct {
		secret string
		want   config.OIDCClientMode
	}{
		{"", config.Public},
		{"s3cr3t", config.Confidential},
	}
	for _, tc := range tests {
		cfg := newTestConfig("https://idp.example", tc.secret)
		if got := cfg.OIDCClientMode(); got != tc.want {
			t.Errorf("secret=%q: OIDCClientMode() = %q, want %q", tc.secret, got, tc.want)
		}
	}
}

func TestNewSucceedsInPublicModeWhenProviderSupportsS256(t *testing.T) {
	idp := newPKCEIssuer(t, "test-client", []string{"S256"})
	cfg := newTestConfig(idp.URL, "")

	if _, err := New(context.Background(), cfg, NewStore(), testLogger()); err != nil {
		t.Fatalf("New: %v", err)
	}
}

// A public client has no secret, so PKCE — bound to a value only this
// specific browser session knows — is the only thing that binds the
// authorization code to whoever requested it. A provider that does not
// support S256 gives that up silently, so this must fail loudly at startup
// instead of ever completing a login that has no such binding.
func TestNewFailsInPublicModeWithoutS256Support(t *testing.T) {
	tests := []struct {
		name    string
		methods []string // nil = field omitted entirely
	}{
		{"field omitted", nil},
		{"only plain, no S256", []string{"plain"}},
		{"empty list", []string{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			idp := newPKCEIssuer(t, "test-client", tc.methods)
			cfg := newTestConfig(idp.URL, "")

			_, err := New(context.Background(), cfg, NewStore(), testLogger())
			if err == nil {
				t.Fatal("New should have failed: provider does not advertise S256")
			}
		})
	}
}

// Confidential mode is unaffected by the provider's advertised PKCE support —
// the client secret is already a binding on its own, and this check exists
// specifically because a public client has nothing else. Requiring it there
// too would be a new, unrequested constraint on providers already working.
func TestNewSucceedsInConfidentialModeRegardlessOfS256Support(t *testing.T) {
	idp := newPKCEIssuer(t, "test-client", nil) // no PKCE support advertised
	cfg := newTestConfig(idp.URL, "s3cr3t")

	if _, err := New(context.Background(), cfg, NewStore(), testLogger()); err != nil {
		t.Fatalf("New: %v", err)
	}
}

// loginAndCallback drives the full flow through a real net/http client with a
// cookie jar, exactly as a browser would, and returns the issuer's record of
// what the token exchange sent.
func loginAndCallback(t *testing.T, idp *pkceIssuer, secret string) tokenRequestRecord {
	t.Helper()

	// The redirect URI must be the dashboard test server's own absolute URL,
	// which httptest only assigns once the server starts — so start it with a
	// placeholder handler, then build the real one now that dash.URL is known
	// and swap it in.
	dash := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(dash.Close)

	cfg := newTestConfig(idp.URL, secret)
	cfg.RedirectURI = dash.URL + "/auth/callback"
	authenticator, err := New(context.Background(), cfg, NewStore(), testLogger())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", authenticator.Login)
	mux.HandleFunc("GET /auth/callback", authenticator.Callback)
	// Login() defaults the post-login landing page to /peers when no ?next=
	// is given, and the client below follows redirects automatically, so the
	// chain needs somewhere to land.
	mux.HandleFunc("GET /peers", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	dash.Config.Handler = mux

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	client := &http.Client{Jar: jar}

	res, err := client.Get(dash.URL + "/login")
	if err != nil {
		t.Fatalf("GET /login: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login flow did not complete: status=%d body=%q", res.StatusCode, body)
	}

	idp.mu.Lock()
	defer idp.mu.Unlock()
	if len(idp.tokenRequests) == 0 {
		t.Fatal("token endpoint was never called")
	}
	return idp.tokenRequests[len(idp.tokenRequests)-1]
}

func TestPublicModeSendsNoClientSecretToTokenEndpoint(t *testing.T) {
	idp := newPKCEIssuer(t, "test-client", []string{"S256"})
	got := loginAndCallback(t, idp, "")

	if got.clientSecretParam != "" {
		t.Errorf("client_secret param = %q, want empty", got.clientSecretParam)
	}
	if got.hasBasicAuth {
		t.Error("token request carried HTTP Basic auth, want none for a public client")
	}
	// Confirms the exchange actually used PKCE rather than merely omitting
	// the secret by accident — a verifier must have been sent.
	if got.codeVerifier == "" {
		t.Error("code_verifier was not sent — public mode must still use PKCE")
	}
}

func TestConfidentialModeStillSendsClientSecret(t *testing.T) {
	idp := newPKCEIssuer(t, "test-client", []string{"S256"})
	got := loginAndCallback(t, idp, "s3cr3t")

	if got.clientSecretParam != "s3cr3t" {
		t.Errorf("client_secret param = %q, want s3cr3t — confidential mode must be unchanged", got.clientSecretParam)
	}
}

// The redirect Login() issues is the one artifact a test can inspect without
// a full round trip, and it is where code_challenge actually appears — the
// token exchange itself only ever sends code_verifier.
func TestLoginRedirectCarriesS256Challenge(t *testing.T) {
	for _, secret := range []string{"", "s3cr3t"} {
		t.Run("secret="+secret, func(t *testing.T) {
			idp := newPKCEIssuer(t, "test-client", []string{"S256"})
			cfg := newTestConfig(idp.URL, secret)
			authenticator, err := New(context.Background(), cfg, NewStore(), testLogger())
			if err != nil {
				t.Fatalf("New: %v", err)
			}

			req := httptest.NewRequest(http.MethodGet, "/login", nil)
			rec := httptest.NewRecorder()
			authenticator.Login(rec, req)

			loc, err := req.URL.Parse(rec.Header().Get("Location"))
			if err != nil {
				t.Fatalf("parse Location: %v", err)
			}
			q := loc.Query()
			if q.Get("code_challenge_method") != "S256" {
				t.Errorf("code_challenge_method = %q, want S256", q.Get("code_challenge_method"))
			}
			if q.Get("code_challenge") == "" {
				t.Error("code_challenge is empty")
			}
		})
	}
}

// A quick sanity check that the mode strings are what a log line would
// actually show — this is what makes a misconfiguration visible in the
// journal, per the task's own requirement.
func TestOIDCClientModeStringsAreLogFriendly(t *testing.T) {
	if got := string(config.Public); got != "public" {
		t.Errorf("Public = %q, want \"public\"", got)
	}
	if got := string(config.Confidential); got != "confidential" {
		t.Errorf("Confidential = %q, want \"confidential\"", got)
	}
}
