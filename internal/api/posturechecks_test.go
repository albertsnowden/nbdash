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

const twoPostureChecks = `[
	{"id":"pc1","name":"Zeta","checks":{}},
	{"id":"pc2","name":"Alpha","checks":{"nb_version_check":{"min_version":"0.30.0"}}}
]`

func TestListPostureChecksSortsByName(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/posture-checks" {
			t.Errorf("path = %q, want /api/posture-checks", r.URL.Path)
		}
		_, _ = io.WriteString(w, twoPostureChecks)
	})

	rec := httptest.NewRecorder()
	srv.listPostureChecks(rec, authed(http.MethodGet, "/api/bff/posture-checks", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got postureChecksResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 2 || len(got.PostureChecks) != 2 {
		t.Fatalf("Total/len = %d/%d, want 2/2", got.Total, len(got.PostureChecks))
	}
	if got.PostureChecks[0].Name != "Alpha" {
		t.Errorf("PostureChecks[0].Name = %q, want Alpha (alphabetical)", got.PostureChecks[0].Name)
	}
}

func TestCreatePostureCheckSendsChecksAndRejectsEmptyName(t *testing.T) {
	var gotBody nbapi.PostureCheckRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"pc3","name":"Recent NetBird","checks":{"nb_version_check":{"min_version":"0.30.0"}}}`)
	})

	req := authed(http.MethodPost, "/api/bff/posture-checks",
		strings.NewReader(`{"name":"Recent NetBird","description":"","checks":{"nb_version_check":{"min_version":"0.30.0"}}}`))
	rec := httptest.NewRecorder()
	srv.createPostureCheck(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	if gotBody.Checks.NBVersionCheck == nil || gotBody.Checks.NBVersionCheck.MinVersion != "0.30.0" {
		t.Errorf("upstream nb_version_check = %+v, want min_version 0.30.0", gotBody.Checks.NBVersionCheck)
	}

	// Empty name must be rejected before reaching the Management API.
	empty := authed(http.MethodPost, "/api/bff/posture-checks", strings.NewReader(`{"name":"  ","checks":{}}`))
	rec2 := httptest.NewRecorder()
	srv.createPostureCheck(rec2, empty)
	if rec2.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422 for empty name", rec2.Code)
	}
}

func TestDeletePostureCheckReturnsNoContent(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	req := authed(http.MethodDelete, "/api/bff/posture-checks/pc1", nil)
	req.SetPathValue("id", "pc1")

	rec := httptest.NewRecorder()
	srv.deletePostureCheck(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotPath != "/api/posture-checks/pc1" {
		t.Errorf("path = %q, want /api/posture-checks/pc1", gotPath)
	}
}
