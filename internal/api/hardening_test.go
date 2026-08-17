package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// routesFor builds the fully-wrapped handler (body limit, security headers,
// origin check, auth middleware) rather than calling a handler method
// directly — every property asserted in this file lives in that middleware
// chain, not in an individual handler.
func routesFor(t *testing.T) http.Handler {
	t.Helper()

	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})
	routes, err := srv.Routes()
	if err != nil {
		t.Fatalf("Routes: %v", err)
	}
	return routes
}

// TestLogoutRejectsGET pins logout as a state-changing POST.
//
// As a GET it was the one mutation in the app reachable cross-site:
// requireSameSiteOrigin deliberately waves GET and HEAD through, so any page
// anywhere could sign an admin out with <img src="https://dashboard/logout">.
func TestLogoutRejectsGET(t *testing.T) {
	routes := routesFor(t)

	rec := httptest.NewRecorder()
	routes.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/logout", nil))

	// The route is registered for POST only, so a GET falls through to the
	// catch-all, which serves the public SPA shell. What must NOT happen is
	// the session being dropped.
	if rec.Code == http.StatusSeeOther {
		t.Errorf("GET /logout performed a logout (status %d); it must be POST-only", rec.Code)
	}
	for _, c := range rec.Result().Cookies() {
		if strings.Contains(c.Name, "nb_session") && c.MaxAge < 0 {
			t.Error("GET /logout expired the session cookie; it must be POST-only")
		}
	}
}

// TestLogoutRejectsCrossSitePOST confirms that routing logout as POST actually
// buys the origin check — the whole point of the change.
func TestLogoutRejectsCrossSitePOST(t *testing.T) {
	routes := routesFor(t)

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	rec := httptest.NewRecorder()
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("cross-site POST /logout = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

// TestStaticDirectoryListingIsNotServed pins the noDirFS wrapper in
// internal/web. Nothing under /static/ is secret, but http.FileServer's index
// listing hands an unauthenticated scanner a free inventory of the origin's
// assets, and there is no reason to answer.
func TestStaticDirectoryListingIsNotServed(t *testing.T) {
	routes := routesFor(t)

	rec := httptest.NewRecorder()
	routes.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/static/", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /static/ = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if body := rec.Body.String(); strings.Contains(body, "favicon.ico") {
		t.Errorf("GET /static/ listed directory contents:\n%s", body)
	}
}

// TestStaticFilesStillServed is the counterweight to the test above: the
// no-listings wrapper must not have broken ordinary asset serving.
func TestStaticFilesStillServed(t *testing.T) {
	routes := routesFor(t)

	for _, asset := range []string{"/static/favicon.ico", "/static/netbird-full.svg", "/static/apple-touch-icon.png"} {
		rec := httptest.NewRecorder()
		routes.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, asset, nil))

		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d, want %d", asset, rec.Code, http.StatusOK)
		}
		if rec.Body.Len() == 0 {
			t.Errorf("GET %s served an empty body", asset)
		}
	}
}

// TestCallbackDoesNotReflectProviderError pins that the provider's error and
// error_description are never echoed into the response.
//
// Both are fully attacker-controlled: anyone can send an admin a link to
// /auth/callback?error=...&error_description=.... Reflecting them renders
// arbitrary attacker text on the dashboard's own origin, which is a phishing
// primitive even though http.Error's text/plain plus nosniff stops it being
// executable script.
func TestCallbackDoesNotReflectProviderError(t *testing.T) {
	routes := routesFor(t)

	const marker = "ATTACKER-CONTROLLED-TEXT"
	target := "/auth/callback?error=access_denied&error_description=" + marker

	rec := httptest.NewRecorder()
	routes.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("callback with provider error = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if body := rec.Body.String(); strings.Contains(body, marker) {
		t.Errorf("callback reflected the provider's error_description into the response:\n%s", body)
	}
}

// TestBFFRoutesRejectUnauthenticatedRequestsWithJSONNot302 pins the bug M1
// found the hard way: /api/bff/... only ever serves fetch() calls, and
// fetch() follows a redirect silently — a 302 to /login would get read back
// as an opaque response and choke the caller trying to JSON-parse it. The
// frontend's apiFetch() wrapper (ui/src/api/client.ts) is what turns a 401
// into an actual navigation to /login; the server's only job is to answer
// with a plain JSON body, never a Location header.
func TestBFFRoutesRejectUnauthenticatedRequestsWithJSONNot302(t *testing.T) {
	routes := routesFor(t)

	rec := httptest.NewRecorder()
	routes.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/bff/peers", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Errorf("unauthenticated BFF request got a redirect to %q; fetch() would follow it silently instead of surfacing a 401", loc)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var env errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v (body=%s)", err, rec.Body.String())
	}
	if env.Error.Code != http.StatusUnauthorized {
		t.Errorf("error.code = %d, want 401", env.Error.Code)
	}
}
