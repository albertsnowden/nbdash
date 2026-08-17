package nbapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// Account is the caller's own account — the API "always returns a list of
// one account" per its own doc comment, so ListAccounts below collapses
// that to a single value rather than making every caller unwrap a
// single-element slice. Settings is kept as raw JSON: AccountSettings has
// 20+ fields (peer expiration, DNS domain, network ranges, dashboard
// feature flags, ...) this dashboard doesn't manage, and a PUT is a full
// replace of the whole settings object — see UpdateAccountSettings for how
// the JWT Group Sync page patches just its three fields without touching
// the rest.
type Account struct {
	ID        string          `json:"id"`
	Domain    string          `json:"domain,omitempty"`
	CreatedAt string          `json:"created_at,omitempty"`
	CreatedBy string          `json:"created_by,omitempty"`
	Settings  json.RawMessage `json:"settings,omitempty"`
}

// UpdateAccountSettings sends a full-replace PUT of the account's settings
// object. Callers are responsible for having fetched the current settings
// and merged their change into it first (see CurrentAccount).
func (c *Client) UpdateAccountSettings(ctx context.Context, token, id string, settings json.RawMessage) (*Account, error) {
	var account Account
	req := struct {
		Settings json.RawMessage `json:"settings"`
	}{Settings: settings}
	if err := c.do(ctx, token, http.MethodPut, "/accounts/"+url.PathEscape(id), nil, req, &account); err != nil {
		return nil, err
	}
	return &account, nil
}

// CurrentAccount returns the caller's account.
func (c *Client) CurrentAccount(ctx context.Context, token string) (*Account, error) {
	var accounts []Account
	if err := c.do(ctx, token, http.MethodGet, "/accounts", nil, nil, &accounts); err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return &Account{}, nil
	}
	return &accounts[0], nil
}

// DeleteAccount permanently deletes the account and every resource in it.
// Only the account owner may call this — the Management API itself
// enforces that, this is not re-checked here.
func (c *Client) DeleteAccount(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/accounts/"+url.PathEscape(id), nil, nil, nil)
}
