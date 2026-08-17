package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const twoNetworks = `[
	{"id":"n1","name":"office","routers":["r1"],"resources":["res1"]},
	{"id":"n2","name":"cloud","routers":[],"resources":[]}
]`

func TestListNetworksReturnsAll(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/networks" {
			t.Errorf("path = %q, want /api/networks", r.URL.Path)
		}
		_, _ = io.WriteString(w, twoNetworks)
	})

	rec := httptest.NewRecorder()
	srv.listNetworks(rec, authed(http.MethodGet, "/api/bff/networks", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got networksResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 2 {
		t.Errorf("Total = %d, want 2", got.Total)
	}
}

func TestCreateNetworkRejectsEmptyName(t *testing.T) {
	var posted bool
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		posted = true
	})

	rec := httptest.NewRecorder()
	srv.createNetwork(rec, authed(http.MethodPost, "/api/bff/networks", strings.NewReader(`{"name":"  "}`)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
	if posted {
		t.Error("an empty name must not reach the Management API")
	}
}

func TestDeleteNetworkReturnsNoContent(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	req := authed(http.MethodDelete, "/api/bff/networks/n1", nil)
	req.SetPathValue("id", "n1")

	rec := httptest.NewRecorder()
	srv.deleteNetwork(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotPath != "/api/networks/n1" {
		t.Errorf("path = %q, want /api/networks/n1", gotPath)
	}
}

func TestCreateResourceRequiresAddress(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("a missing address must not reach the Management API")
	})

	req := authed(http.MethodPost, "/api/bff/networks/n1/resources", strings.NewReader(`{"name":"web","address":""}`))
	req.SetPathValue("id", "n1")

	rec := httptest.NewRecorder()
	srv.createResource(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestCreateResourceSendsGroups(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"res1","name":"web","address":"10.0.0.1","type":"host","groups":[]}`)
	})

	body := `{"name":"web","address":"10.0.0.1","groups":["g1","g2"]}`
	req := authed(http.MethodPost, "/api/bff/networks/n1/resources", strings.NewReader(body))
	req.SetPathValue("id", "n1")

	rec := httptest.NewRecorder()
	srv.createResource(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	if gotPath != "/api/networks/n1/resources" {
		t.Errorf("path = %q, want /api/networks/n1/resources", gotPath)
	}
	groups, _ := gotBody["groups"].([]any)
	if len(groups) != 2 {
		t.Errorf("groups = %v, want 2 entries", gotBody["groups"])
	}
}

func TestDeleteResourceUsesNestedPath(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	req := authed(http.MethodDelete, "/api/bff/networks/n1/resources/res1", nil)
	req.SetPathValue("id", "n1")
	req.SetPathValue("resourceID", "res1")

	rec := httptest.NewRecorder()
	srv.deleteResource(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotPath != "/api/networks/n1/resources/res1" {
		t.Errorf("path = %q, want /api/networks/n1/resources/res1", gotPath)
	}
}

func TestCreateRouterRejectsBothPeerAndPeerGroups(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("an invalid peer/peer_groups combination must not reach the Management API")
	})

	body := `{"peer":"p1","peer_groups":["g1"],"metric":100}`
	req := authed(http.MethodPost, "/api/bff/networks/n1/routers", strings.NewReader(body))
	req.SetPathValue("id", "n1")

	rec := httptest.NewRecorder()
	srv.createRouter(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestCreateRouterForcesEnabledTrue(t *testing.T) {
	var gotBody map[string]any
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"r1","peer":"p1","metric":100,"enabled":true}`)
	})

	body := `{"peer":"p1","metric":100,"enabled":false}`
	req := authed(http.MethodPost, "/api/bff/networks/n1/routers", strings.NewReader(body))
	req.SetPathValue("id", "n1")

	rec := httptest.NewRecorder()
	srv.createRouter(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	if enabled, _ := gotBody["enabled"].(bool); !enabled {
		t.Errorf("enabled sent = %v, want true (server forces it on create)", gotBody["enabled"])
	}
}

func TestUpdateRouterHonoursEnabledField(t *testing.T) {
	var gotBody map[string]any
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"r1","peer":"p1","metric":100,"enabled":false}`)
	})

	body := `{"peer":"p1","metric":100,"enabled":false}`
	req := authed(http.MethodPut, "/api/bff/networks/n1/routers/r1", strings.NewReader(body))
	req.SetPathValue("id", "n1")
	req.SetPathValue("routerID", "r1")

	rec := httptest.NewRecorder()
	srv.updateRouter(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if enabled, _ := gotBody["enabled"].(bool); enabled {
		t.Errorf("enabled sent = %v, want false (update honours the request)", gotBody["enabled"])
	}
}

func TestCreateRouterRejectsBadMetric(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("an out-of-range metric must not reach the Management API")
	})

	body := `{"peer":"p1","metric":10000}`
	req := authed(http.MethodPost, "/api/bff/networks/n1/routers", strings.NewReader(body))
	req.SetPathValue("id", "n1")

	rec := httptest.NewRecorder()
	srv.createRouter(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}
