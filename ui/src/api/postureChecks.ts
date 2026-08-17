import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiDelete, apiGet, apiPost, apiPut } from "./client";

// Mirrors nbapi.{MinVersionCheck,MinKernelVersionCheck,OSVersionCheck,
// Location,GeoLocationCheck,PeerNetworkRangeCheck,Process,ProcessCheck,
// Checks,PostureCheck} (internal/nbapi/posturechecks.go).
export interface MinVersionCheck {
  min_version: string;
}
export interface MinKernelVersionCheck {
  min_kernel_version: string;
}
export interface OSVersionCheck {
  android?: MinVersionCheck;
  darwin?: MinVersionCheck;
  ios?: MinVersionCheck;
  linux?: MinKernelVersionCheck;
  windows?: MinKernelVersionCheck;
}
export interface Location {
  country_code: string;
  city_name?: string;
}
export interface GeoLocationCheck {
  locations: Location[];
  action: "allow" | "deny";
}
export interface PeerNetworkRangeCheck {
  ranges: string[];
  action: "allow" | "deny";
}
export interface Process {
  linux_path?: string;
  mac_path?: string;
  windows_path?: string;
}
export interface ProcessCheck {
  processes: Process[];
}
export interface Checks {
  nb_version_check?: MinVersionCheck;
  os_version_check?: OSVersionCheck;
  geo_location_check?: GeoLocationCheck;
  peer_network_range_check?: PeerNetworkRangeCheck;
  process_check?: ProcessCheck;
}

export interface PostureCheck {
  id: string;
  name: string;
  description: string;
  checks: Checks;
}

export interface PostureChecksResponse {
  // The server sends `null`, not `[]`, when the account has zero posture
  // checks — never assume this is a real array.
  posture_checks: PostureCheck[] | null;
  total: number;
}

export interface PostureCheckRequest {
  name: string;
  description: string;
  checks: Checks;
}

const keys = {
  all: ["posture-checks"] as const,
  list: () => ["posture-checks", "list"] as const,
  detail: (id: string) => ["posture-checks", "detail", id] as const,
};

export function usePostureChecks() {
  return useQuery({
    queryKey: keys.list(),
    queryFn: () => apiGet<PostureChecksResponse>("/posture-checks"),
  });
}

export function usePostureCheck(id: string) {
  return useQuery({
    queryKey: keys.detail(id),
    queryFn: () => apiGet<PostureCheck>(`/posture-checks/${encodeURIComponent(id)}`),
    enabled: id !== "",
  });
}

export function useCreatePostureCheck() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: PostureCheckRequest) => apiPost<PostureCheck>("/posture-checks", req),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.all }),
  });
}

export function useUpdatePostureCheck(id: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (req: PostureCheckRequest) => apiPut<PostureCheck>(`/posture-checks/${encodeURIComponent(id)}`, req),
    onSuccess: (check) => {
      queryClient.setQueryData(keys.detail(id), check);
      void queryClient.invalidateQueries({ queryKey: keys.all });
    },
  });
}

export function useDeletePostureCheck() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => apiDelete(`/posture-checks/${encodeURIComponent(id)}`),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: keys.all }),
  });
}
