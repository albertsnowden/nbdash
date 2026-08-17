import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiDelete, apiGet, apiPost, apiPut } from "./client";

// The built-in group every account has and every peer joins automatically.
// The server rejects renaming, editing the membership of, or deleting it —
// checked by exact name (types.Group.IsGroupAll in netbird), not a flag in
// the response — so the UI hides those actions rather than offering them
// and surfacing the server's rejection after the fact. Mirrors
// internal/handlers/groups.go's protectedGroupName.
export const PROTECTED_GROUP_NAME = "All";

export interface PeerMinimum {
  id: string;
  name: string;
}

export interface Group {
  id: string;
  name: string;
  peers_count: number;
  resources_count: number;
  // The server sends `null`, not `[]`, for a group with zero peers — never
  // assume this is a real array.
  peers: PeerMinimum[] | null;
}

export interface GroupsResponse {
  groups: Group[];
  total: number;
}

export interface GroupRequest {
  name: string;
  peers: string[];
}

const keys = {
  all: ["groups"] as const,
  list: () => ["groups", "list"] as const,
  detail: (id: string) => ["groups", "detail", id] as const,
};

export function useGroups() {
  return useQuery({
    queryKey: keys.list(),
    queryFn: () => apiGet<GroupsResponse>("/groups"),
  });
}

export function useGroup(id: string) {
  return useQuery({
    queryKey: keys.detail(id),
    queryFn: () => apiGet<Group>(`/groups/${encodeURIComponent(id)}`),
    enabled: id !== "",
  });
}

export function useCreateGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: GroupRequest) => apiPost<Group>("/groups", req),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useUpdateGroup(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: GroupRequest) => apiPut<Group>(`/groups/${encodeURIComponent(id)}`, req),
    onSuccess: (group) => {
      queryClient.setQueryData(keys.detail(id), group);
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useDeleteGroup() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/groups/${encodeURIComponent(id)}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}
