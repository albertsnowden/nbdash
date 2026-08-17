// Policies (Access Control) manages a policy's own fields — name,
// description, enabled — and its rules as sub-resources, rather than one
// dynamic multi-rule form. A policy's PUT replaces its whole rule set
// unconditionally (see nbapi.PolicyRequest), so any write that isn't
// "replace every rule on purpose" has to fetch the current policy first and
// carry its other rules forward unchanged — the same "fetch, mutate one
// thing, resend everything" shape internal/handlers/policies.go documents,
// preserved here even though the HTML-specific plumbing around it isn't.
//
// Every rule this package hands to the frontend carries an "id" that is
// really just its 0-based position in Policy.Rules, not nbapi.PolicyRule's
// own ID — see rewriteRuleIndexIDs' doc comment for why: the real netbird
// server does not assign a rule its own unique ID once a policy has more
// than one, so nbapi.PolicyRule.ID cannot be trusted to tell two rules on
// the same policy apart. updateRule/deleteRule parse that index back out of
// the URL and use it to pick the right array element directly, then send
// netbird whichever real (possibly shared) ID that element actually has —
// this package's own "which rule" bookkeeping never leaks into the request
// netbird itself sees.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/albertsnowden/nbdash/internal/nbapi"
)

type policiesResponse struct {
	Policies []nbapi.Policy `json:"policies"`
	Total    int            `json:"total"`
	Enabled  int            `json:"enabled"`
}

func (s *Server) listPolicies(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	policies, err := s.api.ListPolicies(r.Context(), token)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	resp := policiesResponse{Policies: policies, Total: len(policies)}
	for i := range policies {
		if policies[i].Enabled {
			resp.Enabled++
		}
		rewriteRuleIndexIDs(&policies[i])
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *Server) getPolicy(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	policy, err := s.api.GetPolicy(r.Context(), token, r.PathValue("id"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	rewriteRuleIndexIDs(policy)
	s.writeJSON(w, http.StatusOK, policy)
}

type policyRequest struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Enabled             bool     `json:"enabled"`
	SourcePostureChecks []string `json:"source_posture_checks"`
}

type ruleRequest struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	Enabled       bool     `json:"enabled"`
	Action        string   `json:"action"`
	Bidirectional bool     `json:"bidirectional"`
	Protocol      string   `json:"protocol"`
	Sources       []string `json:"sources"`
	Destinations  []string `json:"destinations"`
	Ports         []string `json:"ports"`
}

// validateRule mirrors internal/handlers/policies.go's parseRuleForm, field
// for field, just reading a decoded ruleRequest instead of r.PostFormValue.
func validateRule(req ruleRequest) (nbapi.PolicyRuleRequest, string) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nbapi.PolicyRuleRequest{}, "Rule name cannot be empty."
	}

	if req.Action != "accept" && req.Action != "drop" {
		return nbapi.PolicyRuleRequest{}, "Action must be accept or drop."
	}

	switch req.Protocol {
	case "all", "tcp", "udp", "icmp":
	default:
		return nbapi.PolicyRuleRequest{}, "Protocol must be all, tcp, udp or icmp."
	}

	if len(req.Sources) == 0 || len(req.Destinations) == 0 {
		return nbapi.PolicyRuleRequest{}, "Select at least one source group and one destination group."
	}

	for _, p := range req.Ports {
		n, err := strconv.Atoi(p)
		if err != nil || n < 1 || n > 65535 {
			return nbapi.PolicyRuleRequest{}, "Ports must be numbers between 1 and 65535."
		}
	}
	// Matches the server's own check: ALL and ICMP carry no port concept.
	if len(req.Ports) > 0 && (req.Protocol == "all" || req.Protocol == "icmp") {
		return nbapi.PolicyRuleRequest{}, "Ports aren't allowed for the All or ICMP protocol."
	}

	return nbapi.PolicyRuleRequest{
		Name:          name,
		Description:   strings.TrimSpace(req.Description),
		Enabled:       req.Enabled,
		Action:        req.Action,
		Bidirectional: req.Bidirectional,
		Protocol:      req.Protocol,
		Sources:       req.Sources,
		Destinations:  req.Destinations,
		Ports:         req.Ports,
	}, ""
}

type createPolicyRequest struct {
	Name                string      `json:"name"`
	Description         string      `json:"description"`
	Enabled             bool        `json:"enabled"`
	Rule                ruleRequest `json:"rule"`
	SourcePostureChecks []string    `json:"source_posture_checks"`
}

