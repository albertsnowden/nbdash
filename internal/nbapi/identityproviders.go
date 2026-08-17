package nbapi

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// ListIdentityProviders returns every SSO connector configured on the
// account, ordered by name.
func (c *Client) ListIdentityProviders(ctx context.Context, token string) ([]IdentityProvider, error) {
	var providers []IdentityProvider
	if err := c.do(ctx, token, http.MethodGet, "/identity-providers", nil, nil, &providers); err != nil {
		return nil, err
	}
	sort.SliceStable(providers, func(i, j int) bool {
		return strings.ToLower(providers[i].Name) < strings.ToLower(providers[j].Name)
	})
	return providers, nil
}

// GetIdentityProvider returns a single identity provider.
func (c *Client) GetIdentityProvider(ctx context.Context, token, id string) (*IdentityProvider, error) {
	var provider IdentityProvider
	if err := c.do(ctx, token, http.MethodGet, "/identity-providers/"+url.PathEscape(id), nil, nil, &provider); err != nil {
		return nil, err
	}
	return &provider, nil
}

// CreateIdentityProvider adds a new SSO connector to the account. See
// IdentityProviderRequest for the server-side validation this can trigger,
// including a live outbound issuer-discovery check.
func (c *Client) CreateIdentityProvider(ctx context.Context, token string, req IdentityProviderRequest) (*IdentityProvider, error) {
	var provider IdentityProvider
	if err := c.do(ctx, token, http.MethodPost, "/identity-providers", nil, req, &provider); err != nil {
		return nil, err
	}
	return &provider, nil
}

// UpdateIdentityProvider writes changes to an existing identity provider. A
// blank req.ClientSecret preserves the currently stored secret rather than
// clearing it — see IdentityProviderRequest.
func (c *Client) UpdateIdentityProvider(ctx context.Context, token, id string, req IdentityProviderRequest) (*IdentityProvider, error) {
	var provider IdentityProvider
	if err := c.do(ctx, token, http.MethodPut, "/identity-providers/"+url.PathEscape(id), nil, req, &provider); err != nil {
		return nil, err
	}
	return &provider, nil
}

// DeleteIdentityProvider removes an identity provider permanently.
func (c *Client) DeleteIdentityProvider(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/identity-providers/"+url.PathEscape(id), nil, nil, nil)
}
