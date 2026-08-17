package nbapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestListSetupKeysSendsBearerTokenToAPIPath(t *testing.T) {
	var gotPath, gotAuth string

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `[]`)
	})

	if _, err := client.ListSetupKeys(context.Background(), "tok-123"); err != nil {
		t.Fatalf("ListSetupKeys: %v", err)
	}

	if gotPath != "/api/setup-keys" {
		t.Errorf("path = %q, want /api/setup-keys", gotPath)
	}
	if gotAuth != "Bearer tok-123" {
		t.Errorf("Authorization = %q, want Bearer tok-123", gotAuth)
	}
}

func TestListSetupKeysSortsValidFirstThenByName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[
			{"id":"1","name":"zulu-key","valid":true,"state":"valid"},
			{"id":"2","name":"alpha-key","valid":false,"state":"revoked"},
			{"id":"3","name":"Bravo-key","valid":true,"state":"valid"},
			{"id":"4","name":"charlie-key","valid":false,"state":"expired"}
		]`)
	})

	keys, err := client.ListSetupKeys(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ListSetupKeys: %v", err)
	}

	var got []string
	for _, k := range keys {
		got = append(got, k.Name)
	}
	want := []string{"Bravo-key", "zulu-key", "alpha-key", "charlie-key"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestGetSetupKeyPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = io.WriteString(w, `{}`)
	})

	if _, err := client.GetSetupKey(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("GetSetupKey: %v", err)
	}
	if gotPath != "/api/setup-keys/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

// The plaintext key only ever appears in the create response. This is the
// nbapi layer's half of that contract: it must round-trip whatever the server
// sends, masked or not, without special-casing either.
func TestCreateSetupKeyReturnsPlaintextKeyFromResponse(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody CreateSetupKeyRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"sk1","name":"bootstrap","key":"PLAINTEXT-ABCD-1234",
			"type":"one-off","state":"valid","valid":true,"auto_groups":[]}`)
	})

	key, err := client.CreateSetupKey(context.Background(), "tok", CreateSetupKeyRequest{
		Name:       "bootstrap",
		Type:       "one-off",
		ExpiresIn:  86400,
		AutoGroups: []string{"g1", "g2"},
		UsageLimit: 1,
	})
	if err != nil {
		t.Fatalf("CreateSetupKey: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/api/setup-keys" {
		t.Errorf("path = %q, want /api/setup-keys", gotPath)
	}
	if gotBody.Name != "bootstrap" || gotBody.Type != "one-off" || gotBody.ExpiresIn != 86400 {
		t.Errorf("unexpected request body: %+v", gotBody)
	}
	if key.Key != "PLAINTEXT-ABCD-1234" {
		t.Errorf("Key = %q, want the plaintext value from the response", key.Key)
	}
}

func TestUpdateSetupKeySendsFullEditableSet(t *testing.T) {
	var gotMethod string
	var gotBody SetupKeyRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"sk1","name":"bootstrap","key":"A6160****",
			"revoked":true,"state":"revoked"}`)
	})

	_, err := client.UpdateSetupKey(context.Background(), "tok", "sk1", SetupKeyRequest{
		Revoked:    true,
		AutoGroups: []string{"g1"},
	})
	if err != nil {
		t.Fatalf("UpdateSetupKey: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	if !gotBody.Revoked {
		t.Error("Revoked should have round-tripped as true")
	}
	if len(gotBody.AutoGroups) != 1 || gotBody.AutoGroups[0] != "g1" {
		t.Errorf("AutoGroups = %v, want [g1]", gotBody.AutoGroups)
	}
}

// The API rejects a nil auto_groups outright. A Go nil slice marshals to
// JSON null, which is exactly that rejected value — this pins that an empty
// slice (valid: "[]") and a nil slice (invalid: "null") are distinguishable
// over the wire, so a caller passing []string{} does not accidentally send
// null.
func TestUpdateSetupKeySendsEmptyArrayNotNullForEmptyAutoGroups(t *testing.T) {
	var rawBody []byte
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		rawBody, _ = io.ReadAll(r.Body)
		_, _ = io.WriteString(w, `{"id":"sk1"}`)
	})

	_, err := client.UpdateSetupKey(context.Background(), "tok", "sk1", SetupKeyRequest{
		Revoked:    true,
		AutoGroups: []string{},
	})
	if err != nil {
		t.Fatalf("UpdateSetupKey: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(rawBody, &decoded); err != nil {
		t.Fatalf("decode sent body: %v", err)
	}
	if decoded["auto_groups"] == nil {
		t.Fatalf("auto_groups was sent as JSON null, want []: body=%s", rawBody)
	}
	groups, ok := decoded["auto_groups"].([]any)
	if !ok || len(groups) != 0 {
		t.Errorf("auto_groups = %v, want an empty array", decoded["auto_groups"])
	}
}

func TestDeleteSetupKeyIgnoresEmptyBody(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteSetupKey(context.Background(), "tok", "sk1"); err != nil {
		t.Fatalf("DeleteSetupKey: %v", err)
	}
}

func TestDeleteSetupKeyPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteSetupKey(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("DeleteSetupKey: %v", err)
	}
	if gotPath != "/api/setup-keys/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

// The real server's GetLastUsed()/GetExpiresAt() return Go's time.Time zero
// value for a never-used/unset key, which marshals to a valid RFC3339 string
// ("0001-01-01T00:00:00Z") — not an empty string and not JSON null. This
// pins that the actual wire format round-trips into a Go zero time.Time,
// which is what relTime/absTime key off of to render "never"/"—". A stub
// server built during manual verification once emitted "" instead and broke
// decoding entirely (encoding/json's time.Time only special-cases null, not
// an empty string) — this is that failure mode, pinned so it can't recur.
func TestZeroTimeFieldsDecodeFromRealServerWireFormat(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":"sk1","name":"never-used",
			"expires":"0001-01-01T00:00:00Z","last_used":"0001-01-01T00:00:00Z",
			"updated_at":"0001-01-01T00:00:00Z","auto_groups":[]}`)
	})

	key, err := client.GetSetupKey(context.Background(), "tok", "sk1")
	if err != nil {
		t.Fatalf("GetSetupKey: %v", err)
	}
	if !key.LastUsed.IsZero() {
		t.Errorf("LastUsed = %v, want the zero value", key.LastUsed)
	}
	if !key.Expires.IsZero() {
		t.Errorf("Expires = %v, want the zero value", key.Expires)
	}
}
