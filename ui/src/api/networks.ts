import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { PeerMinimum } from "./groups";
import { apiDelete, apiGet, apiPost, apiPut } from "./client";

export interface Network {
  id: string;
  name: string;
  description: string;
  // The server sends `null`, not `[]`, for a network with none yet — never
  // assume these are real arrays.
  routers: string[] | null;
  resources: string[] | null;
  policies: string[] | null;
  routing_peers_count: number;
}

export interface NetworksResponse {
  networks: Network[];
  total: number;
}

export interface NetworkRequest {
  name: string;
  description: string;
}

export interface NetworkResource {
  id: string;
  name: string;
  description: string;
  address: string;
  type: string;
  enabled: boolean;
  // The server sends `null`, not `[]`, when the resource's group-info lookup
  // has nothing to return — never assume this is a real array.
  groups: PeerMinimum[] | null;
}

export interface ResourceRequest {
  name: string;
  description: string;
  address: string;
  enabled: boolean;
  groups: string[];
}

export interface NetworkRouter {
  id: string;
  peer: string;
  // The server sends `null`, not `[]`, for a router that targets a single
  // peer instead of peer groups — never assume this is a real array.
  peer_groups: string[] | null;
  metric: number;
  masquerade: boolean;
  enabled: boolean;
}

export interface RouterRequest {
  peer: string;
  peer_groups: string[];
  metric: number;
  masquerade: boolean;
  enabled: boolean;
}

const keys = {
  all: ["networks"] as const,
  list: () => ["networks", "list"] as const,
  detail: (id: string) => ["networks", "detail", id] as const,
  resources: (id: string) => ["networks", id, "resources"] as const,
  routers: (id: string) => ["networks", id, "routers"] as const,
};

export function useNetworks(enabled = true) {
  return useQuery({
    queryKey: keys.list(),
    queryFn: () => apiGet<NetworksResponse>("/networks"),
    enabled,
  });
}

export function useNetwork(id: string) {
  return useQuery({
    queryKey: keys.detail(id),
    queryFn: () => apiGet<Network>(`/networks/${encodeURIComponent(id)}`),
    enabled: id !== "",
  });
}

export function useCreateNetwork() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: NetworkRequest) => apiPost<Network>("/networks", req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.all }),
  });
}

export function useUpdateNetwork(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: NetworkRequest) => apiPut<Network>(`/networks/${encodeURIComponent(id)}`, req),
    onSuccess: (network) => {
      queryClient.setQueryData(keys.detail(id), network);
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useDeleteNetwork() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/networks/${encodeURIComponent(id)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.all }),
  });
}

export function useResources(networkId: string) {
  return useQuery({
    queryKey: keys.resources(networkId),
    // The server sends `null`, not `[]`, for a network with no resources
    // yet — never assume `resources` is a real array.
    queryFn: () =>
      apiGet<{ resources: NetworkResource[] | null }>(`/networks/${encodeURIComponent(networkId)}/resources`),
    enabled: networkId !== "",
  });
}

export function useCreateResource(networkId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ResourceRequest) =>
      apiPost<NetworkResource>(`/networks/${encodeURIComponent(networkId)}/resources`, req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.resources(networkId) }),
  });
}

export function useUpdateResource(networkId: string, resourceId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: ResourceRequest) =>
      apiPut<NetworkResource>(
        `/networks/${encodeURIComponent(networkId)}/resources/${encodeURIComponent(resourceId)}`,
        req,
      ),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.resources(networkId) }),
  });
}

export function useDeleteResource(networkId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (resourceId: string) =>
      apiDelete(`/networks/${encodeURIComponent(networkId)}/resources/${encodeURIComponent(resourceId)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.resources(networkId) }),
  });
}

export function useRouters(networkId: string) {
  return useQuery({
    queryKey: keys.routers(networkId),
    // The server sends `null`, not `[]`, for a network with no routers yet
    // — never assume `routers` is a real array.
    queryFn: () =>
      apiGet<{ routers: NetworkRouter[] | null }>(`/networks/${encodeURIComponent(networkId)}/routers`),
    enabled: networkId !== "",
  });
}

export function useCreateRouter(networkId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: RouterRequest) =>
      apiPost<NetworkRouter>(`/networks/${encodeURIComponent(networkId)}/routers`, req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.routers(networkId) }),
  });
}

export function useUpdateRouter(networkId: string, routerId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: RouterRequest) =>
      apiPut<NetworkRouter>(
        `/networks/${encodeURIComponent(networkId)}/routers/${encodeURIComponent(routerId)}`,
        req,
      ),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.routers(networkId) }),
  });
}

export function useDeleteRouter(networkId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (routerId: string) =>
      apiDelete(`/networks/${encodeURIComponent(networkId)}/routers/${encodeURIComponent(routerId)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.routers(networkId) }),
  });
}
