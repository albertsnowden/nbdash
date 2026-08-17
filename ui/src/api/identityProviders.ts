import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiDelete, apiGet, apiPost, apiPut } from "./client";

// Matches nbapi.IdentityProviderTypes' order (internal/nbapi/types.go) —
// the last two (authentik, keycloak) are missing from the checked-in
// OpenAPI spec's enum but fully implemented server-side; see that file's
// doc comment.
export const IDENTITY_PROVIDER_TYPES = [
  "oidc",
  "google",
  "microsoft",
  "entra",
  "okta",
  "zitadel",
  "pocketid",
  "authentik",
  "keycloak",
  "adfs",
] as const;
export type IdentityProviderType = (typeof IDENTITY_PROVIDER_TYPES)[number];

export const IDENTITY_PROVIDER_LABELS: Record<IdentityProviderType, string> = {
  oidc: "Generic OIDC",
  google: "Google",
  microsoft: "Microsoft",
  entra: "Microsoft Entra",
  okta: "Okta",
  zitadel: "Zitadel",
  pocketid: "PocketID",
  authentik: "Authentik",
  keycloak: "Keycloak",
  adfs: "Microsoft AD FS",
};

// Google and Microsoft use Dex's built-in connectors with hardcoded issuer
// endpoints — the server ignores any issuer submitted for these two.
export function hasBuiltInIssuer(type: IdentityProviderType): boolean {
  return type === "google" || type === "microsoft";
}

export interface IdentityProvider {
  id: string;
  type: IdentityProviderType;
  name: string;
  issuer: string;
  client_id: string;
}

export interface IdentityProvidersResponse {
  providers: IdentityProvider[];
  total: number;
}

export interface IdentityProviderRequest {
  type: IdentityProviderType;
  name: string;
  issuer: string;
  client_id: string;
  client_secret: string;
}

const keys = {
  all: ["identity-providers"] as const,
  list: () => ["identity-providers", "list"] as const,
  detail: (id: string) => ["identity-providers", "detail", id] as const,
};

export function useIdentityProviders() {
  return useQuery({
    queryKey: keys.list(),
    queryFn: () => apiGet<IdentityProvidersResponse>("/settings/identity-providers"),
  });
}

export function useIdentityProvider(id: string) {
  return useQuery({
    queryKey: keys.detail(id),
    queryFn: () => apiGet<IdentityProvider>(`/settings/identity-providers/${encodeURIComponent(id)}`),
    enabled: id !== "",
  });
}

export function useCreateIdentityProvider() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: IdentityProviderRequest) => apiPost<IdentityProvider>("/settings/identity-providers", req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.all }),
  });
}

export function useUpdateIdentityProvider(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: IdentityProviderRequest) =>
      apiPut<IdentityProvider>(`/settings/identity-providers/${encodeURIComponent(id)}`, req),
    onSuccess: (provider) => {
      queryClient.setQueryData(keys.detail(id), provider);
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useDeleteIdentityProvider() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/settings/identity-providers/${encodeURIComponent(id)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.all }),
  });
}
