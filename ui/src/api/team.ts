import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiDelete, apiGet, apiPost, apiPut } from "./client";

export const VALID_ROLES = ["admin", "user", "billing_admin", "auditor", "network_admin"] as const;
export type Role = (typeof VALID_ROLES)[number];

export interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  status: "active" | "invited" | "blocked";
  auto_groups: string[];
  is_current: boolean;
  is_service_user: boolean;
  is_blocked: boolean;
  pending_approval: boolean;
  last_login: string;
}

export interface UsersResponse {
  // The server sends `null`, not `[]`, when there are zero users of this
  // kind (e.g. no service users yet) — never assume this is a real array.
  users: User[] | null;
  total: number;
}

export interface InviteUserRequest {
  name: string;
  email: string;
  role: string;
  auto_groups: string[];
}

export interface CreateServiceUserRequest {
  name: string;
  role: string;
  auto_groups: string[];
}

export interface UpdateUserRequest {
  role: string;
  auto_groups: string[];
  is_blocked: boolean;
}

export interface PersonalAccessToken {
  id: string;
  name: string;
  expiration_date: string;
  created_by: string;
  created_at: string;
  last_used: string;
}

export interface PATGenerated {
  plain_token: string;
  personal_access_token: PersonalAccessToken;
}

export interface CreatePATRequest {
  name: string;
  expires_in: number;
}

const keys = {
  users: ["team", "users"] as const,
  serviceUsers: ["team", "service-users"] as const,
  tokens: (userId: string) => ["team", "service-users", userId, "tokens"] as const,
};

export function useUsers() {
  return useQuery({ queryKey: keys.users, queryFn: () => apiGet<UsersResponse>("/team/users") });
}

export function useInviteUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: InviteUserRequest) => apiPost<User>("/team/users", req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.users }),
  });
}

export function useUpdateUser(id: string, isServiceUser: boolean) {
  const queryClient = useQueryClient();
  const key = isServiceUser ? keys.serviceUsers : keys.users;
  const base = isServiceUser ? "/team/service-users" : "/team/users";
  return useMutation({
    mutationFn: (req: UpdateUserRequest) => apiPut<User>(`${base}/${encodeURIComponent(id)}`, req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: key }),
  });
}

export function useDeleteUser(isServiceUser: boolean) {
  const queryClient = useQueryClient();
  const key = isServiceUser ? keys.serviceUsers : keys.users;
  const base = isServiceUser ? "/team/service-users" : "/team/users";
  return useMutation({
    mutationFn: (id: string) => apiDelete(`${base}/${encodeURIComponent(id)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: key }),
  });
}

export function useResendInvite() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost<void>(`/team/users/${encodeURIComponent(id)}/invite`, {}),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.users }),
  });
}

export function useApproveUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost<User>(`/team/users/${encodeURIComponent(id)}/approve`, {}),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.users }),
  });
}

export function useRejectUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/team/users/${encodeURIComponent(id)}/reject`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.users }),
  });
}

export function useServiceUsers() {
  return useQuery({ queryKey: keys.serviceUsers, queryFn: () => apiGet<UsersResponse>("/team/service-users") });
}

export function useCreateServiceUser() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: CreateServiceUserRequest) => apiPost<User>("/team/service-users", req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.serviceUsers }),
  });
}

export function usePATs(userId: string) {
  return useQuery({
    queryKey: keys.tokens(userId),
    // The server sends `null`, not `[]`, when there are zero PATs — never
    // assume this is a real array.
    queryFn: () =>
      apiGet<{ tokens: PersonalAccessToken[] | null }>(
        `/team/service-users/${encodeURIComponent(userId)}/tokens`,
      ),
    enabled: userId !== "",
  });
}

export function useCreatePAT(userId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: CreatePATRequest) =>
      apiPost<PATGenerated>(`/team/service-users/${encodeURIComponent(userId)}/tokens`, req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.tokens(userId) }),
  });
}

export function useDeletePAT(userId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (tokenId: string) =>
      apiDelete(`/team/service-users/${encodeURIComponent(userId)}/tokens/${encodeURIComponent(tokenId)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.tokens(userId) }),
  });
}
