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

const twoGroups = `[
	{"id":"g1","name":"All","peers_count":2,"peers":[{"id":"p1","name":"gateway-01"},{"id":"p2","name":"laptop-02"}]},
	{"id":"g2","name":"devs","peers_count":1,"peers":[{"id":"p2","name":"laptop-02"}]}
]`

const oneGroup = `{"id":"g2","name":"devs","peers_count":1,"peers":[{"id":"p2","name":"laptop-02"}]}`

func TestListGroupsReturnsAll(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/groups" {
			t.Errorf("path = %q, want /api/groups", r.URL.Path)
		}
		_, _ = io.WriteString(w, twoGroups)
	})

	rec := httptest.NewRecorder()
	srv.listGroups(rec, authed(http.MethodGet, "/api/bff/groups", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got groupsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 2 || len(got.Groups) != 2 {
		t.Errorf("Total/len(Groups) = %d/%d, want 2/2", got.Total, len(got.Groups))
	}
}

func TestGetGroupReturnsResolvedPeers(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/groups/g2" {
			t.Errorf("path = %q, want /api/groups/g2", r.URL.Path)
		}
		_, _ = io.WriteString(w, oneGroup)
	})

	req := authed(http.MethodGet, "/api/bff/groups/g2", nil)
	req.SetPathValue("id", "g2")

	rec := httptest.NewRecorder()
	srv.getGroup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got nbapi.Group
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Name != "devs" || len(got.Peers) != 1 || got.Peers[0].Name != "laptop-02" {
		t.Errorf("got = %+v, want devs with laptop-02", got)
	}
}

func TestCreateGroupSendsFields(t *testing.T) {
	var gotMethod string
	var gotBody nbapi.GroupRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"g3","name":"new-group","peers":[{"id":"p1","name":"gateway-01"}]}`)
	})

	req := authed(http.MethodPost, "/api/bff/groups", strings.NewReader(`{"name":"  new-group  ","peers":["p1"]}`))
	rec := httptest.NewRecorder()
	srv.createGroup(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotBody.Name != "new-group" || len(gotBody.Peers) != 1 || gotBody.Peers[0] != "p1" {
		t.Errorf("unexpected upstream body: %+v", gotBody)
	}
}

func TestCreateGroupRejectsEmptyName(t *testing.T) {
	var posted bool
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		posted = true
	})

	rec := httptest.NewRecorder()
	srv.createGroup(rec, authed(http.MethodPost, "/api/bff/groups", strings.NewReader(`{"name":"   "}`)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
	if posted {
		t.Error("an empty name must not reach the Management API")
	}
}

func TestUpdateGroupSendsFullMembership(t *testing.T) {
	var gotMethod string
	var gotBody nbapi.GroupRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"g2","name":"renamed","peers":[{"id":"p1","name":"gateway-01"}]}`)
	})

	req := authed(http.MethodPut, "/api/bff/groups/g2", strings.NewReader(`{"name":"renamed","peers":["p1"]}`))
	req.SetPathValue("id", "g2")

	rec := httptest.NewRecorder()
	srv.updateGroup(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if gotMethod != http.MethodPut || gotBody.Name != "renamed" || len(gotBody.Peers) != 1 {
		t.Errorf("unexpected upstream call: method=%s body=%+v", gotMethod, gotBody)
	}
}

func TestDeleteGroupReturnsNoContent(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	req := authed(http.MethodDelete, "/api/bff/groups/g2", nil)
	req.SetPathValue("id", "g2")

	rec := httptest.NewRecorder()
	srv.deleteGroup(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotPath != "/api/groups/g2" {
		t.Errorf("path = %q, want /api/groups/g2", gotPath)
	}
}

// The server rejects deleting a group still referenced elsewhere (a policy,
// route, DNS zone, ...) with a normal API error naming what it's linked to
// — this pins that error surfaces as our usual JSON error envelope, not
// swallowed or mangled.
func TestDeleteGroupSurfacesLinkedError(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":400,"message":"group has been linked to policy: my-policy"}`)
	})

	req := authed(http.MethodDelete, "/api/bff/groups/g2", nil)
	req.SetPathValue("id", "g2")

	rec := httptest.NewRecorder()
	srv.deleteGroup(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	var env errorEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("decode error envelope: %v", err)
	}
	if !strings.Contains(env.Error.Message, "linked to policy") {
		t.Errorf("message = %q, want the upstream linkage error", env.Error.Message)
	}
}
