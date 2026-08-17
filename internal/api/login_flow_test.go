package api

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/albertsnowden/nbdash/internal/auth"
	"github.com/albertsnowden/nbdash/internal/config"
	"github.com/albertsnowden/nbdash/internal/nbapi"
)

// fakeIdP is an OIDC provider complete enough for the real authorization-code
// flow: it signs id_tokens with RS256 and publishes the matching JWKS, so
// go-oidc verifies them the way it would in production.
type fakeIdP struct {
	URL      string
	clientID string
	key      *rsa.PrivateKey

	mu     sync.Mutex
	nonces map[string]string // authorization code -> nonce
}

func newFakeIdP(t *testing.T, clientID string) *fakeIdP {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	idp := &fakeIdP{clientID: clientID, key: key, nonces: map[string]string{}}

	mux := http.NewServeMux()

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeFakeIdPJSON(w, map[string]any{
			"issuer":                                idp.URL,
			"authorization_endpoint":                idp.URL + "/authorize",
			"token_endpoint":                        idp.URL + "/oauth/token",
			"jwks_uri":                              idp.URL + "/jwks",
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		writeFakeIdPJSON(w, map[string]any{"keys": []any{map[string]any{
			"kty": "RSA",
			"kid": "test",
			"use": "sig",
			"alg": "RS256",
			"n":   base64.RawURLEncoding.EncodeToString(key.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes()),
		}}})
	})

	// Stand-in for the consent screen: immediately bounce back with a code,
	// which is what a provider does when the user already has a session.
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		code := "code-" + q.Get("state")

		idp.mu.Lock()
		idp.nonces[code] = q.Get("nonce")
		idp.mu.Unlock()

		http.Redirect(w, r, q.Get("redirect_uri")+"?code="+code+"&state="+q.Get("state"), http.StatusFound)
	})

	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()

		idp.mu.Lock()
		nonce := idp.nonces[r.PostFormValue("code")]
		idp.mu.Unlock()

		now := time.Now()
		writeFakeIdPJSON(w, map[string]any{
			"access_token":  "access-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
			"refresh_token": "refresh-token",
			"id_token": idp.signJWT(t, map[string]any{
				"iss":   idp.URL,
				"aud":   clientID,
				"sub":   "user-123",
				"email": "ada@example.com",
				"name":  "Ada Lovelace",
				"nonce": nonce,
				"iat":   now.Unix(),
				"exp":   now.Add(time.Hour).Unix(),
			}),
		})
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	idp.URL = srv.URL
	return idp
}

func (i *fakeIdP) signJWT(t *testing.T, claims map[string]any) string {
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

func writeFakeIdPJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// startDashboard runs the real Routes() handler on a real listener, which the
// flow needs because the redirect URI must be an absolute URL.
func startDashboard(t *testing.T, idp *fakeIdP, apiHandler http.HandlerFunc) string {
	t.Helper()

	apiSrv := httptest.NewServer(apiHandler)
	t.Cleanup(apiSrv.Close)

	// The handler cannot be built until the dashboard's own URL is known, so it
	// is swapped in after the listener is up.
	var handler atomic.Value
	dash := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.Load().(http.Handler).ServeHTTP(w, r)
	}))
	t.Cleanup(dash.Close)

	cfg := &config.Config{
		APIOrigin:        apiSrv.URL,
		AuthAuthority:    idp.URL,
		AuthClientID:     idp.clientID,
		AuthClientSecret: "secret",
		RedirectURI:      dash.URL + "/auth/callback",
		AuthScopes:       []string{"openid", "profile", "email", "offline_access"},
		TokenSource:      config.AccessToken,
	}

	authenticator, err := auth.New(t.Context(), cfg, auth.NewStore(), slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatalf("auth.New: %v", err)
	}

	srv := New(cfg, authenticator, nbapi.New(apiSrv.URL), slog.New(slog.DiscardHandler))
	routes, err := srv.Routes()
	if err != nil {
		t.Fatalf("Routes: %v", err)
	}
	handler.Store(routes)

	return dash.URL
}

