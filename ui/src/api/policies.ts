import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { PeerMinimum } from "./groups";
import { apiDelete, apiGet, apiPost, apiPut } from "./client";

export interface PolicyRule {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  action: "accept" | "drop";
  bidirectional: boolean;
  protocol: "all" | "tcp" | "udp" | "icmp" | "netbird-ssh";
  // The server sends `null`, not `[]`, for sources/destinations when a
  // referenced group no longer exists, and for ports whenever the rule
  // isn't port-restricted (the common case) — never assume these are real
  // arrays.
  sources: PeerMinimum[] | null;
  destinations: PeerMinimum[] | null;
  ports: string[] | null;
}

export interface Policy {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  rules: PolicyRule[];
  source_posture_checks: string[];
}

export interface PoliciesResponse {
  policies: Policy[];
  total: number;
  enabled: number;
}

export interface RuleRequest {
  name: string;
  description: string;
  enabled: boolean;
  action: "accept" | "drop";
  bidirectional: boolean;
  protocol: "all" | "tcp" | "udp" | "icmp";
  sources: string[];
  destinations: string[];
  ports: string[];
}

export interface CreatePolicyRequest {
  name: string;
  description: string;
  enabled: boolean;
  rule: RuleRequest;
  source_posture_checks: string[];
}

export interface PolicyMetaRequest {
  name: string;
  description: string;
  enabled: boolean;
  source_posture_checks: string[];
}

const keys = {
  all: ["policies"] as const,
  list: () => ["policies", "list"] as const,
  detail: (id: string) => ["policies", "detail", id] as const,
};

export function usePolicies() {
  return useQuery({
    queryKey: keys.list(),
    queryFn: () => apiGet<PoliciesResponse>("/policies"),
  });
}

export function usePolicy(id: string) {
  return useQuery({
    queryKey: keys.detail(id),
    queryFn: () => apiGet<Policy>(`/policies/${encodeURIComponent(id)}`),
    enabled: id !== "",
  });
}

export function useCreatePolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: CreatePolicyRequest) => apiPost<Policy>("/policies", req),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useUpdatePolicyMeta(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: PolicyMetaRequest) => apiPut<Policy>(`/policies/${encodeURIComponent(id)}`, req),
    onSuccess: (policy) => {
      queryClient.setQueryData(keys.detail(id), policy);
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useDeletePolicy() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/policies/${encodeURIComponent(id)}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useCreateRule(policyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: RuleRequest) => apiPost<Policy>(`/policies/${encodeURIComponent(policyId)}/rules`, req),
    onSuccess: (policy) => {
      queryClient.setQueryData(keys.detail(policyId), policy);
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useUpdateRule(policyId: string, ruleId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: RuleRequest) =>
      apiPut<Policy>(`/policies/${encodeURIComponent(policyId)}/rules/${encodeURIComponent(ruleId)}`, req),
    onSuccess: (policy) => {
      queryClient.setQueryData(keys.detail(policyId), policy);
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useDeleteRule(policyId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (ruleId: string) =>
      apiDelete(`/policies/${encodeURIComponent(policyId)}/rules/${encodeURIComponent(ruleId)}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.detail(policyId) });
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}
