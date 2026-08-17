import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiDelete, apiGet, apiPost, apiPut } from "./client";

export interface Nameserver {
  ip: string;
  ns_type: string;
  port: number;
}

export interface NameserverGroup {
  id: string;
  name: string;
  description: string;
  nameservers: Nameserver[];
  enabled: boolean;
  groups: string[];
  primary: boolean;
  domains: string[];
  search_domains_enabled: boolean;
}

export interface NameserverGroupRequest {
  name: string;
  description: string;
  nameservers: { ip: string; port: number }[];
  enabled: boolean;
  groups: string[];
  primary: boolean;
  domains: string[];
  search_domains_enabled: boolean;
}

export interface DNSSettings {
  disabled_management_groups: string[];
}

export interface DNSRecord {
  id: string;
  name: string;
  type: "A" | "AAAA" | "CNAME";
  content: string;
  ttl: number;
}

export interface RecordRequest {
  name: string;
  type: "A" | "AAAA" | "CNAME";
  content: string;
  ttl: number;
}

export interface Zone {
  id: string;
  name: string;
  domain: string;
  enabled: boolean;
  enable_search_domain: boolean;
  distribution_groups: string[];
  records: DNSRecord[];
}

export interface ZoneRequest {
  name: string;
  domain: string;
  enabled: boolean;
  enable_search_domain: boolean;
  distribution_groups: string[];
}

const keys = {
  nameservers: ["dns", "nameservers"] as const,
  settings: ["dns", "settings"] as const,
  zones: ["dns", "zones"] as const,
  zone: (id: string) => ["dns", "zones", id] as const,
};

export function useNameserverGroups() {
  return useQuery({
    queryKey: keys.nameservers,
    // The server sends `null`, not `[]`, when the account has zero
    // nameserver groups — never assume this is a real array.
    queryFn: () => apiGet<{ groups: NameserverGroup[] | null; total: number }>("/dns/nameservers"),
  });
}

export function useCreateNameserverGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: NameserverGroupRequest) => apiPost<NameserverGroup>("/dns/nameservers", req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.nameservers }),
  });
}

export function useUpdateNameserverGroup(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: NameserverGroupRequest) =>
      apiPut<NameserverGroup>(`/dns/nameservers/${encodeURIComponent(id)}`, req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.nameservers }),
  });
}

export function useDeleteNameserverGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/dns/nameservers/${encodeURIComponent(id)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.nameservers }),
  });
}

export function useDNSSettings() {
  return useQuery({ queryKey: keys.settings, queryFn: () => apiGet<DNSSettings>("/dns/settings") });
}

export function useUpdateDNSSettings() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (settings: DNSSettings) => apiPut<DNSSettings>("/dns/settings", settings),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.settings }),
  });
}

export function useZones() {
  return useQuery({ queryKey: keys.zones, queryFn: () => apiGet<{ zones: Zone[]; total: number }>("/dns/zones") });
}

export function useZone(id: string) {
  return useQuery({
    queryKey: keys.zone(id),
    queryFn: () => apiGet<Zone>(`/dns/zones/${encodeURIComponent(id)}`),
    enabled: id !== "",
  });
}

export function useCreateZone() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ZoneRequest) => apiPost<Zone>("/dns/zones", req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.zones }),
  });
}

export function useUpdateZone(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ZoneRequest) => apiPut<Zone>(`/dns/zones/${encodeURIComponent(id)}`, req),
    onSuccess: (zone) => {
      queryClient.setQueryData(keys.zone(id), zone);
      void queryClient.invalidateQueries({ queryKey: keys.zones });
    },
  });
}

export function useDeleteZone() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/dns/zones/${encodeURIComponent(id)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.zones }),
  });
}

export function useCreateRecord(zoneId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: RecordRequest) => apiPost<DNSRecord>(`/dns/zones/${encodeURIComponent(zoneId)}/records`, req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.zone(zoneId) }),
  });
}

export function useUpdateRecord(zoneId: string, recordId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: RecordRequest) =>
      apiPut<DNSRecord>(`/dns/zones/${encodeURIComponent(zoneId)}/records/${encodeURIComponent(recordId)}`, req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.zone(zoneId) }),
  });
}

export function useDeleteRecord(zoneId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (recordId: string) =>
      apiDelete(`/dns/zones/${encodeURIComponent(zoneId)}/records/${encodeURIComponent(recordId)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.zone(zoneId) }),
  });
}
