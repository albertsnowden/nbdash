// Package config loads the dashboard's runtime configuration from the
// environment.
//
// The variable names deliberately mirror the ones the Next.js dashboard
// consumes (see dashboard/config.json and docker/init_react_envs.sh) so an
// existing deployment's environment carries over unchanged.
//
// AUTH_CLIENT_SECRET is optional, not required. The initial version of this
// dashboard made it mandatory on the assumption that moving the OIDC exchange
// server-side meant the client could always be confidential — true for
// external providers like Auth0 or Okta, false for NetBird's own embedded Dex
// IdP, which registers "nbdash" as a PUBLIC client with no secret
// at all (see idp/dex/provider.go and management/server/idp/embedded.go in
// the netbird repo: both set Public: true and never assign Secret). Sending
// any secret to that server gets a flat 401 from Dex's token endpoint, so the
// dashboard could not complete a login against the very server it is meant to
// run against. See OIDCClientMode.
package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// TokenSource selects which OIDC token is forwarded to the Management API.
// Mirrors the NETBIRD_TOKEN_SOURCE setting.
type TokenSource string

const (
	AccessToken TokenSource = "accesstoken"
	IDToken     TokenSource = "idtoken"
)

type Config struct {
	// ListenAddr is the address the dashboard's HTTP server binds to.
	ListenAddr string

	// APIOrigin is the base URL of the NetBird Management service. The "/api"
	// prefix is appended by the client, matching the Next.js dashboard's
	// `config.apiOrigin + "/api"`.
	APIOrigin string

	// AuthAuthority is the OIDC issuer URL used for discovery.
	AuthAuthority string
	AuthClientID  string
	// AuthClientSecret is optional. Empty means the dashboard authenticates as
	// a public OIDC client (PKCE only, no client authentication at the token
	// endpoint) rather than a confidential one. See OIDCClientMode.
	AuthClientSecret string
	// AuthAudience is sent as an extra authorization parameter. Auth0 requires
	// it to issue a JWT for the Management API rather than an opaque token.
	AuthAudience string
	AuthScopes   []string
	RedirectURI  string

	// TokenSource decides whether the ID token or the access token is used as
	// the bearer credential against the Management API.
	TokenSource TokenSource

	// Secure marks the session cookie Secure. Derived from RedirectURI's scheme.
	Secure bool
}

// Load reads the configuration from the environment, applying the same
// defaults the Next.js dashboard applies and failing fast on anything missing.
func Load() (*Config, error) {
	c := &Config{
		ListenAddr:       envOr("LISTEN_ADDR", ":8080"),
		APIOrigin:        strings.TrimSuffix(os.Getenv("NETBIRD_MGMT_API_ENDPOINT"), "/"),
		AuthAuthority:    strings.TrimSuffix(os.Getenv("AUTH_AUTHORITY"), "/"),
		AuthClientID:     os.Getenv("AUTH_CLIENT_ID"),
		AuthClientSecret: os.Getenv("AUTH_CLIENT_SECRET"),
		AuthAudience:     os.Getenv("AUTH_AUDIENCE"),
		RedirectURI:      os.Getenv("AUTH_REDIRECT_URI"),
	}

	// "none" is the documented way to disable the audience parameter for
	// providers that reject it (see init_react_envs.sh).
	if c.AuthAudience == "none" {
		c.AuthAudience = ""
	}

	scopes := envOr("AUTH_SUPPORTED_SCOPES", "openid profile email offline_access api")
	c.AuthScopes = strings.Fields(scopes)

	switch TokenSource(strings.ToLower(envOr("NETBIRD_TOKEN_SOURCE", "accessToken"))) {
	case IDToken:
		c.TokenSource = IDToken
	default:
		c.TokenSource = AccessToken
	}

	for _, f := range []struct{ name, value string }{
		{"NETBIRD_MGMT_API_ENDPOINT", c.APIOrigin},
		{"AUTH_AUTHORITY", c.AuthAuthority},
		{"AUTH_CLIENT_ID", c.AuthClientID},
		// AUTH_CLIENT_SECRET is deliberately absent from this list — see the
		// package doc comment and OIDCClientMode. An empty value is a normal,
		// supported configuration, not a missing one.
		//
		// AUTH_REDIRECT_URI is required rather than defaulted. It used to fall
		// back to http://localhost:8080/auth/callback, which fails open twice
		// over: a deployment that forgot it got a callback URL pointing at
		// localhost *and*, because Secure is derived from the scheme below, a
		// session cookie with no Secure flag — and nothing looked wrong until
		// the cookie was already travelling in clear text. No real deployment
		// can use a default here anyway, since the value must match what is
		// registered with the identity provider.
		{"AUTH_REDIRECT_URI", c.RedirectURI},
	} {
		if f.value == "" {
			return nil, fmt.Errorf("%s environment variable must be set", f.name)
		}
	}

	// Secure is derived from the redirect URI rather than configured on its own
	// so the cookie flag cannot drift from how the dashboard is actually
	// reached. Parsing, rather than a "https://" prefix test, is what stops a
	// typo like "htps://dash.example" from quietly clearing the flag.
	redirect, err := url.Parse(c.RedirectURI)
	if err != nil || (redirect.Scheme != "http" && redirect.Scheme != "https") {
		return nil, fmt.Errorf(
			"AUTH_REDIRECT_URI must be an absolute http:// or https:// URL, got %q", c.RedirectURI)
	}
	c.Secure = redirect.Scheme == "https"

	return c, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// OIDCClientMode is whether the dashboard authenticates as a confidential or a
// public OIDC client. It exists as a named type, rather than callers just
// checking AuthClientSecret == "" wherever the distinction matters, so
// there is exactly one place that decides it — auth.New's PKCE-support check
// and the startup log line both call OIDCClientMode rather than each
// re-deriving it, which would risk the two disagreeing after an edit to one
// but not the other.
type OIDCClientMode string

const (
	// Confidential clients authenticate at the token endpoint with a secret.
	// This is the right choice for external providers (Auth0, Okta, ...) that
	// issue one, and was the dashboard's only supported mode originally.
	Confidential OIDCClientMode = "confidential"

	// Public clients have no secret and rely on PKCE alone to bind the
	// authorization code to the party that requested it. This is what
	// NetBird's own embedded Dex IdP requires — see the package doc comment.
	Public OIDCClientMode = "public"
)

// OIDCClientMode reports which mode this configuration selects. The decision
// is made by exactly one thing: whether AUTH_CLIENT_SECRET was set.
func (c *Config) OIDCClientMode() OIDCClientMode {
	if c.AuthClientSecret == "" {
		return Public
	}
	return Confidential
}
