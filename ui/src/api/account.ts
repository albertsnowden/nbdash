import { useMutation, useQuery } from "@tanstack/react-query";
import { apiDelete, apiGet } from "./client";

// Mirrors nbapi.Account (internal/nbapi/accounts.go).
export interface Account {
  id: string;
  domain?: string;
  created_at?: string;
  created_by?: string;
}

export function useCurrentAccount() {
  return useQuery({
    queryKey: ["account"] as const,
    queryFn: () => apiGet<Account>("/account"),
  });
}

export function useDeleteAccount() {
  return useMutation({
    mutationFn: () => apiDelete("/account"),
  });
}
