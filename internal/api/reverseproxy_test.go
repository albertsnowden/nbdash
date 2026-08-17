package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const oneService = `{
	"id":"svc1","name":"myapp","domain":"myapp.example.netbird.app","mode":"http","enabled":true,
	"targets":[{"target_id":"t1","target_type":"host","protocol":"http","host":"10.0.0.5","port":8080,"enabled":true}],
	"auth":{"password_auth":{"enabled":true,"password":"secret"}},
	"access_restrictions":{"allowed_cidrs":["10.0.0.0/8"]},
	"private":true,"access_groups":["g1"],
	"meta":{"created_at":"2026-08-01T00:00:00Z","status":"active"}
}`

func TestListServicesReturnsDecodedList(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/reverse-proxies/services" {
			t.Errorf("path = %q, want /api/reverse-proxies/services", r.URL.Path)
		}
		_, _ = io.WriteString(w, "["+oneService+"]")
	})

	rec := httptest.NewRecorder()
	srv.listServices(rec, authed(http.MethodGet, "/api/bff/reverse-proxy/services", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got servicesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 1 || got.Services[0].Name != "myapp" {
		t.Errorf("got = %+v, want one service named myapp", got)
	}
}

func TestCreateServiceRejectsMissingTargets(t *testing.T) {
	var postCalled bool
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		postCalled = true
	})

	req := authed(http.MethodPost, "/api/bff/reverse-proxy/services",
		strings.NewReader(`{"name":"myapp","domain":"myapp.example.com","enabled":true,"targets":[]}`))
	rec := httptest.NewRecorder()
	srv.createService(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body=%s", rec.Code, rec.Body.String())
	}
	if postCalled {
		t.Error("a service with no targets must not reach the Management API")
	}
}

// TestUpdateServicePreservesAuthAndAccessRestrictions is the load-bearing
// test for this handler: the edit form only sends name/domain/mode/
// listen_port/targets/enabled/pass_host_header/rewrite_redirects, so
// updateService must fetch the current service and carry auth/
// access_restrictions/private/access_groups forward untouched rather than
// letting the PUT's full-replace semantics silently wipe them.
func TestUpdateServicePreservesAuthAndAccessRestrictions(t *testing.T) {
	var gotBody map[string]any
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/reverse-proxies/services/svc1":
			_, _ = io.WriteString(w, oneService)
		case r.Method == http.MethodPut && r.URL.Path == "/api/reverse-proxies/services/svc1":
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = io.WriteString(w, oneService)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	req := authed(http.MethodPut, "/api/bff/reverse-proxy/services/svc1", strings.NewReader(`{
		"name":"myapp-renamed","domain":"myapp.example.netbird.app","mode":"http","enabled":false,
		"targets":[{"target_id":"t1","target_type":"host","protocol":"http","host":"10.0.0.5","port":8080,"enabled":true}]
	}`))
	req.SetPathValue("id", "svc1")

	rec := httptest.NewRecorder()
	srv.updateService(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if gotBody["name"] != "myapp-renamed" {
		t.Errorf("upstream name = %v, want myapp-renamed", gotBody["name"])
	}
	auth, ok := gotBody["auth"].(map[string]any)
	if !ok || auth["password_auth"] == nil {
		t.Errorf("upstream auth = %v, want the fetched password_auth carried forward untouched", gotBody["auth"])
	}
	if gotBody["private"] != true {
		t.Errorf("upstream private = %v, want true (carried forward)", gotBody["private"])
	}
}

func TestDeleteServiceReturnsNoContent(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	req := authed(http.MethodDelete, "/api/bff/reverse-proxy/services/svc1", nil)
	req.SetPathValue("id", "svc1")

	rec := httptest.NewRecorder()
	srv.deleteService(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotPath != "/api/reverse-proxies/services/svc1" {
		t.Errorf("path = %q, want /api/reverse-proxies/services/svc1", gotPath)
	}
}

func TestListProxyClustersAndTokens(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/reverse-proxies/clusters":
			_, _ = io.WriteString(w, `[{"id":"c1","address":"eu.proxy.netbird.io","type":"shared","online":true,"connected_proxies":2}]`)
		case "/api/reverse-proxies/proxy-tokens":
			_, _ = io.WriteString(w, `[{"id":"tok1","name":"ci","created_at":"2026-08-01T00:00:00Z","revoked":false}]`)
		default:
			t.Errorf("unexpected path %q", r.URL.Path)
		}
	})

	rec := httptest.NewRecorder()
	srv.listProxyClusters(rec, authed(http.MethodGet, "/api/bff/reverse-proxy/clusters", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("clusters status = %d, want 200", rec.Code)
	}
	var clusters proxyClustersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &clusters); err != nil || clusters.Total != 1 {
		t.Fatalf("clusters decode/total wrong: err=%v total=%d", err, clusters.Total)
	}

	rec2 := httptest.NewRecorder()
	srv.listProxyTokens(rec2, authed(http.MethodGet, "/api/bff/reverse-proxy/proxy-tokens", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("tokens status = %d, want 200", rec2.Code)
	}
	var tokens proxyTokensResponse
	if err := json.Unmarshal(rec2.Body.Bytes(), &tokens); err != nil || tokens.Total != 1 {
		t.Fatalf("tokens decode/total wrong: err=%v total=%d", err, tokens.Total)
	}
}
