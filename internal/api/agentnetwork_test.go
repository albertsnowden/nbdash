package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const oneProvider = `{
	"id":"prov1","provider_id":"openai_api","name":"OpenAI API","upstream_url":"https://api.openai.com",
	"models":[{"id":"gpt-4o","input_per_1k":0.005,"output_per_1k":0.015}],
	"identity_header_user_id":"x-bf-dim-netbird_user_id","enabled":true
}`

func TestListAgentNetworkProvidersReturnsDecodedList(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/agent-network/providers" {
			t.Errorf("path = %q, want /api/agent-network/providers", r.URL.Path)
		}
		_, _ = io.WriteString(w, "["+oneProvider+"]")
	})

	rec := httptest.NewRecorder()
	srv.listAgentNetworkProviders(rec, authed(http.MethodGet, "/api/bff/agent-network/providers", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got agentNetworkProvidersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 1 || got.Providers[0].Name != "OpenAI API" {
		t.Errorf("got = %+v, want one provider named OpenAI API", got)
	}
}

func TestCreateAgentNetworkProviderRequiresAPIKey(t *testing.T) {
	var postCalled bool
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		postCalled = true
	})

	req := authed(http.MethodPost, "/api/bff/agent-network/providers",
		strings.NewReader(`{"provider_id":"openai_api","name":"OpenAI","upstream_url":"https://api.openai.com","enabled":true}`))
	rec := httptest.NewRecorder()
	srv.createAgentNetworkProvider(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body=%s", rec.Code, rec.Body.String())
	}
	if postCalled {
		t.Error("a provider with no API key must not reach the Management API")
	}
}

// TestUpdateAgentNetworkProviderPreservesModels is the load-bearing test:
// the edit form doesn't send models/identity headers, so updateAgentNetworkProvider
// must fetch the current provider and carry them forward rather than
// letting the PUT's full-replace semantics wipe them.
func TestUpdateAgentNetworkProviderPreservesModels(t *testing.T) {
	var gotBody map[string]any
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/agent-network/providers/prov1":
			_, _ = io.WriteString(w, oneProvider)
		case r.Method == http.MethodPut && r.URL.Path == "/api/agent-network/providers/prov1":
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = io.WriteString(w, oneProvider)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	})

	req := authed(http.MethodPut, "/api/bff/agent-network/providers/prov1", strings.NewReader(
		`{"provider_id":"openai_api","name":"OpenAI API renamed","upstream_url":"https://api.openai.com","enabled":false}`))
	req.SetPathValue("id", "prov1")

	rec := httptest.NewRecorder()
	srv.updateAgentNetworkProvider(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	models, ok := gotBody["models"].([]any)
	if !ok || len(models) != 1 {
		t.Errorf("upstream models = %v, want the fetched gpt-4o entry carried forward", gotBody["models"])
	}
	if gotBody["identity_header_user_id"] != "x-bf-dim-netbird_user_id" {
		t.Errorf("upstream identity_header_user_id = %v, want carried forward", gotBody["identity_header_user_id"])
	}
}

func TestCreateAgentNetworkPolicyRequiresGroupsAndProviders(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("should never reach the Management API")
	})

	req := authed(http.MethodPost, "/api/bff/agent-network/policies",
		strings.NewReader(`{"name":"Engineering","source_groups":[],"destination_provider_ids":[]}`))
	rec := httptest.NewRecorder()
	srv.createAgentNetworkPolicy(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422, body=%s", rec.Code, rec.Body.String())
	}
}

func TestDeleteAgentNetworkPolicyReturnsNoContent(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	req := authed(http.MethodDelete, "/api/bff/agent-network/policies/pol1", nil)
	req.SetPathValue("id", "pol1")

	rec := httptest.NewRecorder()
	srv.deleteAgentNetworkPolicy(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotPath != "/api/agent-network/policies/pol1" {
		t.Errorf("path = %q, want /api/agent-network/policies/pol1", gotPath)
	}
}
