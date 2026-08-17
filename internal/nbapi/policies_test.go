package nbapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestListPoliciesSendsBearerTokenToAPIPath(t *testing.T) {
	var gotPath, gotAuth string

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, _ = io.WriteString(w, `[]`)
	})

	if _, err := client.ListPolicies(context.Background(), "tok-123"); err != nil {
		t.Fatalf("ListPolicies: %v", err)
	}

	if gotPath != "/api/policies" {
		t.Errorf("path = %q, want /api/policies", gotPath)
	}
	if gotAuth != "Bearer tok-123" {
		t.Errorf("Authorization = %q, want Bearer tok-123", gotAuth)
	}
}

func TestListPoliciesSortsByName(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `[{"id":"1","name":"zulu"},{"id":"2","name":"Default"}]`)
	})

	policies, err := client.ListPolicies(context.Background(), "tok")
	if err != nil {
		t.Fatalf("ListPolicies: %v", err)
	}
	if len(policies) != 2 || policies[0].Name != "Default" || policies[1].Name != "zulu" {
		t.Errorf("order = %v, want [Default zulu]", policies)
	}
}

func TestGetPolicyReturnsResolvedGroups(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"id":"p1","name":"Default","rules":[
			{"id":"r1","name":"Default","action":"accept","protocol":"all","bidirectional":true,
			 "sources":[{"id":"g1","name":"All"}],"destinations":[{"id":"g1","name":"All"}]}
		]}`)
	})

	policy, err := client.GetPolicy(context.Background(), "tok", "p1")
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	if len(policy.Rules) != 1 {
		t.Fatalf("Rules = %v, want 1", policy.Rules)
	}
	rule := policy.Rules[0]
	if len(rule.Sources) != 1 || rule.Sources[0].Name != "All" {
		t.Errorf("Sources = %+v, want resolved [All]", rule.Sources)
	}
}

func TestGetPolicyPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = io.WriteString(w, `{}`)
	})

	if _, err := client.GetPolicy(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	if gotPath != "/api/policies/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

func TestCreatePolicySendsNameAndRules(t *testing.T) {
	var gotMethod string
	var gotBody PolicyRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"p1","name":"allow-web"}`)
	})

	_, err := client.CreatePolicy(context.Background(), "tok", PolicyRequest{
		Name:    "allow-web",
		Enabled: true,
		Rules: []PolicyRuleRequest{
			{Name: "web", Action: "accept", Protocol: "tcp", Sources: []string{"g1"}, Destinations: []string{"g2"}},
		},
	})
	if err != nil {
		t.Fatalf("CreatePolicy: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotBody.Name != "allow-web" || len(gotBody.Rules) != 1 {
		t.Errorf("unexpected request body: %+v", gotBody)
	}
	if gotBody.Rules[0].Sources[0] != "g1" {
		t.Errorf("Rules[0].Sources = %v, want [g1]", gotBody.Rules[0].Sources)
	}
}

func TestUpdatePolicySendsFullRuleSet(t *testing.T) {
	var gotMethod string
	var gotBody PolicyRequest

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, `{"id":"p1","name":"renamed"}`)
	})

	_, err := client.UpdatePolicy(context.Background(), "tok", "p1", PolicyRequest{
		Name: "renamed",
		Rules: []PolicyRuleRequest{
			{ID: "r1", Name: "rule-a", Action: "accept", Protocol: "all"},
			{ID: "r2", Name: "rule-b", Action: "drop", Protocol: "tcp"},
		},
	})
	if err != nil {
		t.Fatalf("UpdatePolicy: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %s, want PUT", gotMethod)
	}
	if len(gotBody.Rules) != 2 {
		t.Errorf("Rules = %v, want both sent", gotBody.Rules)
	}
}

func TestDeletePolicyIgnoresEmptyBody(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeletePolicy(context.Background(), "tok", "p1"); err != nil {
		t.Fatalf("DeletePolicy: %v", err)
	}
}

func TestDeletePolicyPathIsEscaped(t *testing.T) {
	var gotPath string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.WriteHeader(http.StatusOK)
	})

	if err := client.DeletePolicy(context.Background(), "tok", "a/../b"); err != nil {
		t.Fatalf("DeletePolicy: %v", err)
	}
	if gotPath != "/api/policies/a%2F..%2Fb" {
		t.Errorf("path = %q, want the id escaped", gotPath)
	}
}

func TestCreatePolicyRejectingEmptyRulesSurfacesServerMessage(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"code":400,"message":"policy rules shouldn't be empty"}`)
	})

	_, err := client.CreatePolicy(context.Background(), "tok", PolicyRequest{Name: "empty"})
	if err == nil {
		t.Fatal("expected an error")
	}
	var apiErr *Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("error was not *nbapi.Error: %T", err)
	}
	if apiErr.Message != "policy rules shouldn't be empty" {
		t.Errorf("Message = %q", apiErr.Message)
	}
}
