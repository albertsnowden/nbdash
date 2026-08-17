package nbapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestListUsersSortsByName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[{"id":"1","name":"zulu"},{"id":"2","name":"Ada"}]`)
	})

	users, err := client.ListUsers(context.Background(), "tok", nil)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(users) != 2 || users[0].Name != "Ada" || users[1].Name != "zulu" {
		t.Errorf("order = %v, want [Ada zulu]", users)
	}
}

func TestListUsersSendsServiceUserFilter(t *testing.T) {
	var gotQuery string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = io.WriteString(w, `[]`)
	})

	trueVal := true
	if _, err := client.ListUsers(context.Background(), "tok", &trueVal); err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if gotQuery != "service_user=true" {
		t.Errorf("query = %q, want service_user=true", gotQuery)
	}
}

func TestListUsersOmitsFilterWhenNil(t *testing.T) {
	var gotQuery string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_, _ = io.WriteString(w, `[]`)
	})

	if _, err := client.ListUsers(context.Background(), "tok", nil); err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (no filter)", gotQuery)
	}
}

func TestCreateUserSendsFields(t *testing.T) {
	var gotMethod string
	var gotBody UserCreateRequest
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"u1","name":"Ada"}`)
	})

	_, err := client.CreateUser(context.Background(), "tok", UserCreateRequest{
		Email: "ada@example.com", Name: "Ada", Role: "admin", AutoGroups: []string{"g1"},
	})
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotBody.Email != "ada@example.com" || gotBody.IsServiceUser {
		t.Errorf("unexpected body: %+v", gotBody)
	}
}

func TestUpdateUserPuts(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = io.WriteString(w, `{"id":"u1","role":"admin"}`)
	})

	_, err := client.UpdateUser(context.Background(), "tok", "u1", UserRequest{Role: "admin", AutoGroups: []string{}})
	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/users/u1" {
		t.Errorf("method/path = %s %s, want PUT /api/users/u1", gotMethod, gotPath)
	}
}

func TestDeleteUserPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteUser(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if gotPath != "/api/users/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

func TestInviteUserPostsToInvitePath(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	if err := client.InviteUser(context.Background(), "tok", "u1"); err != nil {
		t.Fatalf("InviteUser: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/users/u1/invite" {
		t.Errorf("method/path = %s %s, want POST /api/users/u1/invite", gotMethod, gotPath)
	}
}

func TestApproveUserPostsToApprovePath(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		_, _ = io.WriteString(w, `{"id":"u1"}`)
	})

	if _, err := client.ApproveUser(context.Background(), "tok", "u1"); err != nil {
		t.Fatalf("ApproveUser: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/api/users/u1/approve" {
		t.Errorf("method/path = %s %s, want POST /api/users/u1/approve", gotMethod, gotPath)
	}
}

func TestRejectUserDeletesToRejectPath(t *testing.T) {
	var gotMethod, gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	if err := client.RejectUser(context.Background(), "tok", "u1"); err != nil {
		t.Fatalf("RejectUser: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/users/u1/reject" {
		t.Errorf("method/path = %s %s, want DELETE /api/users/u1/reject", gotMethod, gotPath)
	}
}

func TestListPATsAtUserPath(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = io.WriteString(w, `[{"id":"pat1","name":"CI token"}]`)
	})

	pats, err := client.ListPATs(context.Background(), "tok", "u1")
	if err != nil {
		t.Fatalf("ListPATs: %v", err)
	}
	if gotPath != "/api/users/u1/tokens" {
		t.Errorf("path = %q, want /api/users/u1/tokens", gotPath)
	}
	if len(pats) != 1 || pats[0].Name != "CI token" {
		t.Errorf("pats = %+v", pats)
	}
}

func TestCreatePATReturnsPlaintextOnce(t *testing.T) {
	var gotBody PersonalAccessTokenRequest
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"plain_token":"nbp_SECRET123","personal_access_token":{"id":"pat1","name":"CI token"}}`)
	})

	generated, err := client.CreatePAT(context.Background(), "tok", "u1", PersonalAccessTokenRequest{Name: "CI token", ExpiresIn: 90})
	if err != nil {
		t.Fatalf("CreatePAT: %v", err)
	}
	if generated.PlainToken != "nbp_SECRET123" {
		t.Errorf("PlainToken = %q, want nbp_SECRET123", generated.PlainToken)
	}
	if gotBody.ExpiresIn != 90 {
		t.Errorf("ExpiresIn = %d, want 90", gotBody.ExpiresIn)
	}
}

func TestDeletePATPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeletePAT(context.Background(), "tok", "u1", "a/../b"); err != nil {
		t.Fatalf("DeletePAT: %v", err)
	}
	if gotPath != "/api/users/u1/tokens/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}
