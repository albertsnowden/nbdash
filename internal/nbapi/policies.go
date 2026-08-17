package nbapi

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// ListPolicies returns every access-control policy on the account.
func (c *Client) ListPolicies(ctx context.Context, token string) ([]Policy, error) {
	var policies []Policy
	if err := c.do(ctx, token, http.MethodGet, "/policies", nil, nil, &policies); err != nil {
		return nil, err
	}
	sortPolicies(policies)
	return policies, nil
}

// GetPolicy returns a single policy with its full rule set — what every
// mutation in this package re-fetches before writing, so it can carry
// whatever it isn't changing forward unchanged (see PolicyRequest).
func (c *Client) GetPolicy(ctx context.Context, token, id string) (*Policy, error) {
	var policy Policy
	if err := c.do(ctx, token, http.MethodGet, "/policies/"+url.PathEscape(id), nil, nil, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// CreatePolicy creates a policy with one or more rules. req must include at
// least one rule; the server rejects an empty rule list.
func (c *Client) CreatePolicy(ctx context.Context, token string, req PolicyRequest) (*Policy, error) {
	var policy Policy
	if err := c.do(ctx, token, http.MethodPost, "/policies", nil, req, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// UpdatePolicy replaces a policy's name, description, enabled state and whole
// rule set. req must carry every rule the policy should end up with — see
// PolicyRequest.
func (c *Client) UpdatePolicy(ctx context.Context, token, id string, req PolicyRequest) (*Policy, error) {
	var policy Policy
	if err := c.do(ctx, token, http.MethodPut, "/policies/"+url.PathEscape(id), nil, req, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

// DeletePolicy removes a policy and every rule in it. Unlike Groups' "All",
// no policy is protected from deletion server-side — not even a "Default"
// policy an account was seeded with.
func (c *Client) DeletePolicy(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/policies/"+url.PathEscape(id), nil, nil, nil)
}

func sortPolicies(policies []Policy) {
	sort.SliceStable(policies, func(i, j int) bool {
		return strings.ToLower(policies[i].Name) < strings.ToLower(policies[j].Name)
	})
}
