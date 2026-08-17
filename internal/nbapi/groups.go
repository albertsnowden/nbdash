package nbapi

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// ListGroups returns every group on the account.
func (c *Client) ListGroups(ctx context.Context, token string) ([]Group, error) {
	var groups []Group
	if err := c.do(ctx, token, http.MethodGet, "/groups", nil, nil, &groups); err != nil {
		return nil, err
	}
	sortGroups(groups)
	return groups, nil
}

// GetGroup returns a single group, including its resolved peer membership —
// what the edit form pre-populates its peer checkboxes from.
func (c *Client) GetGroup(ctx context.Context, token, id string) (*Group, error) {
	var group Group
	if err := c.do(ctx, token, http.MethodGet, "/groups/"+url.PathEscape(id), nil, nil, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

// CreateGroup creates a group with the given name and initial peer
// membership.
func (c *Client) CreateGroup(ctx context.Context, token string, req GroupRequest) (*Group, error) {
	var group Group
	if err := c.do(ctx, token, http.MethodPost, "/groups", nil, req, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

// UpdateGroup renames a group and replaces its whole peer membership — see
// GroupRequest. req must carry every peer the group should end up with, not
// just changes.
func (c *Client) UpdateGroup(ctx context.Context, token, id string, req GroupRequest) (*Group, error) {
	var group Group
	if err := c.do(ctx, token, http.MethodPut, "/groups/"+url.PathEscape(id), nil, req, &group); err != nil {
		return nil, err
	}
	return &group, nil
}

// DeleteGroup removes a group. The server rejects deleting the built-in "All"
// group (checked client-side too, so the button is never offered for it — see
// Group's doc comment) and any group still referenced by a policy, route, DNS
// zone, setup key, user or other resource; those surface as a normal API
// error this call returns unchanged, with a message naming what it's linked
// to (e.g. "group has been linked to policy: my-policy").
func (c *Client) DeleteGroup(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/groups/"+url.PathEscape(id), nil, nil, nil)
}

// sortGroups orders by name — groups have no notion of "active" the way a
// connected peer or a valid setup key does, so unlike sortPeers/sortSetupKeys
// there is no other axis to sort on first.
func sortGroups(groups []Group) {
	sort.SliceStable(groups, func(i, j int) bool {
		return strings.ToLower(groups[i].Name) < strings.ToLower(groups[j].Name)
	})
}
