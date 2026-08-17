import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiDelete, apiGet, apiPost, apiPut } from "./client";

// Mirrors internal/nbapi/reverseproxy.go.
export interface ProxyCluster {
  id: string;
  address: string;
  type: "account" | "shared";
  online: boolean;
  connected_proxies: number;
  supports_custom_ports?: boolean;
  require_subdomain?: boolean;
  supports_crowdsec?: boolean;
  private?: boolean;
}

export interface ProxyToken {
  id: string;
  name: string;
  expires_at?: string;
  created_at: string;
  last_used?: string;
  revoked: boolean;
}

export interface ProxyTokenCreated extends ProxyToken {
  plain_token: string;
}

export interface ProxyTokenRequest {
  name: string;
  expires_in: number;
}

export interface ReverseProxyDomain {
  id: string;
  domain: string;
  validated: boolean;
  type: "free" | "custom";
  target_cluster?: string;
  supports_custom_ports?: boolean;
  require_subdomain?: boolean;
  supports_crowdsec?: boolean;
  supports_private?: boolean;
}

export interface ReverseProxyDomainRequest {
  domain: string;
  target_cluster: string;
}

export interface ServiceTarget {
  target_id: string;
  target_type: "peer" | "host" | "domain" | "subnet" | "cluster";
  path?: string;
  protocol: "http" | "https" | "tcp" | "udp";
  host?: string;
  port: number;
  enabled: boolean;
}

export interface ServiceMeta {
  created_at: string;
  certificate_issued_at?: string;
  status: string;
}

export interface Service {
  id: string;
  name: string;
  domain: string;
  mode: "http" | "tcp" | "udp" | "tls";
  listen_port?: number;
  port_auto_assigned?: boolean;
  proxy_cluster?: string;
  targets: ServiceTarget[];
  enabled: boolean;
  terminated?: boolean;
  pass_host_header?: boolean;
  rewrite_redirects?: boolean;
  meta: ServiceMeta;
  private?: boolean;
}

export interface ServiceRequest {
  name: string;
  domain: string;
  mode: Service["mode"];
  listen_port: number;
  targets: ServiceTarget[];
  enabled: boolean;
  pass_host_header: boolean;
  rewrite_redirects: boolean;
}

const keys = {
  clusters: ["reverse-proxy", "clusters"] as const,
  tokens: ["reverse-proxy", "proxy-tokens"] as const,
  domains: ["reverse-proxy", "domains"] as const,
  services: ["reverse-proxy", "services"] as const,
  service: (id: string) => ["reverse-proxy", "services", id] as const,
};

export function useProxyClusters() {
  return useQuery({
    queryKey: keys.clusters,
    // The server sends `null`, not `[]`, when there are zero clusters —
    // never assume this is a real array.
    queryFn: () => apiGet<{ clusters: ProxyCluster[] | null; total: number }>("/reverse-proxy/clusters"),
  });
}

export function useDeleteProxyCluster() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (address: string) => apiDelete(`/reverse-proxy/clusters/${encodeURIComponent(address)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.clusters }),
  });
}

export function useProxyTokens() {
  return useQuery({
    queryKey: keys.tokens,
    // The server sends `null`, not `[]`, when there are zero tokens — never
    // assume this is a real array.
    queryFn: () => apiGet<{ tokens: ProxyToken[] | null; total: number }>("/reverse-proxy/proxy-tokens"),
  });
}

export function useCreateProxyToken() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ProxyTokenRequest) => apiPost<ProxyTokenCreated>("/reverse-proxy/proxy-tokens", req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.tokens }),
  });
}

export function useDeleteProxyToken() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/reverse-proxy/proxy-tokens/${encodeURIComponent(id)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.tokens }),
  });
}

export function useReverseProxyDomains() {
  return useQuery({
    queryKey: keys.domains,
    // The server sends `null`, not `[]`, when there are zero domains —
    // never assume this is a real array.
    queryFn: () => apiGet<{ domains: ReverseProxyDomain[] | null; total: number }>("/reverse-proxy/domains"),
  });
}

export function useCreateReverseProxyDomain() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ReverseProxyDomainRequest) => apiPost<void>("/reverse-proxy/domains", req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.domains }),
  });
}

export function useDeleteReverseProxyDomain() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/reverse-proxy/domains/${encodeURIComponent(id)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.domains }),
  });
}

export function useValidateReverseProxyDomain() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiGet<void>(`/reverse-proxy/domains/${encodeURIComponent(id)}/validate`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.domains }),
  });
}

export function useServices() {
  return useQuery({
    queryKey: keys.services,
    // The server sends `null`, not `[]`, when there are zero services —
    // never assume this is a real array.
    queryFn: () => apiGet<{ services: Service[] | null; total: number }>("/reverse-proxy/services"),
  });
}

export function useCreateService() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ServiceRequest) => apiPost<Service>("/reverse-proxy/services", req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.services }),
  });
}

export function useUpdateService(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ServiceRequest) => apiPut<Service>(`/reverse-proxy/services/${encodeURIComponent(id)}`, req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.services }),
  });
}

export function useDeleteService() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/reverse-proxy/services/${encodeURIComponent(id)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.services }),
  });
}
