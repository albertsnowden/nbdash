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

const onePolicyRule = `{"id":"r1","name":"allow-ssh","action":"accept","protocol":"tcp",
	"sources":[{"id":"g1","name":"devs"}],"destinations":[{"id":"g2","name":"servers"}],"ports":["22"]}`

const twoPolicies = `[
	{"id":"p1","name":"default","enabled":true,"rules":[` + onePolicyRule + `]},
	{"id":"p2","name":"disabled-one","enabled":false,"rules":[` + onePolicyRule + `]}
]`

const onePolicyWithOneRule = `{"id":"p1","name":"default","enabled":true,"rules":[` + onePolicyRule + `]}`

const onePolicyWithPostureCheck = `{"id":"p1","name":"default","enabled":true,"rules":[` + onePolicyRule +
	`],"source_posture_checks":["pc1"]}`

const twoRuleTemplate = `{"id":"p1","name":"default","enabled":true,"rules":[` + onePolicyRule + `,
	{"id":"r2","name":"second","action":"drop","protocol":"all","sources":[{"id":"g1","name":"devs"}],"destinations":[{"id":"g2","name":"servers"}]}]}`

func validRuleBody() string {
	return `{"name":"allow-ssh","action":"accept","protocol":"tcp","sources":["g1"],"destinations":["g2"],"ports":["22"]}`
}

func TestListPoliciesReturnsCounts(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, twoPolicies)
	})

	rec := httptest.NewRecorder()
	srv.listPolicies(rec, authed(http.MethodGet, "/api/bff/policies", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got policiesResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Total != 2 || got.Enabled != 1 {
		t.Errorf("Total/Enabled = %d/%d, want 2/1", got.Total, got.Enabled)
	}
}

func TestCreatePolicyRequiresValidRule(t *testing.T) {
	var posted bool
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		posted = true
	})

	body := `{"name":"new-policy","enabled":true,"rule":{"name":"","action":"accept","protocol":"tcp","sources":["g1"],"destinations":["g2"]}}`
	rec := httptest.NewRecorder()
	srv.createPolicy(rec, authed(http.MethodPost, "/api/bff/policies", strings.NewReader(body)))

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
	if posted {
		t.Error("an invalid rule must not reach the Management API")
	}
}

func TestCreatePolicySendsOneRule(t *testing.T) {
	var gotBody nbapi.PolicyRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, onePolicyWithOneRule)
	})

	body := `{"name":"default","enabled":true,"rule":` + validRuleBody() + `}`
	rec := httptest.NewRecorder()
	srv.createPolicy(rec, authed(http.MethodPost, "/api/bff/policies", strings.NewReader(body)))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	if len(gotBody.Rules) != 1 || gotBody.Rules[0].Name != "allow-ssh" {
		t.Errorf("unexpected upstream rules: %+v", gotBody.Rules)
	}
}

func TestCreatePolicySendsSourcePostureChecks(t *testing.T) {
	var gotBody nbapi.PolicyRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_, _ = io.WriteString(w, onePolicyWithPostureCheck)
	})

	body := `{"name":"default","enabled":true,"rule":` + validRuleBody() + `,"source_posture_checks":["pc1"]}`
	rec := httptest.NewRecorder()
	srv.createPolicy(rec, authed(http.MethodPost, "/api/bff/policies", strings.NewReader(body)))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	if len(gotBody.SourcePostureChecks) != 1 || gotBody.SourcePostureChecks[0] != "pc1" {
		t.Errorf("upstream source_posture_checks = %v, want [pc1]", gotBody.SourcePostureChecks)
	}
}

// TestCreateRulePreservesSourcePostureChecks: adding a rule must not detach
// a policy's posture checks — createRule/updateRule/deleteRule all carry
// GetPolicy's SourcePostureChecks forward, same discipline as Rules.
func TestCreateRulePreservesSourcePostureChecks(t *testing.T) {
	var gotBody nbapi.PolicyRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, onePolicyWithPostureCheck)
		case http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = io.WriteString(w, onePolicyWithPostureCheck)
		}
	})

	req := authed(http.MethodPost, "/api/bff/policies/p1/rules", strings.NewReader(validRuleBody()))
	req.SetPathValue("id", "p1")

	rec := httptest.NewRecorder()
	srv.createRule(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	if len(gotBody.SourcePostureChecks) != 1 || gotBody.SourcePostureChecks[0] != "pc1" {
		t.Errorf("upstream source_posture_checks = %v, want [pc1] carried forward", gotBody.SourcePostureChecks)
	}
}