// createPolicy requires the one rule its form collects — the server
// requires at least one rule to create a policy at all, so unlike Groups
// (which can start with no peers) there is no meaningful "policy with no
// rules yet" state to create into.
func (s *Server) createPolicy(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	var req createPolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Name cannot be empty.")
		return
	}
	rule, msg := validateRule(req.Rule)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	created, err := s.api.CreatePolicy(r.Context(), token, nbapi.PolicyRequest{
		Name:                name,
		Description:         strings.TrimSpace(req.Description),
		Enabled:             req.Enabled,
		Rules:               []nbapi.PolicyRuleRequest{rule},
		SourcePostureChecks: req.SourcePostureChecks,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	rewriteRuleIndexIDs(created)
	s.writeJSON(w, http.StatusCreated, created)
}

// updatePolicyMeta changes name, description and enabled — never Rules,
// which is why the write's Rules come from a fresh GetPolicy rather than
// anything the request body carries: there is no rule field on it to omit
// by accident, but PolicyRequest still demands a complete rule list on
// every write, omitted or not.
func (s *Server) updatePolicyMeta(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")

	var req policyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		s.writeError(w, http.StatusUnprocessableEntity, "Name cannot be empty.")
		return
	}

	current, err := s.api.GetPolicy(r.Context(), token, id)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	updated, err := s.api.UpdatePolicy(r.Context(), token, id, nbapi.PolicyRequest{
		Name:                name,
		Description:         strings.TrimSpace(req.Description),
		Enabled:             req.Enabled,
		Rules:               rulesToRequests(current.Rules),
		SourcePostureChecks: req.SourcePostureChecks,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	rewriteRuleIndexIDs(updated)
	s.writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deletePolicy(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}

	if err := s.api.DeletePolicy(r.Context(), token, r.PathValue("id")); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// createRule appends a new rule to the policy's current rule set and PUTs
// the whole thing back, returning the updated policy so the frontend can
// pick up the new rule's index-based id (see rewriteRuleIndexIDs) without a
// second round trip.
func (s *Server) createRule(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	policyID := r.PathValue("id")

	var req ruleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	rule, msg := validateRule(req)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	current, err := s.api.GetPolicy(r.Context(), token, policyID)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	updated, err := s.api.UpdatePolicy(r.Context(), token, policyID, nbapi.PolicyRequest{
		Name:                current.Name,
		Description:         current.Description,
		Enabled:             current.Enabled,
		Rules:               append(rulesToRequests(current.Rules), rule),
		SourcePostureChecks: current.SourcePostureChecks,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	rewriteRuleIndexIDs(updated)
	s.writeJSON(w, http.StatusCreated, updated)
}

// ruleIndex parses ruleID (as rewriteRuleIndexIDs hands it to the frontend)
// back into a position in rules, bounds-checked.
func ruleIndex(ruleID string, rules []nbapi.PolicyRule) (int, bool) {
	idx, err := strconv.Atoi(ruleID)
	if err != nil || idx < 0 || idx >= len(rules) {
		return 0, false
	}
	return idx, true
}

// updateRule replaces one rule, by its index in the policy's current rule
// set (see rewriteRuleIndexIDs — ruleID is a position, not nbapi's own rule
// ID), and PUTs the whole thing back. Every other rule is carried forward
// exactly as GetPolicy returned it.
func (s *Server) updateRule(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	policyID, ruleID := r.PathValue("id"), r.PathValue("ruleID")

	var req ruleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	rule, msg := validateRule(req)
	if msg != "" {
		s.writeError(w, http.StatusUnprocessableEntity, msg)
		return
	}

	current, err := s.api.GetPolicy(r.Context(), token, policyID)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	idx, ok := ruleIndex(ruleID, current.Rules)
	if !ok {
		s.writeError(w, http.StatusNotFound, "rule not found")
		return
	}
	// Netbird's own ID for this position may be shared with other rules
	// (see rewriteRuleIndexIDs) — that's fine, it only needs to identify
	// *a* rule that already exists so the PUT isn't rejected as unknown; it
	// never has to be unique to this position on its own.
	rule.ID = current.Rules[idx].ID

	newRules := rulesToRequests(current.Rules)
	newRules[idx] = rule

	updated, err := s.api.UpdatePolicy(r.Context(), token, policyID, nbapi.PolicyRequest{
		Name:                current.Name,
		Description:         current.Description,
		Enabled:             current.Enabled,
		Rules:               newRules,
		SourcePostureChecks: current.SourcePostureChecks,
	})
	if err != nil {
		s.fail(w, r, err)
		return
	}
	rewriteRuleIndexIDs(updated)
	s.writeJSON(w, http.StatusOK, updated)
}

// deleteRule removes one rule, by its index in the policy's current rule set
// (see rewriteRuleIndexIDs — ruleID is a position, not nbapi's own rule ID),
// and PUTs the remainder back. Refused when it's the policy's last rule —
// the server requires at least one, and "policy rules shouldn't be empty"
// is a confusing message for an action that looks like "delete a rule," not
// "empty out a whole policy."
func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) {
	token, ok := s.token(w, r)
	if !ok {
		return
	}
	policyID, ruleID := r.PathValue("id"), r.PathValue("ruleID")

	current, err := s.api.GetPolicy(r.Context(), token, policyID)
	if err != nil {
		s.fail(w, r, err)
		return
	}

	if len(current.Rules) <= 1 {
		s.writeError(w, http.StatusUnprocessableEntity,
			"A policy needs at least one rule — delete the policy instead, or add a replacement rule first.")
		return
	}

	idx, ok := ruleIndex(ruleID, current.Rules)
	if !ok {
		s.writeError(w, http.StatusNotFound, "rule not found")
		return
	}
	remaining := rulesToRequests(current.Rules)
	remaining = append(remaining[:idx], remaining[idx+1:]...)

	if _, err := s.api.UpdatePolicy(r.Context(), token, policyID, nbapi.PolicyRequest{
		Name:                current.Name,
		Description:         current.Description,
		Enabled:             current.Enabled,
		Rules:               remaining,
		SourcePostureChecks: current.SourcePostureChecks,
	}); err != nil {
		s.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// policyRuleToRequest converts a rule as GetPolicy returns it (Sources and
// Destinations resolved to GroupMinimum) back into the write shape a PUT
// needs (bare group IDs) — how every "other" rule gets carried forward
// unchanged whenever only one rule, or only the policy's own metadata, is
// what's actually being edited.
func policyRuleToRequest(rule nbapi.PolicyRule) nbapi.PolicyRuleRequest {
	return nbapi.PolicyRuleRequest{
		ID:            rule.ID,
		Name:          rule.Name,
		Description:   rule.Description,
		Enabled:       rule.Enabled,
		Action:        rule.Action,
		Bidirectional: rule.Bidirectional,
		Protocol:      rule.Protocol,
		Sources:       groupIDs(rule.Sources),
		Destinations:  groupIDs(rule.Destinations),
		Ports:         append([]string(nil), rule.Ports...),
	}
}

func rulesToRequests(rules []nbapi.PolicyRule) []nbapi.PolicyRuleRequest {
	out := make([]nbapi.PolicyRuleRequest, len(rules))
	for i, rule := range rules {
		out[i] = policyRuleToRequest(rule)
	}
	return out
}

// rewriteRuleIndexIDs overwrites each of policy.Rules' IDs with its own
// index, in place. The real netbird server only assigns a rule its own ID
// when that ID is left empty on a *single*-rule policy; the moment a second
// rule gets added, the server falls back to reusing the whole policy's ID
// for any rule it wasn't given an ID for (management/server/policy.go's
// validatePolicy: "ruleCopy.ID = policy.ID // TODO: when policy can contain
// multiple rules, need refactor" — an acknowledged, unfixed upstream gap,
// not something this dashboard can ask the server to do differently). Left
// alone, every rule added after the first collides on the same ID, which
// made updateRule/deleteRule's ID-based lookup silently touch every
// colliding rule instead of just the one the user picked — edit one rule
// and every rule sharing its ID gets overwritten with the same content;
// delete one and all of them vanish together. Call this on every response
// that carries rules, right before writeJSON, so the frontend never sees a
// netbird-assigned ID at all.
func rewriteRuleIndexIDs(policy *nbapi.Policy) {
	for i := range policy.Rules {
		policy.Rules[i].ID = strconv.Itoa(i)
	}
}

// groupIDs extracts the id from each of a rule's resolved source/destination
// groups.
func groupIDs(groups []nbapi.GroupMinimum) []string {
	ids := make([]string, len(groups))
	for i, g := range groups {
		ids[i] = g.ID
	}
	return ids
}
