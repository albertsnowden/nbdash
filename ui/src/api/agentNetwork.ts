import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiDelete, apiGet, apiPost, apiPut } from "./client";

// Mirrors internal/nbapi/agentnetwork.go — Models/ExtraValues/GuardrailIDs/
// Limits aren't modeled here since this dashboard's forms don't edit them
// (the backend preserves them on update by fetching-then-carrying-forward).
export interface AgentNetworkProvider {
  id: string;
  provider_id: string;
  name: string;
  upstream_url: string;
  enabled: boolean;
  skip_tls_verification?: boolean;
  metadata_disabled?: boolean;
}

export interface AgentNetworkProviderRequest {
  provider_id: string;
  name: string;
  upstream_url: string;
  bootstrap_cluster?: string;
  api_key: string;
  enabled: boolean;
  skip_tls_verification: boolean;
  metadata_disabled: boolean;
}

export interface AgentNetworkPolicy {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  source_groups: string[];
  destination_provider_ids: string[];
}

export interface AgentNetworkPolicyRequest {
  name: string;
  description: string;
  enabled: boolean;
  source_groups: string[];
  destination_provider_ids: string[];
}

const keys = {
  providers: ["agent-network", "providers"] as const,
  policies: ["agent-network", "policies"] as const,
};

export function useAgentNetworkProviders(enabled = true) {
  return useQuery({
    queryKey: keys.providers,
    // The server sends `null`, not `[]`, when the account has zero
    // providers — never assume this is a real array.
    queryFn: () => apiGet<{ providers: AgentNetworkProvider[] | null; total: number }>("/agent-network/providers"),
    enabled,
  });
}

export function useCreateAgentNetworkProvider() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: AgentNetworkProviderRequest) =>
      apiPost<AgentNetworkProvider>("/agent-network/providers", req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.providers }),
  });
}

export function useUpdateAgentNetworkProvider(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: AgentNetworkProviderRequest) =>
      apiPut<AgentNetworkProvider>(`/agent-network/providers/${encodeURIComponent(id)}`, req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.providers }),
  });
}

export function useDeleteAgentNetworkProvider() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/agent-network/providers/${encodeURIComponent(id)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.providers }),
  });
}

export function useAgentNetworkPolicies(enabled = true) {
  return useQuery({
    queryKey: keys.policies,
    // The server sends `null`, not `[]`, when the account has zero agent
    // network policies — never assume this is a real array.
    queryFn: () => apiGet<{ policies: AgentNetworkPolicy[] | null; total: number }>("/agent-network/policies"),
    enabled,
  });
}

// Display label for an agent-network provider type — mirrors ProvidersTab's
// PROVIDER_TYPES list (kept local to that form) but exported here too so
// the Control Center graph's provider picker doesn't duplicate the map.
export const PROVIDER_TYPE_LABELS: Record<string, string> = {
  openai_api: "OpenAI",
  anthropic_api: "Anthropic",
  azure_openai_api: "Azure OpenAI",
  bedrock_api: "AWS Bedrock",
  vertex_ai_api: "Google Vertex AI",
  mistral_api: "Mistral",
  custom: "Custom",
};

export function useCreateAgentNetworkPolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: AgentNetworkPolicyRequest) => apiPost<AgentNetworkPolicy>("/agent-network/policies", req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.policies }),
  });
}

export function useUpdateAgentNetworkPolicy(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: AgentNetworkPolicyRequest) =>
      apiPut<AgentNetworkPolicy>(`/agent-network/policies/${encodeURIComponent(id)}`, req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.policies }),
  });
}

export function useDeleteAgentNetworkPolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/agent-network/policies/${encodeURIComponent(id)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.policies }),
  });
}
