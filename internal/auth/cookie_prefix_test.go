package auth

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/albertsnowden/nbdash/internal/config"
)

// TestHostPrefixTracksSecure pins that both auth cookies carry the __Host-
// prefix on an https deployment and neither carries it on plain http.
//
// The prefix is enforced by the browser, not here: it refuses to store the
// cookie unless it is Secure, Path=/ and has no Domain. The no-Domain half is
// what matters — without it a sibling host under a shared parent domain can
// set a cookie the browser then sends to the dashboard, overwriting the
// session. SameSite does not help there, because it treats siblings as
// same-site.
//
// The plain-http half is not a nicety: a browser silently DROPS a __Host-
// cookie that is not Secure, so applying the prefix unconditionally would
// break local development entirely rather than fail loudly.
func TestHostPrefixTracksSecure(t *testing.T) {
	tests := []struct {
		name       string
		secure     bool
		wantPrefix bool
	}{
		{"https deployment gets the prefix", true, true},
		{"plain-http dev does not", false, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := &Authenticator{cfg: &config.Config{Secure: tc.secure}}

			for label, got := range map[string]string{
				"session": a.cookieName(),
				"state":   a.stateCookieName(),
			} {
				if has := strings.HasPrefix(got, hostPrefix); has != tc.wantPrefix {
					t.Errorf("%s cookie name = %q, __Host- prefix present = %v, want %v",
						label, got, has, tc.wantPrefix)
				}
			}
		})
	}
}

// TestSetCookieSetsNoDomain guards the attribute the __Host- prefix actually
// depends on. A Domain attribute added to setCookie would not fail any other
// test in this package — the cookie would simply stop being stored by real
// browsers on https deployments, which no unit test here would notice.
func TestSetCookieSetsNoDomain(t *testing.T) {
	rec := httptest.NewRecorder()
	setCookie(rec, hostPrefix+sessionCookie, "value", true, 3600)

	header := rec.Header().Get("Set-Cookie")
	if header == "" {
		t.Fatal("setCookie wrote no Set-Cookie header")
	}
	if strings.Contains(strings.ToLower(header), "domain=") {
		t.Errorf("Set-Cookie carries a Domain attribute, which voids the __Host- prefix: %s", header)
	}
	for _, want := range []string{"Path=/", "Secure", "HttpOnly"} {
		if !strings.Contains(header, want) {
			t.Errorf("Set-Cookie missing %q, required by the __Host- prefix: %s", want, header)
		}
	}
}
