package nbapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestListGroupsSendsBearerTokenToAPIPath(t *testing.T) {
	var gotPath, gotAuth string

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `[]`)
	})

	if _, err := client.ListGroups(context.Background(), "tok-123"); err != nil {
		t.Fatalf("ListGroups: %v", err)
	}

	if gotPath != "/api/groups" {
		t.Errorf("path = %q, want /api/groups", gotPath)
	}
	if gotAuth != "Bearer tok-123" {
		t.Errorf("Authorization = %q, want Bearer tok-123", gotAuth)
	}
}

func TestListGroupsSortsByName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[
			{"id":"1","name":"zulu"},
			{"id":"2","name":"alpha"},
			{"id":"3","name":"Bravo"}
		]`)
	})

	groups, err := client.ListGroups(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ListGroups: %v", err)
	}

	var got []string
	for _, g := range groups {
		got = append(got, g.Name)
	}
	want := []string{"alpha", "Bravo", "zulu"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestGetGroupReturnsResolvedPeers(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":"g1","name":"devs","peers_count":2,
			"peers":[{"id":"p1","name":"gateway-01"},{"id":"p2","name":"laptop-02"}]}`)
	})

	group, err := client.GetGroup(context.Background(), "tok", "g1")
	if err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if len(group.Peers) != 2 || group.Peers[0].Name != "gateway-01" {
		t.Errorf("Peers = %+v, want resolved id+name pairs", group.Peers)
	}
}

func TestGetGroupPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = io.WriteString(w, `{}`)
	})

	if _, err := client.GetGroup(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("GetGroup: %v", err)
	}
	if gotPath != "/api/groups/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

func TestCreateGroupSendsNameAndPeers(t *testing.T) {
	var gotMethod string
	var gotBody GroupRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"g1","name":"devs","peers":[{"id":"p1","name":"gateway-01"}]}`)
	})

	group, err := client.CreateGroup(context.Background(), "tok", GroupRequest{
		Name:  "devs",
		Peers: []string{"p1"},
	})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotBody.Name != "devs" || len(gotBody.Peers) != 1 || gotBody.Peers[0] != "p1" {
		t.Errorf("unexpected request body: %+v", gotBody)
	}
	if group.ID != "g1" {
		t.Errorf("ID = %q, want g1", group.ID)
	}
}

func TestUpdateGroupSendsFullPeerSet(t *testing.T) {
	var gotMethod string
	var gotBody GroupRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"g1","name":"renamed"}`)
	})

	_, err := client.UpdateGroup(context.Background(), "tok", "g1", GroupRequest{
		Name:  "renamed",
		Peers: []string{"p1", "p2", "p3"},
	})
	if err != nil {
		t.Fatalf("UpdateGroup: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	if gotBody.Name != "renamed" {
		t.Errorf("Name = %q, want renamed", gotBody.Name)
	}
	if len(gotBody.Peers) != 3 {
		t.Errorf("Peers = %v, want all 3 sent", gotBody.Peers)
	}
}

func TestDeleteGroupIgnoresEmptyBody(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteGroup(context.Background(), "tok", "g1"); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
}

func TestDeleteGroupPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeleteGroup(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("DeleteGroup: %v", err)
	}
	if gotPath != "/api/groups/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

// The server surfaces a delete-blocked-by-reference error with a readable
// message naming what the group is linked to — this pins that it reaches the
// caller unmodified, the same contract every other endpoint's errors follow.
func TestDeleteGroupSurfacesLinkedResourceError(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":400,"message":"group has been linked to policy: Default"}`)
	})

	err := client.DeleteGroup(context.Background(), "tok", "g1")
	if err == nil {
		t.Fatal("expected an error")
	}
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("error was not *nbapi.Error: %T", err)
	}
	if apiErr.Message != "group has been linked to policy: Default" {
		t.Errorf("Message = %q", apiErr.Message)
	}
}
