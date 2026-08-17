package config

import (
	"reflect"
	"strings"
	"testing"
)

// setEnv sets the minimum required variables plus any overrides. t.Setenv
// restores the previous values automatically.
func setEnv(t *testing.T, overrides map[string]string) {
	t.Helper()

	env := map[string]string{
		"NETBIRD_MGMT_API_ENDPOINT": "https://api.netbird.example",
		"AUTH_AUTHORITY":            "https://idp.example",
		"AUTH_CLIENT_ID":            "client-id",
		"AUTH_CLIENT_SECRET":        "client-secret",
		"AUTH_REDIRECT_URI":         "http://localhost:8080/auth/callback",
	}
	for k, v := range overrides {
		env[k] = v
	}
	// Clear anything the test does not set so a developer's shell cannot leak in.
	for _, k := range []string{
		"LISTEN_ADDR", "AUTH_AUDIENCE", "AUTH_SUPPORTED_SCOPES",
		"NETBIRD_TOKEN_SOURCE",
	} {
		if _, ok := env[k]; !ok {
			t.Setenv(k, "")
		}
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
}

func TestLoadDefaults(t *testing.T) {
	setEnv(t, nil)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.ListenAddr != ":8080" {
		t.Errorf("ListenAddr = %q", cfg.ListenAddr)
	}
	if cfg.TokenSource != AccessToken {
		t.Errorf("TokenSource = %q, want accesstoken", cfg.TokenSource)
	}
	want := []string{"openid", "profile", "email", "offline_access", "api"}
	if !reflect.DeepEqual(cfg.AuthScopes, want) {
		t.Errorf("AuthScopes = %v, want %v", cfg.AuthScopes, want)
	}
	// A plain-http redirect URI means the cookie must not be Secure, or the
	// browser would silently drop it in local development.
	if cfg.Secure {
		t.Error("Secure should be false for an http redirect URI")
	}
}

func TestLoadTrimsTrailingSlashes(t *testing.T) {
	setEnv(t, map[string]string{
		"NETBIRD_MGMT_API_ENDPOINT": "https://api.netbird.example/",
		"AUTH_AUTHORITY":            "https://idp.example/",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	// A trailing slash would produce "https://api.netbird.example//api".
	if cfg.APIOrigin != "https://api.netbird.example" {
		t.Errorf("APIOrigin = %q", cfg.APIOrigin)
	}
	if cfg.AuthAuthority != "https://idp.example" {
		t.Errorf("AuthAuthority = %q", cfg.AuthAuthority)
	}
}

// "none" is the documented way to disable the audience parameter for providers
// that reject it.
func TestAudienceNoneIsTreatedAsUnset(t *testing.T) {
	setEnv(t, map[string]string{"AUTH_AUDIENCE": "none"})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AuthAudience != "" {
		t.Errorf("AuthAudience = %q, want empty", cfg.AuthAudience)
	}
}

func TestTokenSourceIsCaseInsensitive(t *testing.T) {
	for _, in := range []string{"idToken", "IDTOKEN", "idtoken"} {
		t.Run(in, func(t *testing.T) {
			setEnv(t, map[string]string{"NETBIRD_TOKEN_SOURCE": in})

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.TokenSource != IDToken {
				t.Errorf("TokenSource = %q, want idtoken", cfg.TokenSource)
			}
		})
	}
}

func TestUnknownTokenSourceFallsBackToAccessToken(t *testing.T) {
	setEnv(t, map[string]string{"NETBIRD_TOKEN_SOURCE": "nonsense"})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TokenSource != AccessToken {
		t.Errorf("TokenSource = %q, want accesstoken", cfg.TokenSource)
	}
}

func TestSecureCookieFollowsRedirectScheme(t *testing.T) {
	tests := []struct {
		redirectURI string
		wantSecure  bool
	}{
		{"https://dash.example/auth/callback", true},
		{"http://localhost:8080/auth/callback", false},
		// Scheme comparison is case-insensitive per RFC 3986, and url.Parse
		// lowercases it, so this must still count as secure.
		{"HTTPS://dash.example/auth/callback", true},
	}

	for _, tc := range tests {
		t.Run(tc.redirectURI, func(t *testing.T) {
			setEnv(t, map[string]string{"AUTH_REDIRECT_URI": tc.redirectURI})

			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if cfg.Secure != tc.wantSecure {
				t.Errorf("Secure = %v, want %v", cfg.Secure, tc.wantSecure)
			}
		})
	}
}

// The Secure flag is derived from this one value, so a value that is not a
// usable URL has to be a startup error. Accepting it would ship the session
// cookie without Secure while everything else appeared to work — the exact
// fail-open this replaced.
func TestMalformedRedirectURIIsRejected(t *testing.T) {
	for _, bad := range []string{
		"htps://dash.example/auth/callback", // typo'd scheme
		"dash.example/auth/callback",        // no scheme
		"/auth/callback",                    // path only
		"ftp://dash.example/auth/callback",  // wrong scheme
		"://not a url",
	} {
		t.Run(bad, func(t *testing.T) {
			setEnv(t, map[string]string{"AUTH_REDIRECT_URI": bad})

			cfg, err := Load()
			if err == nil {
				t.Fatalf("Load accepted %q (Secure=%v); want an error", bad, cfg.Secure)
			}
			if !strings.Contains(err.Error(), "AUTH_REDIRECT_URI") {
				t.Errorf("error %q should name AUTH_REDIRECT_URI", err)
			}
		})
	}
}

// Forgetting AUTH_REDIRECT_URI used to yield a localhost callback and a
// non-Secure cookie, silently. It must now stop startup.
func TestMissingRedirectURIDoesNotSilentlyDisableSecure(t *testing.T) {
	setEnv(t, map[string]string{"AUTH_REDIRECT_URI": ""})

	if _, err := Load(); err == nil {
		t.Fatal("Load should fail when AUTH_REDIRECT_URI is unset")
	}
}

// NetBird's own embedded Dex IdP registers "nbdash" as a public
// client with no secret at all — sending one gets a 401 from Dex's token
// endpoint. An empty AUTH_CLIENT_SECRET has to load cleanly, end to end
// through Load() and not just via a directly-constructed Config, since Load()
// is what a real deployment actually goes through.
func TestEmptyClientSecretSelectsPublicMode(t *testing.T) {
	setEnv(t, map[string]string{"AUTH_CLIENT_SECRET": ""})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OIDCClientMode() != Public {
		t.Errorf("OIDCClientMode() = %q, want %q", cfg.OIDCClientMode(), Public)
	}
}

func TestNonEmptyClientSecretSelectsConfidentialMode(t *testing.T) {
	setEnv(t, map[string]string{"AUTH_CLIENT_SECRET": "a-real-secret"})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.OIDCClientMode() != Confidential {
		t.Errorf("OIDCClientMode() = %q, want %q", cfg.OIDCClientMode(), Confidential)
	}
}

// AUTH_CLIENT_SECRET is deliberately not in this list: an empty value is a
// supported configuration (public OIDC client mode), not a missing one — see
// TestEmptyClientSecretSelectsPublicMode.
func TestMissingRequiredVariablesAreReported(t *testing.T) {
	for _, missing := range []string{
		"NETBIRD_MGMT_API_ENDPOINT",
		"AUTH_AUTHORITY",
		"AUTH_CLIENT_ID",
		"AUTH_REDIRECT_URI",
	} {
		t.Run(missing, func(t *testing.T) {
			setEnv(t, map[string]string{missing: ""})

			_, err := Load()
			if err == nil {
				t.Fatalf("expected an error when %s is unset", missing)
			}
			if !strings.Contains(err.Error(), missing) {
				t.Errorf("error %q should name %s", err, missing)
			}
		})
	}
}