// TestLoginFlowEndToEnd drives the whole flow: /login bounces out to the
// provider, back through /auth/callback, and lands on the default
// post-login path (see Authenticator.Login's redirect default), with a
// session established that authorizes the SPA's own BFF calls.
//
// NOTE ON SAMESITE: this test cannot tell SameSite=Lax from SameSite=Strict.
// net/http/cookiejar records the attribute but never consults it when choosing
// which cookies to send (jar.go stores entry.SameSite and no read path reads
// it), and it models no notion of a request initiator or a redirect chain. Only
// a real browser enforces SameSite. See TestAuthCookiesUseSameSiteLax.
func TestLoginFlowEndToEnd(t *testing.T) {
	idp := newFakeIdP(t, "test-client")
	dashURL := startDashboard(t, idp, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[{"id":"p1","name":"gateway-01","ip":"100.92.0.1","connected":true}]`)
	})

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}

	var trail []string
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			trail = append(trail, r.URL.String())
			if len(via) > 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	res, err := client.Get(dashURL + "/login")
	if err != nil {
		t.Fatalf("GET /login: %v", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200\ntrail: %v\nbody: %s", res.StatusCode, trail, body)
	}
	if got := res.Request.URL.Path; got != "/peers" {
		t.Errorf("landed on %q, want /peers, the default post-login redirect (trail: %v)", got, trail)
	}

	// The flow really went out to the provider and back.
	var viaIdP bool
	for _, u := range trail {
		if strings.HasPrefix(u, idp.URL) {
			viaIdP = true
		}
	}
	if !viaIdP {
		t.Errorf("flow never reached the provider: %v", trail)
	}

	// The session the flow established authorizes the SPA's own BFF calls —
	// this is the thing that actually matters post-cutover, since the landing
	// page itself is just the static SPA shell.
	apiRes, err := client.Get(dashURL + "/api/bff/peers")
	if err != nil {
		t.Fatalf("GET /api/bff/peers: %v", err)
	}
	defer apiRes.Body.Close()

	apiBody, err := io.ReadAll(apiRes.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if apiRes.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/bff/peers status = %d, want 200: %s", apiRes.StatusCode, apiBody)
	}
	if !strings.Contains(string(apiBody), "gateway-01") {
		t.Errorf("response missing the stub peer — landed unauthenticated? body=%s", apiBody)
	}
}

// A second request reuses the established session rather than re-running the
// flow, which is what makes the session cookie's SameSite mode load-bearing.
func TestSessionCookieAuthenticatesSubsequentRequests(t *testing.T) {
	idp := newFakeIdP(t, "test-client")
	dashURL := startDashboard(t, idp, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[{"id":"p1","name":"gateway-01","connected":true}]`)
	})

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	res, err := client.Get(dashURL + "/login")
	if err != nil {
		t.Fatalf("GET /login: %v", err)
	}
	res.Body.Close()

	res, err = client.Get(dashURL + "/api/bff/peers")
	if err != nil {
		t.Fatalf("first GET /api/bff/peers: %v", err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("first GET /api/bff/peers status = %d, want 200", res.StatusCode)
	}

	res, err = client.Get(dashURL + "/api/bff/peers")
	if err != nil {
		t.Fatalf("second GET /api/bff/peers: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("second request status = %d, want 200 — the session cookie was not honoured", res.StatusCode)
	}
}

// TestAuthCookiesUseSameSiteLax pins a decision that no Go test can verify, so
// it records the reasoning instead.
//
// SameSite=Strict was tried and reverted. Strict's defining behaviour — the
// property that separates it from Lax — is that the cookie is withheld on
// cross-site top-level navigations. The provider's redirect to /auth/callback
// is exactly that: a top-level GET navigation initiated by the provider's
// origin. Under Strict the browser would not send nb_auth_state, Callback's
// r.Cookie(stateCookie) would fail, and every login would end at "login state
// missing or expired". setCookie is shared by both cookies, so the state cookie
// cannot avoid whatever the session cookie gets.
//
// Restricting Strict to the session cookie alone still does not work reliably:
// the cookie is set on the callback response and first needed on the immediate
// redirect to the landing page, and whether a browser sends a Strict cookie
// there depends on if it considers the whole redirect chain cross-site. That
// varies between engines and, in Chrome, sits behind a non-default flag. A
// login that works in one browser and loops in another is worse than no change.
//
// requireSameSiteOrigin in middleware.go is what actually carries CSRF
// protection here; it is a server-side check and does not depend on any of
// the above.
func TestAuthCookiesUseSameSiteLax(t *testing.T) {
	idp := newFakeIdP(t, "test-client")
	dashURL := startDashboard(t, idp, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[]`)
	})

	jar, _ := cookiejar.New(nil)
	recorder := &cookieRecorder{base: http.DefaultTransport, seen: map[string]http.SameSite{}}
	client := &http.Client{Jar: jar, Transport: recorder}

	res, err := client.Get(dashURL + "/login")
	if err != nil {
		t.Fatalf("GET /login: %v", err)
	}
	res.Body.Close()

	for _, name := range []string{"nb_auth_state", "nb_session"} {
		mode, ok := recorder.sameSite(name)
		if !ok {
			t.Fatalf("%s was never set during the login flow", name)
		}
		if mode != http.SameSiteLaxMode {
			t.Errorf("%s SameSite = %v, want Lax — see this test's comment", name, mode)
		}
	}
}

// cookieRecorder captures the SameSite attribute of every cookie set anywhere
// in a redirect chain. A CheckRedirect hook cannot do this: Request.Response is
// nil for the first hop.
type cookieRecorder struct {
	base http.RoundTripper

	mu   sync.Mutex
	seen map[string]http.SameSite
}

func (c *cookieRecorder) RoundTrip(r *http.Request) (*http.Response, error) {
	res, err := c.base.RoundTrip(r)
	if err != nil {
		return res, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, ck := range res.Cookies() {
		// Skip deletions: Callback clears the state cookie once it is consumed.
		if ck.Value != "" {
			c.seen[ck.Name] = ck.SameSite
		}
	}
	return res, nil
}

func (c *cookieRecorder) sameSite(name string) (http.SameSite, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	mode, ok := c.seen[name]
	return mode, ok
}

// stripStateCookieOnCallback withholds nb_auth_state on the callback hop, which
// is exactly what a browser does when that cookie is SameSite=Strict: the
// provider's redirect to /auth/callback is a cross-site top-level navigation,
// the one case Lax permits and Strict refuses.
type stripStateCookieOnCallback struct{ base http.RoundTripper }

func (s stripStateCookieOnCallback) RoundTrip(r *http.Request) (*http.Response, error) {
	if strings.HasSuffix(r.URL.Path, "/auth/callback") {
		var kept []string
		for _, c := range r.Cookies() {
			if c.Name != "nb_auth_state" {
				kept = append(kept, c.Name+"="+c.Value)
			}
		}
		if len(kept) == 0 {
			r.Header.Del("Cookie")
		} else {
			r.Header.Set("Cookie", strings.Join(kept, "; "))
		}
	}
	return s.base.RoundTrip(r)
}

// TestLoginFailsWhenStateCookieIsWithheldAtCallback shows the concrete
// consequence of SameSite=Strict on the state cookie, without pretending to
// emulate a browser: it asserts what our own Callback does when that cookie
// does not arrive. Strict guarantees it will not arrive, so this is the login
// failure every user would hit. See TestAuthCookiesUseSameSiteLax.
func TestLoginFailsWhenStateCookieIsWithheldAtCallback(t *testing.T) {
	idp := newFakeIdP(t, "test-client")
	dashURL := startDashboard(t, idp, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[]`)
	})

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar:       jar,
		Transport: stripStateCookieOnCallback{base: http.DefaultTransport},
	}

	res, err := client.Get(dashURL + "/login")
	if err != nil {
		t.Fatalf("GET /login: %v", err)
	}
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; login should fail without the state cookie", res.StatusCode)
	}
	if !strings.Contains(string(body), "login state missing") {
		t.Errorf("body = %q, want the state-cookie failure", strings.TrimSpace(string(body)))
	}
}
