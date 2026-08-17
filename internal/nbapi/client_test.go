package nbapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(srv.URL)
}

func TestListPeersSendsBearerTokenToAPIPath(t *testing.T) {
	var gotPath, gotAuth string

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `[]`)
	})

	if _, err := client.ListPeers(context.Background(), "tok-123"); err != nil {
		t.Fatalf("ListPeers: %v", err)
	}

	// The "/api" prefix mirrors the Next.js dashboard's apiOrigin + "/api".
	if gotPath != "/api/peers" {
		t.Errorf("path = %q, want /api/peers", gotPath)
	}
	if gotAuth != "Bearer tok-123" {
		t.Errorf("Authorization = %q, want Bearer tok-123", gotAuth)
	}
}

func TestListPeersSortsConnectedFirstThenByName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[
			{"id":"1","name":"zulu","connected":true},
			{"id":"2","name":"alpha","connected":false},
			{"id":"3","name":"Bravo","connected":true},
			{"id":"4","name":"charlie","connected":false}
		]`)
	})

	peers, err := client.ListPeers(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ListPeers: %v", err)
	}

	var got []string
	for _, p := range peers {
		got = append(got, p.Name)
	}
	want := []string{"Bravo", "zulu", "alpha", "charlie"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestUpdatePeerSendsFullEditableSet(t *testing.T) {
	var body PeerRequest
	var method string

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = io.WriteString(w, `{"id":"p1","name":"renamed"}`)
	})

	peer, err := client.UpdatePeer(context.Background(), "tok", "p1", PeerRequest{
		Name:                   "renamed",
		SSHEnabled:             true,
		LoginExpirationEnabled: false,
	})
	if err != nil {
		t.Fatalf("UpdatePeer: %v", err)
	}

	if method != http.MethodPut {
		t.Errorf("method = %s, want PUT", method)
	}
	if body.Name != "renamed" || !body.SSHEnabled {
		t.Errorf("unexpected request body: %+v", body)
	}
	// login_expiration_enabled must be present-and-false, not omitted: the API
	// replaces the whole set, so an omitted field would be read as false anyway
	// but an omitted *required* field is a 400.
	if body.LoginExpirationEnabled {
		t.Error("LoginExpirationEnabled should have round-tripped as false")
	}
	if peer.Name != "renamed" {
		t.Errorf("peer.Name = %q, want renamed", peer.Name)
	}
}

func TestErrorEnvelopeIsDecoded(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(requestIDHeader, "req-abc")
		w.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(w, `{"code":403,"message":"user is blocked"}`)
	})

	_, err := client.GetPeer(context.Background(), "tok", "p1")
	if err == nil {
		t.Fatal("expected an error")
	}

	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("error was not *nbapi.Error: %T", err)
	}
	if apiErr.Code != http.StatusForbidden {
		t.Errorf("Code = %d, want 403", apiErr.Code)
	}
	if apiErr.Message != "user is blocked" {
		t.Errorf("Message = %q", apiErr.Message)
	}
	if apiErr.RequestID != "req-abc" {
		t.Errorf("RequestID = %q, want req-abc", apiErr.RequestID)
	}
}

// A proxy in front of Management returns HTML, not the JSON envelope. The
// client must still produce a usable error rather than a decode failure.
func TestNonJSONErrorBodyFallsBackToStatus(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = io.WriteString(w, `<html><body>502 Bad Gateway</body></html>`)
	})

	_, err := client.GetPeer(context.Background(), "tok", "p1")

	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("error was not *nbapi.Error: %T (%v)", err, err)
	}
	if apiErr.Code != http.StatusBadGateway {
		t.Errorf("Code = %d, want 502", apiErr.Code)
	}
	if apiErr.Message == "" {
		t.Error("Message should fall back to the HTTP status")
	}
}

func TestIsUnauthorized(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"code":401,"message":"token invalid"}`)
	})

	_, err := client.ListPeers(context.Background(), "tok")
	if !IsUnauthorized(err) {
		t.Errorf("IsUnauthorized(%v) = false, want true", err)
	}
	if IsUnauthorized(errors.New("network down")) {
		t.Error("a plain error must not report as unauthorized")
	}
}

func TestDeletePeerIgnoresEmptyBody(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeletePeer(context.Background(), "tok", "p1"); err != nil {
		t.Fatalf("DeletePeer: %v", err)
	}
}

// Peer IDs go into the path, so they must be escaped rather than concatenated.
func TestPeerIDIsPathEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = io.WriteString(w, `{}`)
	})

	if _, err := client.GetPeer(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("GetPeer: %v", err)
	}
	if gotPath != "/api/peers/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}
