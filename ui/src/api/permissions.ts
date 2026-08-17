import { useQuery } from "@tanstack/react-query";
import { apiGet } from "./client";

// Mirrors internal/api/permissions.go's permissionsResponse.
export interface UserPermissions {
  is_restricted: boolean;
  modules: Record<string, Record<string, boolean>>;
}

export interface PermissionsResponse {
  role: string;
  permissions: UserPermissions;
}

export function usePermissions() {
  return useQuery({
    queryKey: ["permissions"] as const,
    queryFn: () => apiGet<PermissionsResponse>("/permissions"),
  });
}
