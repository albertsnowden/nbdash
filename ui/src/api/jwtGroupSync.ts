import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiGet, apiPut } from "./client";

// Mirrors internal/api/jwtgroupsync.go's jwtGroupSync.
export interface JWTGroupSync {
  jwt_groups_enabled: boolean;
  jwt_groups_claim_name: string;
  jwt_allow_groups: string[];
}

const key = ["settings", "jwt-group-sync"] as const;

export function useJWTGroupSync() {
  return useQuery({
    queryKey: key,
    queryFn: () => apiGet<JWTGroupSync>("/settings/jwt-group-sync"),
  });
}

export function useUpdateJWTGroupSync() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: JWTGroupSync) => apiPut<JWTGroupSync>("/settings/jwt-group-sync", req),
    onSuccess: (data) => queryClient.setQueryData(key, data),
  });
}
