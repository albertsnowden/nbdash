package nbapi

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// ListUsers returns the account's users. serviceUser filters the result:
// false for regular users, true for service users, nil for both — this
// dashboard always passes an explicit filter to keep the Users and Service
// Users tabs from ever mixing.
func (c *Client) ListUsers(ctx context.Context, token string, serviceUser *bool) ([]User, error) {
	var params url.Values
	if serviceUser != nil {
		params = url.Values{"service_user": {strconv.FormatBool(*serviceUser)}}
	}

	var users []User
	if err := c.do(ctx, token, http.MethodGet, "/users", params, nil, &users); err != nil {
		return nil, err
	}
	sortUsers(users)
	return users, nil
}

// CurrentUser returns the signed-in user, including the Permissions block
// that drives the Permissions/RBAC settings view — the server resolves
// role-to-capability itself, so the dashboard never hardcodes what each
// role can do.
func (c *Client) CurrentUser(ctx context.Context, token string) (*User, error) {
	var user User
	if err := c.do(ctx, token, http.MethodGet, "/users/current", nil, nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser either invites a regular user through the configured IdP or
// creates a service user directly, decided by req.IsServiceUser — see
// UserCreateRequest's doc comment.
func (c *Client) CreateUser(ctx context.Context, token string, req UserCreateRequest) (*User, error) {
	var user User
	if err := c.do(ctx, token, http.MethodPost, "/users", nil, req, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser replaces a user's role, auto_groups and blocked state — see
// UserRequest's doc comment for the server-side restrictions on what value
// each may hold and who may set them.
func (c *Client) UpdateUser(ctx context.Context, token, id string, req UserRequest) (*User, error) {
	var user User
	if err := c.do(ctx, token, http.MethodPut, "/users/"+url.PathEscape(id), nil, req, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// DeleteUser removes a user. The server refuses to delete the account owner
// or to let a user delete themselves; both surface as a normal API error
// this call returns unchanged.
func (c *Client) DeleteUser(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/users/"+url.PathEscape(id), nil, nil, nil)
}

// InviteUser resends the invite email to a user who has not yet activated
// their account.
func (c *Client) InviteUser(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodPost, "/users/"+url.PathEscape(id)+"/invite", nil, nil, nil)
}

// ApproveUser activates a user who joined via domain matching and is
// pending_approval.
func (c *Client) ApproveUser(ctx context.Context, token, id string) (*User, error) {
	var user User
	if err := c.do(ctx, token, http.MethodPost, "/users/"+url.PathEscape(id)+"/approve", nil, nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// RejectUser removes a pending_approval user from the account entirely,
// rather than merely leaving them unapproved.
func (c *Client) RejectUser(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/users/"+url.PathEscape(id)+"/reject", nil, nil, nil)
}

func sortUsers(users []User) {
	sort.SliceStable(users, func(i, j int) bool {
		return strings.ToLower(users[i].Name) < strings.ToLower(users[j].Name)
	})
}

// ListPATs returns every personal access token belonging to the given user,
// masked — the plaintext never appears here, only in CreatePAT's response.
func (c *Client) ListPATs(ctx context.Context, token, userID string) ([]PersonalAccessToken, error) {
	var pats []PersonalAccessToken
	path := "/users/" + url.PathEscape(userID) + "/tokens"
	if err := c.do(ctx, token, http.MethodGet, path, nil, nil, &pats); err != nil {
		return nil, err
	}
	return pats, nil
}

// CreatePAT creates a personal access token for the given user. The
// response is the only place its plaintext ever exists in this codebase —
// see PersonalAccessTokenGenerated's doc comment.
func (c *Client) CreatePAT(ctx context.Context, token, userID string, req PersonalAccessTokenRequest) (*PersonalAccessTokenGenerated, error) {
	var generated PersonalAccessTokenGenerated
	path := "/users/" + url.PathEscape(userID) + "/tokens"
	if err := c.do(ctx, token, http.MethodPost, path, nil, req, &generated); err != nil {
		return nil, err
	}
	return &generated, nil
}

// DeletePAT removes a personal access token, immediately invalidating it.
func (c *Client) DeletePAT(ctx context.Context, token, userID, tokenID string) error {
	path := "/users/" + url.PathEscape(userID) + "/tokens/" + url.PathEscape(tokenID)
	return c.do(ctx, token, http.MethodDelete, path, nil, nil, nil)
}
