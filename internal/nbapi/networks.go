package nbapi

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"
)

// ListNetworks returns every network on the account. Each entry's Routers and
// Resources are bare ID lists — GetNetwork doesn't resolve them any further
// either; the detail page fetches the full router/resource objects
// separately, through their own endpoints.
func (c *Client) ListNetworks(ctx context.Context, token string) ([]Network, error) {
	var networks []Network
	if err := c.do(ctx, token, http.MethodGet, "/networks", nil, nil, &networks); err != nil {
		return nil, err
	}
	sortNetworks(networks)
	return networks, nil
}

// GetNetwork returns a single network.
func (c *Client) GetNetwork(ctx context.Context, token, id string) (*Network, error) {
	var network Network
	if err := c.do(ctx, token, http.MethodGet, "/networks/"+url.PathEscape(id), nil, nil, &network); err != nil {
		return nil, err
	}
	return &network, nil
}

// CreateNetwork creates a network with just a name and description — routers
// and resources are added afterward, through their own endpoints.
func (c *Client) CreateNetwork(ctx context.Context, token string, req NetworkRequest) (*Network, error) {
	var network Network
	if err := c.do(ctx, token, http.MethodPost, "/networks", nil, req, &network); err != nil {
		return nil, err
	}
	return &network, nil
}

// UpdateNetwork renames or re-describes a network. Unlike UpdateGroup or
// UpdatePolicy, there is no membership or rule set to carry forward here —
// NetworkRequest has no other fields.
func (c *Client) UpdateNetwork(ctx context.Context, token, id string, req NetworkRequest) (*Network, error) {
	var network Network
	if err := c.do(ctx, token, http.MethodPut, "/networks/"+url.PathEscape(id), nil, req, &network); err != nil {
		return nil, err
	}
	return &network, nil
}

// DeleteNetwork removes a network and, server-side, every resource and
// router in it — DeleteNetwork in management/server/networks cascades rather
// than requiring them to be deleted first.
func (c *Client) DeleteNetwork(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/networks/"+url.PathEscape(id), nil, nil, nil)
}

func sortNetworks(networks []Network) {
	sort.SliceStable(networks, func(i, j int) bool {
		return strings.ToLower(networks[i].Name) < strings.ToLower(networks[j].Name)
	})
}

// ListNetworkResources returns every resource in the given network.
func (c *Client) ListNetworkResources(ctx context.Context, token, networkID string) ([]NetworkResource, error) {
	var resources []NetworkResource
	path := "/networks/" + url.PathEscape(networkID) + "/resources"
	if err := c.do(ctx, token, http.MethodGet, path, nil, nil, &resources); err != nil {
		return nil, err
	}
	sortResources(resources)
	return resources, nil
}

// GetNetworkResource returns a single resource.
func (c *Client) GetNetworkResource(ctx context.Context, token, networkID, resourceID string) (*NetworkResource, error) {
	var resource NetworkResource
	path := "/networks/" + url.PathEscape(networkID) + "/resources/" + url.PathEscape(resourceID)
	if err := c.do(ctx, token, http.MethodGet, path, nil, nil, &resource); err != nil {
		return nil, err
	}
	return &resource, nil
}

// CreateNetworkResource adds a resource to a network. The server derives
// Type from req.Address; this dashboard never sends one.
func (c *Client) CreateNetworkResource(ctx context.Context, token, networkID string, req NetworkResourceRequest) (*NetworkResource, error) {
	var resource NetworkResource
	path := "/networks/" + url.PathEscape(networkID) + "/resources"
	if err := c.do(ctx, token, http.MethodPost, path, nil, req, &resource); err != nil {
		return nil, err
	}
	return &resource, nil
}

// UpdateNetworkResource replaces a resource's editable fields, including its
// whole group membership — req.Groups must carry every group the resource
// should end up belonging to, not just changes.
func (c *Client) UpdateNetworkResource(ctx context.Context, token, networkID, resourceID string, req NetworkResourceRequest) (*NetworkResource, error) {
	var resource NetworkResource
	path := "/networks/" + url.PathEscape(networkID) + "/resources/" + url.PathEscape(resourceID)
	if err := c.do(ctx, token, http.MethodPut, path, nil, req, &resource); err != nil {
		return nil, err
	}
	return &resource, nil
}

// DeleteNetworkResource removes a resource from a network. The server
// refuses to delete one still backing a reverse-proxy service, surfaced as a
// normal API error this call returns unchanged.
func (c *Client) DeleteNetworkResource(ctx context.Context, token, networkID, resourceID string) error {
	path := "/networks/" + url.PathEscape(networkID) + "/resources/" + url.PathEscape(resourceID)
	return c.do(ctx, token, http.MethodDelete, path, nil, nil, nil)
}

func sortResources(resources []NetworkResource) {
	sort.SliceStable(resources, func(i, j int) bool {
		return strings.ToLower(resources[i].Name) < strings.ToLower(resources[j].Name)
	})
}

// ListNetworkRouters returns every router in the given network.
func (c *Client) ListNetworkRouters(ctx context.Context, token, networkID string) ([]NetworkRouter, error) {
	var routers []NetworkRouter
	path := "/networks/" + url.PathEscape(networkID) + "/routers"
	if err := c.do(ctx, token, http.MethodGet, path, nil, nil, &routers); err != nil {
		return nil, err
	}
	return routers, nil
}

// GetNetworkRouter returns a single router.
func (c *Client) GetNetworkRouter(ctx context.Context, token, networkID, routerID string) (*NetworkRouter, error) {
	var router NetworkRouter
	path := "/networks/" + url.PathEscape(networkID) + "/routers/" + url.PathEscape(routerID)
	if err := c.do(ctx, token, http.MethodGet, path, nil, nil, &router); err != nil {
		return nil, err
	}
	return &router, nil
}

// CreateNetworkRouter adds a router to a network. The server forces Enabled
// to true on create regardless of req.Enabled — see NetworkRouterRequest.
func (c *Client) CreateNetworkRouter(ctx context.Context, token, networkID string, req NetworkRouterRequest) (*NetworkRouter, error) {
	var router NetworkRouter
	path := "/networks/" + url.PathEscape(networkID) + "/routers"
	if err := c.do(ctx, token, http.MethodPost, path, nil, req, &router); err != nil {
		return nil, err
	}
	return &router, nil
}

// UpdateNetworkRouter replaces a router's editable fields. req.Peer and
// req.PeerGroups are mutually exclusive and one is required — see
// NetworkRouterRequest.
func (c *Client) UpdateNetworkRouter(ctx context.Context, token, networkID, routerID string, req NetworkRouterRequest) (*NetworkRouter, error) {
	var router NetworkRouter
	path := "/networks/" + url.PathEscape(networkID) + "/routers/" + url.PathEscape(routerID)
	if err := c.do(ctx, token, http.MethodPut, path, nil, req, &router); err != nil {
		return nil, err
	}
	return &router, nil
}

// DeleteNetworkRouter removes a router from a network.
func (c *Client) DeleteNetworkRouter(ctx context.Context, token, networkID, routerID string) error {
	path := "/networks/" + url.PathEscape(networkID) + "/routers/" + url.PathEscape(routerID)
	return c.do(ctx, token, http.MethodDelete, path, nil, nil, nil)
}
