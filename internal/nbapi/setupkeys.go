package nbapi

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// ListSetupKeys returns every setup key on the account.
func (c *Client) ListSetupKeys(ctx context.Context, token string) ([]SetupKey, error) {
	var keys []SetupKey
	if err := c.do(ctx, token, http.MethodGet, "/setup-keys", nil, nil, &keys); err != nil {
		return nil, err
	}
	sortSetupKeys(keys)
	return keys, nil
}

// GetSetupKey returns a single setup key, with its key field masked — only
// CreateSetupKey's response ever carries the plaintext.
func (c *Client) GetSetupKey(ctx context.Context, token, id string) (*SetupKey, error) {
	var key SetupKey
	if err := c.do(ctx, token, http.MethodGet, "/setup-keys/"+url.PathEscape(id), nil, nil, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// CreateSetupKey creates a setup key and returns it with the plaintext key —
// the only response from this API that ever contains it. Callers must
// display it immediately and must not persist or log it: there is no way to
// retrieve it again once this call returns.
func (c *Client) CreateSetupKey(ctx context.Context, token string, req CreateSetupKeyRequest) (*SetupKey, error) {
	var key SetupKey
	if err := c.do(ctx, token, http.MethodPost, "/setup-keys", nil, req, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// UpdateSetupKey writes the key's revoked status and auto-groups and returns
// the stored result — with the key masked, as on every endpoint but create.
// req must carry the key's current AutoGroups for anything it is not
// changing; see SetupKeyRequest.
func (c *Client) UpdateSetupKey(ctx context.Context, token, id string, req SetupKeyRequest) (*SetupKey, error) {
	var key SetupKey
	if err := c.do(ctx, token, http.MethodPut, "/setup-keys/"+url.PathEscape(id), nil, req, &key); err != nil {
		return nil, err
	}
	return &key, nil
}

// DeleteSetupKey removes a setup key permanently.
func (c *Client) DeleteSetupKey(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/setup-keys/"+url.PathEscape(id), nil, nil, nil)
}

// sortSetupKeys puts valid keys first, then orders by name — the keys an
// operator is most likely to hand out or check on sit at the top, the same
// reasoning sortPeers uses for connected peers.
func sortSetupKeys(keys []SetupKey) {
	sort.SliceStable(keys, func(i, j int) bool {
		if keys[i].Valid != keys[j].Valid {
			return keys[i].Valid
		}
		return strings.ToLower(keys[i].Name) < strings.ToLower(keys[j].Name)
	})
}
