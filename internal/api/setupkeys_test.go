package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

const twoSetupKeys = `[
	{"id":"sk1","name":"office","type":"reusable","state":"valid","valid":true,"auto_groups":["g1"]},
	{"id":"sk2","name":"old","type":"one-off","state":"revoked","valid":false,"revoked":true,"auto_groups":[]}
]`

func TestListSetupKeysReturnsCounts(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/setup-keys" {
			t.Errorf("path = %q, want /api/setup-keys", r.URL.Path)
		}
		_, _ = io.WriteString(w, twoSetupKeys)
	})

	rec := httptest.NewRecorder()
	srv.listSetupKeys(rec, authed(http.MethodGet, "/api/bff/setup-keys", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got setupKeysResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 2 || got.Valid != 1 {
		t.Errorf("Total/Valid = %d/%d, want 2/1", got.Total, got.Valid)
	}
}

func TestCreateSetupKeyReturnsPlaintext(t *testing.T) {
	var gotBody nbapi.CreateSetupKeyRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"sk1","name":"bootstrap","key":"PLAINTEXT-ABCD",
			"type":"one-off","state":"valid","valid":true,"auto_groups":[]}`)
	})

	body := `{"name":"bootstrap","type":"one-off","expires_in_days":1,"auto_groups":["g1","g2"],"usage_limit":1}`
	rec := httptest.NewRecorder()
	srv.createSetupKey(rec, authed(http.MethodPost, "/api/bff/setup-keys", strings.NewReader(body)))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	if gotBody.ExpiresIn != 86400 {
		t.Errorf("ExpiresIn = %d, want 86400 (1 day)", gotBody.ExpiresIn)
	}
	if len(gotBody.AutoGroups) != 2 {
		t.Errorf("AutoGroups = %v, want 2 entries", gotBody.AutoGroups)
	}

	var got nbapi.SetupKey
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Key != "PLAINTEXT-ABCD" {
		t.Errorf("Key = %q, want the plaintext value round-tripped", got.Key)
	}
}

func TestCreateSetupKeyRejectsBadExpiry(t *testing.T) {
	var posted bool
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		posted = true
	})

	body := `{"name":"x","type":"one-off","expires_in_days":400}`
	rec := httptest.NewRecorder()
	srv.createSetupKey(rec, authed(http.MethodPost, "/api/bff/setup-keys", strings.NewReader(body)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
	if posted {
		t.Error("an out-of-range expiry must not reach the Management API")
	}
}

func TestCreateSetupKeyRejectsBadType(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("an invalid type must not reach the Management API")
	})

	body := `{"name":"x","type":"forever","expires_in_days":1}`
	rec := httptest.NewRecorder()
	srv.createSetupKey(rec, authed(http.MethodPost, "/api/bff/setup-keys", strings.NewReader(body)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

// revokeSetupKey must fetch the key's current groups and resend them — the
// API rejects a nil auto_groups outright on PUT.
func TestRevokeSetupKeyPreservesAutoGroups(t *testing.T) {
	var gotMethod string
	var gotBody nbapi.SetupKeyRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, `{"id":"sk1","name":"office","auto_groups":["g1","g2"]}`)
		case http.MethodPut:
			gotMethod = r.Method
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = io.WriteString(w, `{"id":"sk1","name":"office","revoked":true,"auto_groups":["g1","g2"]}`)
		}
	})

	req := authed(http.MethodPost, "/api/bff/setup-keys/sk1/revoke", nil)
	req.SetPathValue("id", "sk1")

	rec := httptest.NewRecorder()
	srv.revokeSetupKey(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if gotMethod != http.MethodPut {
		t.Error("revoke should PUT")
	}
	if !gotBody.Revoked {
		t.Error("Revoked should have round-tripped as true")
	}
	if len(gotBody.AutoGroups) != 2 {
		t.Errorf("AutoGroups = %v, want the key's existing 2 groups resent", gotBody.AutoGroups)
	}
}

// The key's current auto_groups can legitimately be nil (never set) — this
// must not be forwarded as JSON null, which the API rejects.
func TestRevokeSetupKeyNormalizesNilAutoGroups(t *testing.T) {
	var rawBody []byte
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, `{"id":"sk1","name":"office"}`)
		case http.MethodPut:
			rawBody, _ = io.ReadAll(r.Body)
			_, _ = io.WriteString(w, `{"id":"sk1","revoked":true,"auto_groups":[]}`)
		}
	})

	req := authed(http.MethodPost, "/api/bff/setup-keys/sk1/revoke", nil)
	req.SetPathValue("id", "sk1")

	rec := httptest.NewRecorder()
	srv.revokeSetupKey(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var decoded map[string]any
	if err := json.Unmarshal(rawBody, &decoded); err != nil {
		t.Fatalf("decode sent body: %v", err)
	}
	if decoded["auto_groups"] == nil {
		t.Fatalf("auto_groups sent as JSON null, want []: body=%s", rawBody)
	}
}

func TestDeleteSetupKeyReturnsNoContent(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	req := authed(http.MethodDelete, "/api/bff/setup-keys/sk1", nil)
	req.SetPathValue("id", "sk1")

	rec := httptest.NewRecorder()
	srv.deleteSetupKey(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotPath != "/api/setup-keys/sk1" {
		t.Errorf("path = %q, want /api/setup-keys/sk1", gotPath)
	}
}
