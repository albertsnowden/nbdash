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

// AgentNetworkProvider is an account's connection to an upstream AI
// provider (OpenAI, Anthropic, ...). Models and ExtraValues are kept as raw
// JSON: this dashboard's form doesn't expose per-model price overrides or
// catalog-declared extra headers yet, and round-tripping them unparsed
// means saving a provider through this UI can't silently clear them.
type AgentNetworkProvider struct {
	ID                   string          `json:"id"`
	ProviderID           string          `json:"provider_id"`
	Name                 string          `json:"name"`
	UpstreamURL          string          `json:"upstream_url"`
	Models               json.RawMessage `json:"models,omitempty"`
	ExtraValues          json.RawMessage `json:"extra_values,omitempty"`
	IdentityHeaderUserID string          `json:"identity_header_user_id,omitempty"`
	IdentityHeaderGroups string          `json:"identity_header_groups,omitempty"`
	Enabled              bool            `json:"enabled"`
	SkipTLSVerification  bool            `json:"skip_tls_verification,omitempty"`
	MetadataDisabled     bool            `json:"metadata_disabled,omitempty"`
	CreatedAt            time.Time       `json:"created_at,omitempty"`
	UpdatedAt            time.Time       `json:"updated_at,omitempty"`
}

type AgentNetworkProviderRequest struct {
	ProviderID           string          `json:"provider_id"`
	Name                 string          `json:"name"`
	UpstreamURL          string          `json:"upstream_url"`
	BootstrapCluster     string          `json:"bootstrap_cluster,omitempty"`
	APIKey               string          `json:"api_key,omitempty"`
	Models               json.RawMessage `json:"models,omitempty"`
	ExtraValues          json.RawMessage `json:"extra_values,omitempty"`
	IdentityHeaderUserID string          `json:"identity_header_user_id,omitempty"`
	IdentityHeaderGroups string          `json:"identity_header_groups,omitempty"`
	Enabled              bool            `json:"enabled"`
	SkipTLSVerification  bool            `json:"skip_tls_verification,omitempty"`
	MetadataDisabled     bool            `json:"metadata_disabled,omitempty"`
}

func (c *Client) ListAgentNetworkProviders(ctx context.Context, token string) ([]AgentNetworkProvider, error) {
	var providers []AgentNetworkProvider
	if err := c.do(ctx, token, http.MethodGet, "/agent-network/providers", nil, nil, &providers); err != nil {
		return nil, err
	}
	sort.SliceStable(providers, func(i, j int) bool {
		return strings.ToLower(providers[i].Name) < strings.ToLower(providers[j].Name)
	})
	return providers, nil
}

func (c *Client) GetAgentNetworkProvider(ctx context.Context, token, id string) (*AgentNetworkProvider, error) {
	var provider AgentNetworkProvider
	if err := c.do(ctx, token, http.MethodGet, "/agent-network/providers/"+url.PathEscape(id), nil, nil, &provider); err != nil {
		return nil, err
	}
	return &provider, nil
}

func (c *Client) CreateAgentNetworkProvider(ctx context.Context, token string, req AgentNetworkProviderRequest) (*AgentNetworkProvider, error) {
	var provider AgentNetworkProvider
	if err := c.do(ctx, token, http.MethodPost, "/agent-network/providers", nil, req, &provider); err != nil {
		return nil, err
	}
	return &provider, nil
}

func (c *Client) UpdateAgentNetworkProvider(ctx context.Context, token, id string, req AgentNetworkProviderRequest) (*AgentNetworkProvider, error) {
	var provider AgentNetworkProvider
	if err := c.do(ctx, token, http.MethodPut, "/agent-network/providers/"+url.PathEscape(id), nil, req, &provider); err != nil {
		return nil, err
	}
	return &provider, nil
}

func (c *Client) DeleteAgentNetworkProvider(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/agent-network/providers/"+url.PathEscape(id), nil, nil, nil)
}

// AgentNetworkPolicy binds source groups to destination providers.
// GuardrailIDs and Limits are kept as raw/untouched by this dashboard's
// form for the same reason AgentNetworkProvider.Models is — no guardrail or
// rate-limit management UI yet, so PUT must not silently clear them.
type AgentNetworkPolicy struct {
	ID                     string          `json:"id"`
	Name                   string          `json:"name"`
	Description            string          `json:"description"`
	Enabled                bool            `json:"enabled"`
	SourceGroups           []string        `json:"source_groups"`
	DestinationProviderIDs []string        `json:"destination_provider_ids"`
	GuardrailIDs           []string        `json:"guardrail_ids"`
	Limits                 json.RawMessage `json:"limits,omitempty"`
	CreatedAt              time.Time       `json:"created_at,omitempty"`
	UpdatedAt              time.Time       `json:"updated_at,omitempty"`
}

type AgentNetworkPolicyRequest struct {
	Name                   string          `json:"name"`
	Description            string          `json:"description"`
	Enabled                bool            `json:"enabled"`
	SourceGroups           []string        `json:"source_groups"`
	DestinationProviderIDs []string        `json:"destination_provider_ids"`
	GuardrailIDs           []string        `json:"guardrail_ids,omitempty"`
	Limits                 json.RawMessage `json:"limits,omitempty"`
}

func (c *Client) ListAgentNetworkPolicies(ctx context.Context, token string) ([]AgentNetworkPolicy, error) {
	var policies []AgentNetworkPolicy
	if err := c.do(ctx, token, http.MethodGet, "/agent-network/policies", nil, nil, &policies); err != nil {
		return nil, err
	}
	sort.SliceStable(policies, func(i, j int) bool {
		return strings.ToLower(policies[i].Name) < strings.ToLower(policies[j].Name)
	})
	return policies, nil
}

func (c *Client) GetAgentNetworkPolicy(ctx context.Context, token, id string) (*AgentNetworkPolicy, error) {
	var policy AgentNetworkPolicy
	if err := c.do(ctx, token, http.MethodGet, "/agent-network/policies/"+url.PathEscape(id), nil, nil, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

func (c *Client) CreateAgentNetworkPolicy(ctx context.Context, token string, req AgentNetworkPolicyRequest) (*AgentNetworkPolicy, error) {
	var policy AgentNetworkPolicy
	if err := c.do(ctx, token, http.MethodPost, "/agent-network/policies", nil, req, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

func (c *Client) UpdateAgentNetworkPolicy(ctx context.Context, token, id string, req AgentNetworkPolicyRequest) (*AgentNetworkPolicy, error) {
	var policy AgentNetworkPolicy
	if err := c.do(ctx, token, http.MethodPut, "/agent-network/policies/"+url.PathEscape(id), nil, req, &policy); err != nil {
		return nil, err
	}
	return &policy, nil
}

func (c *Client) DeleteAgentNetworkPolicy(ctx context.Context, token, id string) error {
	return c.do(ctx, token, http.MethodDelete, "/agent-network/policies/"+url.PathEscape(id), nil, nil, nil)
}
