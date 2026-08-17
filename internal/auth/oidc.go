// Package auth implements the OIDC authorization-code flow and session
// handling for the dashboard.
//
// The Next.js dashboard runs this flow in the browser (@axa-fr/react-oidc) and
// keeps the resulting token in browser storage. Here the flow runs server-side
// with PKCE, and the token stays on the server — in both confidential and
// public client mode. See config.OIDCClientMode for what decides the mode and
// why both are needed.
package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/albertsnowden/nbdash/internal/config"
)

type ctxKey int

const sessionKey ctxKey = iota

// providerTimeout bounds every HTTP call this package makes to the identity
// provider: discovery, the code exchange, token refresh, and JWKS fetches.
//
// It has to be set explicitly. Both golang.org/x/oauth2 and go-oidc resolve
// their HTTP client with oauth2's internal.ContextClient, which falls back to
// http.DefaultClient when the context carries none — and http.DefaultClient
// has no timeout at all. Left at that default, an unresponsive provider parks
// a request goroutine forever; worse, BearerToken refreshes while holding
// sess.mu, so one hung refresh would block every other request for that
// session indefinitely. http.Server's WriteTimeout does not rescue this: it
// fails the response write but never cancels the handler's context.
const providerTimeout = 15 * time.Second

// ErrPublicClientPKCEUnsupported means New was asked to run in public client
// mode (config.Public — no AUTH_CLIENT_SECRET) against a provider whose
// discovery document does not advertise PKCE S256 support. Callers that retry
// New on failure — main.go's startup retry loop exists for a provider that is
// still coming up — should check for this with errors.Is and not retry it:
// the provider's advertised capabilities will not change between attempts, so
// retrying only delays reporting a static misconfiguration.
var ErrPublicClientPKCEUnsupported = errors.New("provider does not advertise PKCE S256 support")

// Authenticator wires the OIDC provider to the session store.
type Authenticator struct {
	cfg      *config.Config
	store    *Store
	provider *oidc.Provider
	verifier *oidc.IDTokenVerifier
	oauth    oauth2.Config
	log      *slog.Logger

	// hc is the timeout-bearing client every provider call runs on. See
	// providerTimeout, and providerContext for how it reaches oauth2/go-oidc.
	hc *http.Client

	// endSessionEndpoint is the provider's RP-Initiated Logout endpoint, from
	// its discovery document. Empty when the provider does not advertise one —
	// Logout then falls back to dropping only the local session. See Logout.
	endSessionEndpoint string

	// postLogoutRedirectURI is where the provider is asked to send the browser
	// back after it ends its own session. Derived from RedirectURI rather than
	// separately configured, the same way Secure is: it must be this
	// dashboard's own origin, and a second knob would only risk drifting from
	// the origin the provider already has registered for the callback.
	postLogoutRedirectURI string
}

