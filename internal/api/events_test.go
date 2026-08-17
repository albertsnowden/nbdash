package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

const twoEvents = `[
	{"id":"1","timestamp":"2026-08-10T10:00:00Z","activity":"Peer added","activity_code":"peer.user.add","initiator_id":"u1","initiator_name":"Ada","initiator_email":"ada@example.com","target_id":"p1","meta":{}},
	{"id":"2","timestamp":"2026-08-15T10:00:00Z","activity":"Group created","activity_code":"group.add","initiator_id":"u1","initiator_name":"Ada","initiator_email":"ada@example.com","target_id":"g1","meta":{"name":"engineering"}}
]`

func TestListAuditEventsReturnsNewestFirst(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/events/audit" {
			t.Errorf("path = %q, want /api/events/audit", r.URL.Path)
		}
		_, _ = io.WriteString(w, twoEvents)
	})

	rec := httptest.NewRecorder()
	srv.listAuditEvents(rec, authed(http.MethodGet, "/api/bff/events/audit", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got eventsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 2 || len(got.Events) != 2 {
		t.Fatalf("Total/len(Events) = %d/%d, want 2/2", got.Total, len(got.Events))
	}
	if got.Events[0].ID != "2" {
		t.Errorf("Events[0].ID = %q, want 2 (most recent first)", got.Events[0].ID)
	}
}

func TestListAuditEventsRequiresAuth(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("should never reach the Management API without a session")
	})

	rec := httptest.NewRecorder()
	srv.listAuditEvents(rec, httptest.NewRequest(http.MethodGet, "/api/bff/events/audit", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
