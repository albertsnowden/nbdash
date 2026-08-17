package nbapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// --- Proxy clusters (read-only status, BYOP clusters deletable) ---

type ProxyCluster struct {
	ID                  string `json:"id"`
	Address             string `json:"address"`
	Type                string `json:"type"` // "account" | "shared"
	Online              bool   `json:"online"`
	ConnectedProxies    int    `json:"connected_proxies"`
	SupportsCustomPorts bool   `json:"supports_custom_ports,omitempty"`
	RequireSubdomain    bool   `json:"require_subdomain,omitempty"`
	SupportsCrowdsec    bool   `json:"supports_crowdsec,omitempty"`
	Private             bool   `json:"private,omitempty"`
}

func (c *Client) ListProxyClusters(ctx context.Context, token string) ([]ProxyCluster, error) {
	var clusters []ProxyCluster
	if err := c.do(ctx, token, http.MethodGet, "/reverse-proxies/clusters", nil, nil, &clusters); err != nil {
		return nil, err
	}
	sort.SliceStable(clusters, func(i, j int) bool {
		return strings.ToLower(clusters[i].Address) < strings.ToLower(clusters[j].Address)
	})
	return clusters, nil
}

// DeleteProxyCluster removes a self-hosted (BYOP) cluster registration —
// shared NetBird-operated clusters cannot be deleted this way.
func (c *Client) DeleteProxyCluster(ctx context.Context, token, address string) error {
	return c.do(ctx, token, http.MethodDelete, "/reverse-proxies/clusters/"+url.PathEscape(address), nil, nil, nil)
}

// --- Proxy tokens (self-hosted proxy registration) ---

type ProxyTokenRequest struct {
	Name      string `json:"name"`
	ExpiresIn int    `json:"expires_in"`
}

type ProxyToken struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used,omitempty"`
	Revoked   bool      `json:"revoked"`
}

// ProxyTokenCreated is what creation returns — the plain token is shown
// this once and never again.
type ProxyTokenCreated struct {
	ProxyToken
	PlainToken string `json:"plain_token"`
}

func (c *Client) ListProxyTokens(ctx context.Context, token string) ([]ProxyToken, error) {
	var tokens []ProxyToken
	if err := c.do(ctx, token, http.MethodGet, "/reverse-proxies/proxy-tokens", nil, nil, &tokens); err != nil {
		return nil, err
	}
	sort.SliceStable(tokens, func(i, j int) bool {
		return tokens[i].CreatedAt.After(tokens[j].CreatedAt)
	})
	return tokens, nil
}

func (c *Client) CreateProxyToken(ctx context.Context, token string, req ProxyTokenRequest) (*ProxyTokenCreated, error) {
	var created ProxyTokenCreated
	if err := c.do(ctx, token, http.MethodPost, "/reverse-proxies/proxy-tokens", nil, req, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) DeleteProxyToken(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/reverse-proxies/proxy-tokens/"+url.PathEscape(id), nil, nil, nil)
}

// --- Custom domains ---

type ReverseProxyDomain struct {
	ID                  string `json:"id"`
	Domain              string `json:"domain"`
	Validated           bool   `json:"validated"`
	Type                string `json:"type"` // "free" | "custom"
	TargetCluster       string `json:"target_cluster,omitempty"`
	SupportsCustomPorts bool   `json:"supports_custom_ports,omitempty"`
	RequireSubdomain    bool   `json:"require_subdomain,omitempty"`
	SupportsCrowdsec    bool   `json:"supports_crowdsec,omitempty"`
	SupportsPrivate     bool   `json:"supports_private,omitempty"`
}

type ReverseProxyDomainRequest struct {
	Domain        string `json:"domain"`
	TargetCluster string `json:"target_cluster"`
}

func (c *Client) ListReverseProxyDomains(ctx context.Context, token string) ([]ReverseProxyDomain, error) {
	var domains []ReverseProxyDomain
	if err := c.do(ctx, token, http.MethodGet, "/reverse-proxies/domains", nil, nil, &domains); err != nil {
		return nil, err
	}
	return domains, nil
}

func (c *Client) CreateReverseProxyDomain(ctx context.Context, token string, req ReverseProxyDomainRequest) error {
	return c.do(ctx, token, http.MethodPost, "/reverse-proxies/domains", nil, req, nil)
}

func (c *Client) DeleteReverseProxyDomain(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/reverse-proxies/domains/"+url.PathEscape(id), nil, nil, nil)
}

func (c *Client) ValidateReverseProxyDomain(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodGet, "/reverse-proxies/domains/"+url.PathEscape(id)+"/validate", nil, nil, nil)
}

// --- Services ---

