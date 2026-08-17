package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func validNameserverGroupBody() string {
	return `{"name":"Cloudflare","nameservers":[{"ip":"1.1.1.1","port":53}],
		"enabled":true,"groups":["g1"],"primary":true}`
}

func TestCreateNameserverGroupRejectsTooManyNameservers(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("more than 3 nameservers must not reach the Management API")
	})

	body := `{"name":"x","nameservers":[{"ip":"1.1.1.1","port":53},{"ip":"1.0.0.1","port":53},
		{"ip":"8.8.8.8","port":53},{"ip":"8.8.4.4","port":53}],"groups":["g1"],"primary":true}`
	rec := httptest.NewRecorder()
	srv.createNameserverGroup(rec, authed(http.MethodPost, "/api/bff/dns/nameservers", strings.NewReader(body)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestCreateNameserverGroupRejectsPrimaryWithDomains(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("primary + domains must not reach the Management API")
	})

	body := `{"name":"x","nameservers":[{"ip":"1.1.1.1","port":53}],"groups":["g1"],
		"primary":true,"domains":["example.com"]}`
	rec := httptest.NewRecorder()
	srv.createNameserverGroup(rec, authed(http.MethodPost, "/api/bff/dns/nameservers", strings.NewReader(body)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestCreateNameserverGroupForcesUDP(t *testing.T) {
	var gotBody map[string]any
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"ns1","name":"Cloudflare","nameservers":[{"ip":"1.1.1.1","ns_type":"udp","port":53}]}`)
	})

	rec := httptest.NewRecorder()
	srv.createNameserverGroup(rec, authed(http.MethodPost, "/api/bff/dns/nameservers", strings.NewReader(validNameserverGroupBody())))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	nameservers, _ := gotBody["nameservers"].([]any)
	if len(nameservers) != 1 {
		t.Fatalf("nameservers = %v", gotBody["nameservers"])
	}
	first, _ := nameservers[0].(map[string]any)
	if first["ns_type"] != "udp" {
		t.Errorf("ns_type = %v, want udp", first["ns_type"])
	}
}

func TestDNSSettingsRoundTrip(t *testing.T) {
	var gotMethod string
	var gotBody map[string]any
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		if r.Method == http.MethodPut {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
		}
		_, _ = io.WriteString(w, `{"disabled_management_groups":["g2"]}`)
	})

	rec := httptest.NewRecorder()
	srv.updateDNSSettings(rec, authed(http.MethodPut, "/api/bff/dns/settings",
		strings.NewReader(`{"disabled_management_groups":["g2"]}`)))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if gotMethod != http.MethodPut {
		t.Error("update should PUT")
	}
	groups, _ := gotBody["disabled_management_groups"].([]any)
	if len(groups) != 1 {
		t.Errorf("disabled_management_groups = %v, want 1 entry", gotBody["disabled_management_groups"])
	}
}

func TestCreateZoneRejectsEmptyDomain(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("an empty domain must not reach the Management API")
	})

	body := `{"name":"Office","domain":"","distribution_groups":["g1"]}`
	rec := httptest.NewRecorder()
	srv.createZone(rec, authed(http.MethodPost, "/api/bff/dns/zones", strings.NewReader(body)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

// updateZoneMeta must carry the zone's real Domain forward regardless of
// what the request body says — the server rejects a domain change outright.
func TestUpdateZoneMetaCarriesDomainForward(t *testing.T) {
	var gotBody map[string]any
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, `{"id":"z1","name":"Office","domain":"example.com","distribution_groups":["g1"]}`)
		case http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = io.WriteString(w, `{"id":"z1","name":"renamed","domain":"example.com"}`)
		}
	})

	body := `{"name":"renamed","domain":"attacker-controlled.com","distribution_groups":["g1"]}`
	req := authed(http.MethodPut, "/api/bff/dns/zones/z1", strings.NewReader(body))
	req.SetPathValue("id", "z1")

	rec := httptest.NewRecorder()
	srv.updateZoneMeta(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if gotBody["domain"] != "example.com" {
		t.Errorf("domain sent = %v, want the zone's real domain carried forward, not the request body's", gotBody["domain"])
	}
}

func TestCreateRecordRejectsInvalidType(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("an invalid record type must not reach the Management API")
	})

	body := `{"name":"www","type":"MX","content":"10 mail.example.com","ttl":300}`
	req := authed(http.MethodPost, "/api/bff/dns/zones/z1/records", strings.NewReader(body))
	req.SetPathValue("id", "z1")

	rec := httptest.NewRecorder()
	srv.createRecord(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestCreateRecordRejectsNegativeTTL(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("a negative TTL must not reach the Management API")
	})

	body := `{"name":"www","type":"A","content":"192.168.1.1","ttl":-1}`
	req := authed(http.MethodPost, "/api/bff/dns/zones/z1/records", strings.NewReader(body))
	req.SetPathValue("id", "z1")

	rec := httptest.NewRecorder()
	srv.createRecord(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestDeleteRecordUsesNestedPath(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	req := authed(http.MethodDelete, "/api/bff/dns/zones/z1/records/rec1", nil)
	req.SetPathValue("id", "z1")
	req.SetPathValue("recordID", "rec1")

	rec := httptest.NewRecorder()
	srv.deleteRecord(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotPath != "/api/dns/zones/z1/records/rec1" {
		t.Errorf("path = %q, want /api/dns/zones/z1/records/rec1", gotPath)
	}
}