func New(ctx context.Context, cfg *config.Config, store *Store, log *slog.Logger) (*Authenticator, error) {
	hc := &http.Client{Timeout: providerTimeout}

	// Injected before discovery so the provider's remote key set — built here
	// and reused for every later ID-token verification — inherits the same
	// bounded client rather than http.DefaultClient.
	provider, err := oidc.NewProvider(oidc.ClientContext(ctx, hc), cfg.AuthAuthority)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery against %s: %w", cfg.AuthAuthority, err)
	}

	// A public client (no secret) relies on PKCE alone to bind the
	// authorization code to whoever requested it — there is no other check at
	// the token endpoint. If the provider doesn't advertise S256 support, it
	// may simply ignore the code_challenge parameter this dashboard sends,
	// silently dropping that binding rather than rejecting the request. A
	// confidential client already has the secret as a binding and is not held
	// to this — the check exists specifically because a public client has
	// nothing else, not as a general PKCE mandate.
	if cfg.OIDCClientMode() == config.Public {
		var claims struct {
			CodeChallengeMethodsSupported []string `json:"code_challenge_methods_supported"`
		}
		if err := provider.Claims(&claims); err != nil {
			return nil, fmt.Errorf("reading discovery document for PKCE support: %w", err)
		}
		if !slices.Contains(claims.CodeChallengeMethodsSupported, "S256") {
			return nil, fmt.Errorf(
				"%w: %s did not advertise it in code_challenge_methods_supported (got %v); "+
					"set AUTH_CLIENT_SECRET to run as a confidential client instead if this "+
					"provider issues one",
				ErrPublicClientPKCEUnsupported, cfg.AuthAuthority, claims.CodeChallengeMethodsSupported)
		}
	}

	endpoint := provider.Endpoint()
	// Explicit rather than AuthStyleAutoDetect (the zero value): auto-detect's
	// first attempt sends credentials via HTTP Basic, and only falls back to
	// this style after a failed round trip. Forcing it removes that probe and
	// keeps both client modes' token requests predictable for logging and
	// testing. In params, with an empty client_secret, oauth2 omits the
	// parameter entirely rather than sending client_secret="" — see
	// golang.org/x/oauth2/internal/token.go's newTokenRequest, which only sets
	// it `if clientSecret != ""` — so a public client sends no client secret
	// at all, in the header or the body.
	endpoint.AuthStyle = oauth2.AuthStyleInParams

	// end_session_endpoint is optional in the discovery document, so a missing
	// or unreadable value is not fatal — Claims just re-unmarshals the
	// discovery bytes already fetched above, no extra round trip — it only
	// means Logout cannot reach the provider and falls back to a local-only
	// sign-out, the behaviour this dashboard already had.
	var discoveryClaims struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	_ = provider.Claims(&discoveryClaims)

	redirectURL, err := url.Parse(cfg.RedirectURI)
	if err != nil {
		// config.Load already validates this; a caller that hand-builds a
		// Config (as tests do) gets the same fail-fast treatment.
		return nil, fmt.Errorf("parsing AUTH_REDIRECT_URI: %w", err)
	}

	return &Authenticator{
		cfg:                cfg,
		store:              store,
		provider:           provider,
		verifier:           provider.Verifier(&oidc.Config{ClientID: cfg.AuthClientID}),
		log:                log,
		hc:                 hc,
		endSessionEndpoint: discoveryClaims.EndSessionEndpoint,
		postLogoutRedirectURI: (&url.URL{
			Scheme: redirectURL.Scheme,
			Host:   redirectURL.Host,
			Path:   "/",
		}).String(),
		oauth: oauth2.Config{
			ClientID:     cfg.AuthClientID,
			ClientSecret: cfg.AuthClientSecret,
			RedirectURL:  cfg.RedirectURI,
			Endpoint:     endpoint,
			Scopes:       cfg.AuthScopes,
		},
	}, nil
}

// providerContext returns a context that carries the bounded HTTP client, for
// the oauth2 calls (Exchange, TokenSource) that resolve their client from the
// context. Cancellation still comes from the caller's request context, so a
// disconnecting client aborts the call as before; this only replaces the
// client that would otherwise be http.DefaultClient.
func (a *Authenticator) providerContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, a.hc)
}

// Mode reports which OIDC client mode this authenticator is running in, so
// main.go can log it at startup — the one place a misconfiguration (e.g. a
// secret set for a provider that will reject it, or omitted for one that
// requires it) becomes visible, in the journal, before a user ever hits it as
// a failed login.
func (a *Authenticator) Mode() config.OIDCClientMode {
	return a.cfg.OIDCClientMode()
}

