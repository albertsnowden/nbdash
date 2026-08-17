package api

import (
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
)

// requireSameSiteOrigin is the server-side half of CSRF protection. The session
// cookie's SameSite attribute is a promise made by the browser; this is a check
// made by the server, and unlike SameSite it also refuses a sibling subdomain
// that shares the parent cookie scope.
func TestRequireSameSiteOrigin(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		secFetch   string
		origin     string
		wantStatus int
	}{
		{
			name:       "cross-site POST is rejected",
			method:     http.MethodPost,
			secFetch:   "cross-site",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "same-origin POST is allowed",
			method:     http.MethodPost,
			secFetch:   "same-origin",
			wantStatus: http.StatusOK,
		},
		{
			name:       "same-site POST is allowed",
			method:     http.MethodPost,
			secFetch:   "same-site",
			wantStatus: http.StatusOK,
		},
		{
			// A client old enough to send neither header carries no ambient
			// cookies to abuse, so there is nothing to defend against.
			name:       "both headers absent is allowed",
			method:     http.MethodPost,
			wantStatus: http.StatusOK,
		},
		{
			name:       "Origin host mismatch is rejected",
			method:     http.MethodPost,
			origin:     "https://evil.example",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "matching Origin is allowed",
			method:     http.MethodPost,
			origin:     "http://example.com",
			wantStatus: http.StatusOK,
		},
		{
			// Same host, different port: a different origin, and on a shared
			// host that is a different application.
			name:       "Origin port mismatch is rejected",
			method:     http.MethodPost,
			origin:     "http://example.com:8443",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "malformed Origin is rejected",
			method:     http.MethodPost,
			origin:     "://not a url",
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "DELETE is checked too",
			method:     http.MethodDelete,
			secFetch:   "cross-site",
			wantStatus: http.StatusForbidden,
		},
		{
			// Reads are never blocked: the routes are side-effect free, and
			// blocking them would break ordinary inbound links.
			name:       "GET is never blocked",
			method:     http.MethodGet,
			secFetch:   "cross-site",
			origin:     "https://evil.example",
			wantStatus: http.StatusOK,
		},
		{
			name:       "HEAD is never blocked",
			method:     http.MethodHead,
			secFetch:   "cross-site",
			origin:     "https://evil.example",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reached := false
			handler := requireSameSiteOrigin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				reached = true
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(tc.method, "/api/bff/peers/p1", nil)
			if tc.secFetch != "" {
				req.Header.Set("Sec-Fetch-Site", tc.secFetch)
			}
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if want := tc.wantStatus == http.StatusOK; reached != want {
				t.Errorf("handler reached = %v, want %v", reached, want)
			}
		})
	}
}

// Sec-Fetch-Site is set by the browser and cannot be forged by page script, so
// it wins over an Origin header that disagrees with it.
func TestSecFetchSiteTakesPrecedenceOverOrigin(t *testing.T) {
	handler := requireSameSiteOrigin(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/bff/peers/p1", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	req.Header.Set("Origin", "http://example.com") // matches Host, but is not trusted here

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403 — Sec-Fetch-Site should decide", rec.Code)
	}
}

// The check has to be wired into Routes(), not merely defined.
func TestRoutesRejectCrossSiteMutation(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	routes, err := srv.Routes()
	if err != nil {
		t.Fatalf("Routes: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/bff/groups", nil)
	req.Header.Set("Sec-Fetch-Site", "cross-site")

	rec := httptest.NewRecorder()
	routes.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
	// It must be refused before authentication, so an unauthenticated
	// cross-site POST cannot be answered with a redirect to the login flow.
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Errorf("cross-site POST should be refused outright, got redirect to %q", loc)
	}
}

// HSTS is emitted by this service rather than assumed to come from a reverse
// proxy, and is conditional on the deployment actually being served over TLS.
// See securityHeaders in middleware.go for why the origin owns the header.
func TestStrictTransportSecurity(t *testing.T) {
	tests := []struct {
		name   string
		secure bool
		want   string
	}{
		{"TLS deployment sets HSTS", true, "max-age=31536000"},
		{"plain-http dev does not", false, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := securityHeaders(tc.secure, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/bff/peers", nil))

			if got := rec.Header().Get("Strict-Transport-Security"); got != tc.want {
				t.Errorf("Strict-Transport-Security = %q, want %q", got, tc.want)
			}

			// The unconditional headers must not have been disturbed.
			if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Error("X-Content-Type-Options missing")
			}
			if rec.Header().Get("Content-Security-Policy") == "" {
				t.Error("Content-Security-Policy missing")
			}
		})
	}
}

// includeSubDomains and preload bind hostnames this service does not own, and
// preload is close to irreversible. Neither should appear by accident.
func TestHSTSDoesNotClaimSubdomainsOrPreload(t *testing.T) {
	handler := securityHeaders(true, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/bff/peers", nil))

	hsts := rec.Header().Get("Strict-Transport-Security")
	for _, forbidden := range []string{"includeSubDomains", "preload"} {
		if strings.Contains(hsts, forbidden) {
			t.Errorf("HSTS %q must not contain %q", hsts, forbidden)
		}
	}
}

// limitRequestBody exists so a single oversized POST can't exhaust memory
// before any handler-level validation runs — see its doc comment in
// middleware.go for why that matters more than usual with in-memory sessions.
func TestLimitRequestBody(t *testing.T) {
	handler := limitRequestBody(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.ReadAll(r.Body); err != nil {
			http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("a body under the limit is read in full", func(t *testing.T) {
		body := strings.NewReader(strings.Repeat("a", 1024))
		req := httptest.NewRequest(http.MethodPost, "/api/bff/groups", body)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("a body over the limit fails to read in full", func(t *testing.T) {
		body := strings.NewReader(strings.Repeat("a", maxRequestBodyBytes+1))
		req := httptest.NewRequest(http.MethodPost, "/api/bff/groups", body)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code == http.StatusOK {
			t.Error("an oversized body should not have been read successfully")
		}
	})
}

// The check has to be wired into Routes(), not merely defined — the same
// property TestRoutesRejectCrossSiteMutation pins for requireSameSiteOrigin.
// This drives a real authenticated session through the full middleware chain
// (limitRequestBody sits outside auth, but only manifests once a handler
// actually reads the body, so a real login via /login is the only way to
// reach one) and confirms an oversized create-group request never reaches the
// Management API at all.
func TestRoutesEnforceRequestBodyLimit(t *testing.T) {
	idp := newFakeIdP(t, "test-client")

	var apiCalled bool
	dashURL := startDashboard(t, idp, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			apiCalled = true
		}
		_, _ = io.WriteString(w, `[]`)
	})

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	client := &http.Client{Jar: jar}

	// Establish a real session the same way TestSessionCookieAuthenticatesSubsequentRequests does.
	res, err := client.Get(dashURL + "/login")
	if err != nil {
		t.Fatalf("GET /login: %v", err)
	}
	res.Body.Close()

	body := `{"name":"` + strings.Repeat("a", maxRequestBodyBytes+1) + `"}`
	res, err = client.Post(dashURL+"/api/bff/groups", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/bff/groups: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		t.Errorf("status = %d, want a failure — an oversized body should never reach a successful create", res.StatusCode)
	}
	if apiCalled {
		t.Error("the oversized request reached the Management API; it should have failed reading the body before any handler logic ran")
	}
}
