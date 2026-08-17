import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiDelete, apiGet, apiPost } from "./client";

// Key is masked (e.g. "A6160****") on every response except the one
// returned by create, which carries the plaintext once — see
// internal/api/setupkeys.go's doc comment on createSetupKey.
export interface SetupKey {
  id: string;
  name: string;
  key: string;
  expires: string;
  type: string;
  valid: boolean;
  revoked: boolean;
  used_times: number;
  last_used: string;
  state: string;
  auto_groups: string[];
  usage_limit: number;
  ephemeral: boolean;
}

export interface SetupKeysResponse {
  setup_keys: SetupKey[];
  total: number;
  valid: number;
}

export interface CreateSetupKeyRequest {
  name: string;
  type: "one-off" | "reusable";
  expires_in_days: number;
  auto_groups: string[];
  usage_limit: number;
  ephemeral: boolean;
}

const keys = {
  all: ["setup-keys"] as const,
  list: () => ["setup-keys", "list"] as const,
};

export function useSetupKeys() {
  return useQuery({
    queryKey: keys.list(),
    queryFn: () => apiGet<SetupKeysResponse>("/setup-keys"),
  });
}

export function useCreateSetupKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: CreateSetupKeyRequest) => apiPost<SetupKey>("/setup-keys", req),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useRevokeSetupKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiPost<SetupKey>(`/setup-keys/${encodeURIComponent(id)}/revoke`, {}),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useDeleteSetupKey() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/setup-keys/${encodeURIComponent(id)}`),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}