// Login starts the authorization-code flow. The path the user was trying to
// reach is carried through the flow so they land back on it afterwards —
// the equivalent of the Next.js dashboard's `login(currentPath)`.
func (a *Authenticator) Login(w http.ResponseWriter, r *http.Request) {
	state, err := randomID()
	if err != nil {
		http.Error(w, "failed to start login", http.StatusInternalServerError)
		return
	}
	nonce, err := randomID()
	if err != nil {
		http.Error(w, "failed to start login", http.StatusInternalServerError)
		return
	}
	verifier := oauth2.GenerateVerifier()

	redirect := r.URL.Query().Get("next")
	if !isSafeRedirect(redirect) {
		redirect = "/peers"
	}

	a.store.savePending(state, &pending{
		verifier: verifier,
		nonce:    nonce,
		redirect: redirect,
	})
	setCookie(w, a.stateCookieName(), state, a.cfg.Secure, int(pendingTTL.Seconds()))

	opts := []oauth2.AuthCodeOption{
		oidc.Nonce(nonce),
		oauth2.S256ChallengeOption(verifier),
	}
	if a.cfg.AuthAudience != "" {
		opts = append(opts, oauth2.SetAuthURLParam("audience", a.cfg.AuthAudience))
	}

	http.Redirect(w, r, a.oauth.AuthCodeURL(state, opts...), http.StatusFound)
}

