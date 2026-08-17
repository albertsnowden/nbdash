import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { PeerMinimum } from "./groups";
import { apiDelete, apiGet, apiPut } from "./client";

// Mirrors nbapi.Peer's JSON shape (internal/nbapi/types.go) — only the
// fields this feature actually renders, not the full wire response.
export interface Peer {
  id: string;
  name: string;
  ip: string;
  hostname: string;
  connected: boolean;
  last_seen: string;
  os: string;
  version: string;
  ssh_enabled: boolean;
  login_expiration_enabled: boolean;
  inactivity_expiration_enabled: boolean;
  user_id: string;
  country_code: string;
  // Same nil-slice-from-server shape as Group.peers — the server only
  // appends to this, never initialises it, so a peer with no groups (should
  // not happen in practice, since every peer joins "All") comes back null.
  groups: PeerMinimum[] | null;
}

export interface PeersResponse {
  peers: Peer[];
  total: number;
  connected_count: number;
}

export interface UpdatePeerRequest {
  name: string;
  ssh_enabled: boolean;
  login_expiration_enabled: boolean;
  inactivity_expiration_enabled: boolean;
}

const keys = {
  all: ["peers"] as const,
  list: (q: string) => ["peers", "list", q] as const,
  detail: (id: string) => ["peers", "detail", id] as const,
};

export function usePeers(query: string) {
  return useQuery({
    queryKey: keys.list(query),
    queryFn: () =>
      apiGet<PeersResponse>(
        `/peers${query ? `?q=${encodeURIComponent(query)}` : ""}`,
      ),
  });
}

export function usePeer(id: string) {
  return useQuery({
    queryKey: keys.detail(id),
    queryFn: () => apiGet<Peer>(`/peers/${encodeURIComponent(id)}`),
    enabled: id !== "",
  });
}

// invalidateQueries({queryKey: keys.all}) on every mutation is this app's
// direct equivalent of the htmx app's `HX-Trigger: peersChanged` response
// header — it's what makes the list/summary refetch after an edit or delete.
export function useUpdatePeer(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: UpdatePeerRequest) =>
      apiPut<Peer>(`/peers/${encodeURIComponent(id)}`, req),
    onSuccess: (peer) => {
      queryClient.setQueryData(keys.detail(id), peer);
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useDeletePeer() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/peers/${encodeURIComponent(id)}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}