func TestUpdatePolicyMetaPreservesRules(t *testing.T) {
	var getCalled, putCalled bool
	var gotBody nbapi.PolicyRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCalled = true
			_, _ = io.WriteString(w, onePolicyWithOneRule)
		case http.MethodPut:
			putCalled = true
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = io.WriteString(w, `{"id":"p1","name":"renamed","enabled":false,"rules":[`+onePolicyRule+`]}`)
		}
	})

	req := authed(http.MethodPut, "/api/bff/policies/p1", strings.NewReader(`{"name":"renamed","enabled":false}`))
	req.SetPathValue("id", "p1")

	rec := httptest.NewRecorder()
	srv.updatePolicyMeta(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if !getCalled || !putCalled {
		t.Fatalf("expected both a GET (fetch current rules) and a PUT, got get=%v put=%v", getCalled, putCalled)
	}
	if len(gotBody.Rules) != 1 {
		t.Errorf("Rules = %+v, want the existing rule carried forward unchanged", gotBody.Rules)
	}
}

func TestCreateRuleAppendsToExisting(t *testing.T) {
	var gotBody nbapi.PolicyRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, onePolicyWithOneRule)
		case http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = io.WriteString(w, twoRuleTemplate)
		}
	})

	req := authed(http.MethodPost, "/api/bff/policies/p1/rules", strings.NewReader(
		`{"name":"second","action":"drop","protocol":"all","sources":["g1"],"destinations":["g2"]}`))
	req.SetPathValue("id", "p1")

	rec := httptest.NewRecorder()
	srv.createRule(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	if len(gotBody.Rules) != 2 {
		t.Fatalf("Rules sent = %d, want 2 (existing + new)", len(gotBody.Rules))
	}
	if gotBody.Rules[0].Name != "allow-ssh" || gotBody.Rules[1].Name != "second" {
		t.Errorf("unexpected rule order/content: %+v", gotBody.Rules)
	}
}

func TestUpdateRuleReplacesOnlyTargetRule(t *testing.T) {
	var gotBody nbapi.PolicyRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, twoRuleTemplate)
		case http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = io.WriteString(w, twoRuleTemplate)
		}
	})

	// "1" addresses this rule by its position in the policy's rule list, not
	// nbapi's own rule ID — see rewriteRuleIndexIDs' doc comment in
	// policies.go for why the real netbird API can't be trusted to keep a
	// rule's own ID unique once a policy has more than one rule.
	req := authed(http.MethodPut, "/api/bff/policies/p1/rules/1", strings.NewReader(
		`{"name":"renamed-rule","action":"accept","protocol":"udp","sources":["g1"],"destinations":["g2"]}`))
	req.SetPathValue("id", "p1")
	req.SetPathValue("ruleID", "1")

	rec := httptest.NewRecorder()
	srv.updateRule(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if len(gotBody.Rules) != 2 {
		t.Fatalf("Rules sent = %d, want 2", len(gotBody.Rules))
	}
	// r1 carried forward unchanged, r2 replaced with the new content.
	if gotBody.Rules[0].ID != "r1" || gotBody.Rules[0].Name != "allow-ssh" {
		t.Errorf("rule 1 should be unchanged: %+v", gotBody.Rules[0])
	}
	if gotBody.Rules[1].ID != "r2" || gotBody.Rules[1].Name != "renamed-rule" {
		t.Errorf("rule 2 should be replaced: %+v", gotBody.Rules[1])
	}
}

func TestUpdateRuleRejectsUnknownRuleID(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, onePolicyWithOneRule)
		} else {
			t.Error("a rule ID that doesn't exist in the policy must not reach the Management API")
		}
	})

	req := authed(http.MethodPut, "/api/bff/policies/p1/rules/does-not-exist", strings.NewReader(validRuleBody()))
	req.SetPathValue("id", "p1")
	req.SetPathValue("ruleID", "does-not-exist")

	rec := httptest.NewRecorder()
	srv.updateRule(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestDeleteRuleRefusesLastRule(t *testing.T) {
	var putCalled bool
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, onePolicyWithOneRule)
		case http.MethodPut:
			putCalled = true
		}
	})

	req := authed(http.MethodDelete, "/api/bff/policies/p1/rules/0", nil)
	req.SetPathValue("id", "p1")
	req.SetPathValue("ruleID", "0")

	rec := httptest.NewRecorder()
	srv.deleteRule(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
	if putCalled {
		t.Error("deleting a policy's only rule must not reach the Management API")
	}
}