// Callback completes the flow: it validates state against the cookie, exchanges
// the code, verifies the ID token and its nonce, then establishes a session.
func (a *Authenticator) Callback(w http.ResponseWriter, r *http.Request) {
	if errParam := r.URL.Query().Get("error"); errParam != "" {
		// The provider's error and error_description are echoed to the journal,
		// never to the page. Both are attacker-controllable — anyone can send an
		// admin a link to /auth/callback?error=...&error_description=... — so
		// reflecting them would let a crafted link render arbitrary text on the
		// dashboard's own origin, which is a phishing primitive even though
		// http.Error's text/plain plus nosniff stops it being script.
		a.log.Warn("provider returned an error at the callback",
			"error", errParam,
			"error_description", r.URL.Query().Get("error_description"))
		http.Error(w, "login failed at the identity provider, please retry",
			http.StatusUnauthorized)
		return
	}

	// The state must match both the cookie and a live pending entry: the cookie
	// binds the callback to this browser, the entry to this server.
	cookie, err := r.Cookie(a.stateCookieName())
	if err != nil {
		http.Error(w, "login state missing or expired, please retry", http.StatusBadRequest)
		return
	}
	state := r.URL.Query().Get("state")
	if state == "" || state != cookie.Value {
		http.Error(w, "login state mismatch", http.StatusBadRequest)
		return
	}
	p, ok := a.store.takePending(state)
	if !ok {
		http.Error(w, "login state expired, please retry", http.StatusBadRequest)
		return
	}
	a.clearCookie(w, a.stateCookieName())

	token, err := a.oauth.Exchange(a.providerContext(r.Context()), r.URL.Query().Get("code"),
		oauth2.VerifierOption(p.verifier))
	if err != nil {
		a.log.Warn("authorization code exchange failed", "error", err)
		http.Error(w, "failed to exchange authorization code", http.StatusUnauthorized)
		return
	}

	rawID, ok := token.Extra("id_token").(string)
	if !ok {
		a.log.Warn("provider returned no id_token in the token response")
		http.Error(w, "provider did not return an id_token", http.StatusUnauthorized)
		return
	}
	idToken, err := a.verifier.Verify(r.Context(), rawID)
	if err != nil {
		a.log.Warn("id_token verification failed", "error", err)
		http.Error(w, "failed to verify id_token", http.StatusUnauthorized)
		return
	}
	// A nonce mismatch means this ID token was not minted for the authorization
	// request this browser started — the signature is valid, so it is a replay
	// of a token from elsewhere rather than a forgery. Worth its own log line.
	if idToken.Nonce != p.nonce {
		a.log.Warn("id_token nonce mismatch, possible replay", "sub", idToken.Subject)
		http.Error(w, "id_token nonce mismatch", http.StatusUnauthorized)
		return
	}

	var claims struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	// Claims are cosmetic here (header display), so a decode failure is not
	// fatal to the session.
	_ = idToken.Claims(&claims)

	id, err := randomID()
	if err != nil {
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	a.store.save(&Session{
		ID:      id,
		Token:   token,
		IDToken: rawID,
		Email:   claims.Email,
		Name:    claims.Name,
		Sub:     idToken.Subject,
	})
	// Derived from sessionLifetime so the two cannot drift. The cookie is only
	// a hint to a cooperating browser — the store is what actually enforces it.
	setCookie(w, a.cookieName(), id, a.cfg.Secure, int(sessionLifetime.Seconds()))

	http.Redirect(w, r, p.redirect, http.StatusFound)
}

// Logout drops the local session and, when the provider advertises
// end_session_endpoint, also ends the session at the provider (RP-Initiated
// Logout).
//
// The second half matters: a local-only logout clears this dashboard's own
// cookie, but the provider's own SSO session survives it. The next protected
// page load then bounces through /login out to the provider, which — still
// holding its own session — silently re-authenticates with no prompt at all,
// and the user lands right back in signed in. To the user that looks
// indistinguishable from sign-out doing nothing.
//
// It is routed as POST, not GET. Ending a session is a state change, and
// requireSameSiteOrigin deliberately exempts GET and HEAD (reads are
// side-effect free and blocking them would break inbound links) — so as a GET
// this was the one mutation in the app reachable cross-site, lettings any page
// sign an admin out with <img src="https://dashboard/logout">. As a POST it
// goes through the same origin check as every other mutation.
func (a *Authenticator) Logout(w http.ResponseWriter, r *http.Request) {
	var idToken string
	if c, err := r.Cookie(a.cookieName()); err == nil {
		// Read before delete: the id_token_hint below is what lets the
		// provider end the exact session this browser holds, rather than
		// (depending on the provider) prompting the user to pick one or
		// rejecting the request outright for lacking it.
		if sess, ok := a.store.get(c.Value); ok {
			idToken = sess.IDToken
		}
		a.store.delete(c.Value)
	}
	a.clearCookie(w, a.cookieName())
	// 303, not 302: the browser must follow this with a GET. A 302 leaves the
	// method conversion technically up to the client.
	http.Redirect(w, r, a.logoutRedirectTarget(idToken), http.StatusSeeOther)
}

// logoutRedirectTarget is where Logout sends the browser after dropping the
// local session. With no end_session_endpoint advertised, that is just this
// dashboard's own front page, same as before RP-Initiated Logout support
// existed. With one, it is that endpoint, which ends the provider's session
// and then returns the browser to postLogoutRedirectURI itself.
func (a *Authenticator) logoutRedirectTarget(idToken string) string {
	if a.endSessionEndpoint == "" {
		return "/"
	}

	u, err := url.Parse(a.endSessionEndpoint)
	if err != nil {
		// Malformed provider metadata. Fail open to local-only logout rather
		// than send the browser to a broken URL — the session is already
		// dropped locally by the time this runs.
		return "/"
	}
	q := u.Query()
	q.Set("client_id", a.cfg.AuthClientID)
	q.Set("post_logout_redirect_uri", a.postLogoutRedirectURI)
	if idToken != "" {
		q.Set("id_token_hint", idToken)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// Authenticate resolves the session for r's cookie, if any, and reports
// whether one was found. It does the cookie/store lookup only — every
// caller owns its own response for the "no session" case. internal/api's
// Server.Middleware is the sole caller: it wraps every /api/bff/... route,
// attaches the resolved session to the request context on success (see
// NewContext), and answers a missing one with a JSON 401 body rather than
// an HTML redirect, since /api/bff/... only ever serves fetch() calls from
// the SPA, which can't act on a redirect the way a browser navigation can.
func (a *Authenticator) Authenticate(r *http.Request) (context.Context, bool) {
	c, err := r.Cookie(a.cookieName())
	if err != nil {
		return r.Context(), false
	}
	sess, ok := a.store.get(c.Value)
	if !ok {
		return r.Context(), false
	}
	return NewContext(r.Context(), sess), true
}

// FromContext returns the session Authenticate attached to ctx (see
// NewContext).
func FromContext(ctx context.Context) (*Session, bool) {
	sess, ok := ctx.Value(sessionKey).(*Session)
	return sess, ok
}

// NewContext attaches a session to a context. Authenticate uses it on the
// request path; it is exported so handlers can be exercised without driving
// a full authorization-code flow.
func NewContext(ctx context.Context, sess *Session) context.Context {
	return context.WithValue(ctx, sessionKey, sess)
}

// BearerToken returns the credential to present to the Management API,
// refreshing it first if it has expired.
//
// Which token is sent is a deployment choice (NETBIRD_TOKEN_SOURCE): Auth0
// issues an API-audience access token, while some providers are configured so
// the ID token is what Management validates.
func (a *Authenticator) BearerToken(ctx context.Context, sess *Session) (string, error) {
	// The lock spans the whole read-modify-write, including the network call,
	// rather than just the assignment. The decision to refresh is made by
	// reading sess.Token, so releasing earlier would let every concurrent
	// caller independently decide to refresh. Providers that rotate refresh
	// tokens invalidate the previous one on use, so the second refresh would
	// fail and log the user out mid-session.
	//
	// Serialising here costs nothing in the common case: callers that arrive
	// after the refresh find a valid token and return without touching the
	// network, and the lock is per-session, so users never contend.
	sess.mu.Lock()
	defer sess.mu.Unlock()

	token, err := a.oauth.TokenSource(a.providerContext(ctx), sess.Token).Token()
	if err != nil {
		return "", fmt.Errorf("refresh token: %w", err)
	}

	if token.AccessToken != sess.Token.AccessToken {
		sess.Token = token
		if raw, ok := token.Extra("id_token").(string); ok {
			sess.IDToken = raw
		}
		a.store.save(sess)
	}

	if a.cfg.TokenSource == config.IDToken {
		if sess.IDToken == "" {
			return "", errors.New("no id_token available for this session")
		}
		return sess.IDToken, nil
	}
	return token.AccessToken, nil
}

// ErrNoSession means ctx carries no authenticated session (see NewContext) —
// there was no cookie, or Middleware never ran. Callers should treat it
// exactly like an expired or rejected token: send the user back through
// login.
var ErrNoSession = errors.New("no session in context")

// RequestToken resolves the bearer credential for whatever session is
// attached to ctx, refreshing it if needed. It is the presentation-agnostic
// core both internal/handlers' HTML responses and internal/api's JSON
// responses need — each owns its own failure response (an HTML redirect vs
// a JSON 401 body), but neither should reimplement session lookup or token
// refresh, which is why this exists here rather than being duplicated in
// both packages.
func (a *Authenticator) RequestToken(ctx context.Context) (string, error) {
	sess, ok := FromContext(ctx)
	if !ok {
		return "", ErrNoSession
	}
	return a.BearerToken(ctx, sess)
}

// isSafeRedirect allows only same-site absolute paths, so a crafted ?next=
// cannot bounce the user to another origin after login. Getting this wrong is
// a phishing vector against an admin of a VPN control plane, so the input is
// parsed rather than prefix-matched: a leading "/" says nothing about what the
// browser will actually resolve.
func isSafeRedirect(p string) bool {
	if !strings.HasPrefix(p, "/") || strings.HasPrefix(p, "//") {
		return false
	}

	u, err := url.Parse(p)
	if err != nil {
		return false
	}

	// Any scheme or authority means the destination is not this origin. Opaque
	// covers "scheme:opaque" forms and User covers "//user@host" credentials.
	if u.Scheme != "" || u.Opaque != "" || u.Host != "" || u.User != nil {
		return false
	}

	// Browsers normalise "\" to "/" and strip control characters inside the
	// authority before parsing (WHATWG URL), so "/\evil.example" is fetched as
	// //evil.example and leaves the origin even though url.Parse reads it as an
	// ordinary path. CR and LF would additionally split the Location header.
	// Rejecting these bytes outright is safer than reproducing the browser's
	// normalisation here; u.Path is scanned as well as the raw input because
	// %5C only becomes a backslash once decoded.
	for _, s := range [2]string{p, u.Path} {
		for i := 0; i < len(s); i++ {
			if c := s[i]; c == '\\' || c < 0x20 || c == 0x7f {
				return false
			}
		}
	}

	return true
}
