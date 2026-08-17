import { useQuery } from "@tanstack/react-query";
import { apiGet } from "./client";

// Mirrors nbapi.Event's JSON shape (internal/nbapi/events.go).
export interface AuditEvent {
  id: string;
  timestamp: string;
  activity: string;
  activity_code: string;
  initiator_id: string;
  initiator_name: string;
  initiator_email: string;
  target_id: string;
  meta: Record<string, string>;
}

export interface EventsResponse {
  events: AuditEvent[];
  total: number;
}

export function useAuditEvents() {
  return useQuery({
    queryKey: ["events", "audit"] as const,
    queryFn: () => apiGet<EventsResponse>("/events/audit"),
  });
}