func TestDeleteRuleRemovesJustThatRule(t *testing.T) {
	var gotBody nbapi.PolicyRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, twoRuleTemplate)
		case http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = io.WriteString(w, onePolicyWithOneRule)
		}
	})

	req := authed(http.MethodDelete, "/api/bff/policies/p1/rules/1", nil)
	req.SetPathValue("id", "p1")
	req.SetPathValue("ruleID", "1")

	rec := httptest.NewRecorder()
	srv.deleteRule(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", rec.Code, rec.Body.String())
	}
	if len(gotBody.Rules) != 1 || gotBody.Rules[0].ID != "r1" {
		t.Errorf("Rules sent = %+v, want just r1 remaining", gotBody.Rules)
	}
}

// sharedIDTwoRuleTemplate reproduces the real netbird bug this file's
// rewriteRuleIndexIDs works around: management/server/policy.go's
// validatePolicy assigns a rule the *policy's own* ID whenever the rule
// wasn't given one, for every rule, not just the first — so a policy with
// more than one rule routinely comes back from the real API with every
// rule sharing one ID (here "p1", matching the policy's own ID, exactly as
// a real multi-rule policy looks).
const sharedIDTwoRuleTemplate = `{"id":"p1","name":"default","enabled":true,"rules":[
	{"id":"p1","name":"first","action":"accept","protocol":"tcp","sources":[{"id":"g1","name":"devs"}],"destinations":[{"id":"g2","name":"servers"}],"ports":["22"]},
	{"id":"p1","name":"second","action":"drop","protocol":"all","sources":[{"id":"g1","name":"devs"}],"destinations":[{"id":"g2","name":"servers"}]}
]}`

// TestGetPolicyGivesDuplicateUpstreamRuleIDsUniqueIDs is the direct
// regression test for the bug this file's rewriteRuleIndexIDs fixes: two
// rules sharing nbapi's own ID (see sharedIDTwoRuleTemplate) must not reach
// the frontend still colliding, or its "find the rule with this id" lookups
// (edit, delete) can't tell them apart.
func TestGetPolicyGivesDuplicateUpstreamRuleIDsUniqueIDs(t *testing.T) {
	srv := newTestServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, sharedIDTwoRuleTemplate)
	})

	req := authed(http.MethodGet, "/api/bff/policies/p1", nil)
	req.SetPathValue("id", "p1")

	rec := httptest.NewRecorder()
	srv.getPolicy(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var got nbapi.Policy
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(got.Rules) != 2 {
		t.Fatalf("Rules = %d, want 2", len(got.Rules))
	}
	if got.Rules[0].ID == got.Rules[1].ID {
		t.Fatalf("rule IDs still collide after rewriteRuleIndexIDs: both %q", got.Rules[0].ID)
	}
	if got.Rules[0].ID != "0" || got.Rules[1].ID != "1" {
		t.Errorf("rule IDs = %q, %q, want \"0\", \"1\"", got.Rules[0].ID, got.Rules[1].ID)
	}
}

// TestDeleteRuleWithDuplicateUpstreamIDsRemovesOnlyThatPosition is the
// direct regression test for the data-loss half of the bug: before
// rewriteRuleIndexIDs/ruleIndex, deleting "the rule with ID p1" from
// sharedIDTwoRuleTemplate matched *both* rules (they share that ID) and
// would have sent an empty rule list upstream, silently deleting both
// instead of the one the user picked.
func TestDeleteRuleWithDuplicateUpstreamIDsRemovesOnlyThatPosition(t *testing.T) {
	var gotBody nbapi.PolicyRequest
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_, _ = io.WriteString(w, sharedIDTwoRuleTemplate)
		case http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
			_, _ = io.WriteString(w, onePolicyWithOneRule)
		}
	})

	// Index 1 is "second" — both rules carry upstream ID "p1", so only the
	// index disambiguates which one the user meant.
	req := authed(http.MethodDelete, "/api/bff/policies/p1/rules/1", nil)
	req.SetPathValue("id", "p1")
	req.SetPathValue("ruleID", "1")

	rec := httptest.NewRecorder()
	srv.deleteRule(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", rec.Code, rec.Body.String())
	}
	if len(gotBody.Rules) != 1 {
		t.Fatalf("Rules sent = %d, want 1 (only 'second' removed, 'first' survives)", len(gotBody.Rules))
	}
	if gotBody.Rules[0].Name != "first" {
		t.Errorf("surviving rule = %q, want %q", gotBody.Rules[0].Name, "first")
	}
}

func TestDeletePolicyReturnsNoContent(t *testing.T) {
	var gotPath string
	srv := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	})

	req := authed(http.MethodDelete, "/api/bff/policies/p1", nil)
	req.SetPathValue("id", "p1")

	rec := httptest.NewRecorder()
	srv.deletePolicy(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if gotPath != "/api/policies/p1" {
		t.Errorf("path = %q, want /api/policies/p1", gotPath)
	}
}