// ServiceTarget is a backend behind a service. Options is kept as raw JSON
// rather than a typed struct: it carries per-target advanced knobs
// (skip_tls_verify, custom_headers, proxy_protocol, ...) this dashboard
// doesn't expose editing yet, and round-tripping it unparsed means saving a
// service through this UI can never silently wipe them.
type ServiceTarget struct {
	TargetID   string          `json:"target_id"`
	TargetType string          `json:"target_type"` // peer | host | domain | subnet | cluster
	Path       string          `json:"path,omitempty"`
	Protocol   string          `json:"protocol"` // http | https | tcp | udp
	Host       string          `json:"host,omitempty"`
	Port       int             `json:"port"`
	Enabled    bool            `json:"enabled"`
	Options    json.RawMessage `json:"options,omitempty"`
}

type ServiceMeta struct {
	CreatedAt           time.Time `json:"created_at"`
	CertificateIssuedAt time.Time `json:"certificate_issued_at,omitempty"`
	Status              string    `json:"status"`
}

// Service and ServiceRequest keep Auth/AccessRestrictions as raw JSON for
// the same reason ServiceTarget.Options is: this dashboard's create/edit
// form doesn't expose per-service auth (password/PIN/bearer/link/header) or
// IP/geo access restrictions yet, and a PUT is a full replace — sending a
// zero-valued Auth would silently disable whatever auth was configured
// through another client. internal/api/reverseproxy.go's updateService
// fetches the current service first and carries these fields forward
// unedited, mirroring this codebase's established "full-replace on PUT"
// pattern (see updateGroup, updateZoneMeta).
type Service struct {
	ID                 string          `json:"id"`
	Name               string          `json:"name"`
	Domain             string          `json:"domain"`
	Mode               string          `json:"mode,omitempty"` // http | tcp | udp | tls
	ListenPort         int             `json:"listen_port,omitempty"`
	PortAutoAssigned   bool            `json:"port_auto_assigned,omitempty"`
	ProxyCluster       string          `json:"proxy_cluster,omitempty"`
	Targets            []ServiceTarget `json:"targets"`
	Enabled            bool            `json:"enabled"`
	Terminated         bool            `json:"terminated,omitempty"`
	PassHostHeader     bool            `json:"pass_host_header,omitempty"`
	RewriteRedirects   bool            `json:"rewrite_redirects,omitempty"`
	Auth               json.RawMessage `json:"auth,omitempty"`
	AccessRestrictions json.RawMessage `json:"access_restrictions,omitempty"`
	Meta               ServiceMeta     `json:"meta"`
	Private            bool            `json:"private,omitempty"`
	AccessGroups       []string        `json:"access_groups,omitempty"`
}

type ServiceRequest struct {
	Name               string          `json:"name"`
	Domain             string          `json:"domain"`
	Mode               string          `json:"mode,omitempty"`
	ListenPort         int             `json:"listen_port"`
	Targets            []ServiceTarget `json:"targets"`
	Enabled            bool            `json:"enabled"`
	PassHostHeader     bool            `json:"pass_host_header,omitempty"`
	RewriteRedirects   bool            `json:"rewrite_redirects,omitempty"`
	Auth               json.RawMessage `json:"auth,omitempty"`
	AccessRestrictions json.RawMessage `json:"access_restrictions,omitempty"`
	Private            bool            `json:"private,omitempty"`
	AccessGroups       []string        `json:"access_groups,omitempty"`
}

func (c *Client) ListServices(ctx context.Context, token string) ([]Service, error) {
	var services []Service
	if err := c.do(ctx, token, http.MethodGet, "/reverse-proxies/services", nil, nil, &services); err != nil {
		return nil, err
	}
	sort.SliceStable(services, func(i, j int) bool {
		return strings.ToLower(services[i].Name) < strings.ToLower(services[j].Name)
	})
	return services, nil
}

func (c *Client) GetService(ctx context.Context, token, id string) (*Service, error) {
	var service Service
	if err := c.do(ctx, token, http.MethodGet, "/reverse-proxies/services/"+url.PathEscape(id), nil, nil, &service); err != nil {
		return nil, err
	}
	return &service, nil
}

func (c *Client) CreateService(ctx context.Context, token string, req ServiceRequest) (*Service, error) {
	var service Service
	if err := c.do(ctx, token, http.MethodPost, "/reverse-proxies/services", nil, req, &service); err != nil {
		return nil, err
	}
	return &service, nil
}

func (c *Client) UpdateService(ctx context.Context, token, id string, req ServiceRequest) (*Service, error) {
	var service Service
	if err := c.do(ctx, token, http.MethodPut, "/reverse-proxies/services/"+url.PathEscape(id), nil, req, &service); err != nil {
		return nil, err
	}
	return &service, nil
}

func (c *Client) DeleteService(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/reverse-proxies/services/"+url.PathEscape(id), nil, nil, nil)
}
