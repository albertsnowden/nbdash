import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PermissionsProvider } from "@/contexts/PermissionsContext";
import { FULL_ACCESS_PERMISSIONS_RESPONSE } from "@/test-fixtures/permissions";
import PeersList from "./PeersList";

function renderWithProviders(ui: ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <PermissionsProvider>
        <MemoryRouter>{ui}</MemoryRouter>
      </PermissionsProvider>
    </QueryClientProvider>,
  );
}

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

describe("PeersList", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = typeof input === "string" ? input : input.toString();
        if (url.includes("/permissions")) return jsonResponse(FULL_ACCESS_PERMISSIONS_RESPONSE);
        return jsonResponse({
          peers: [
            {
              id: "p1",
              name: "gateway-01",
              ip: "100.92.0.1",
              hostname: "gw01",
              connected: true,
              os: "Ubuntu 22.04",
              last_seen: "2026-08-15T00:00:00Z",
            },
          ],
          total: 1,
          connected_count: 1,
        });
      }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders peers returned from the API", async () => {
    renderWithProviders(<PeersList />);

    await waitFor(() => expect(screen.getByText("gateway-01")).toBeInTheDocument());
    expect(screen.getByText("1 of 1 peer connected")).toBeInTheDocument();
    expect(screen.getByText("100.92.0.1")).toBeInTheDocument();
  });

  it("shows an empty state when no peers match", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = typeof input === "string" ? input : input.toString();
        if (url.includes("/permissions")) return jsonResponse(FULL_ACCESS_PERMISSIONS_RESPONSE);
        return jsonResponse({ peers: [], total: 0, connected_count: 0 });
      }),
    );

    renderWithProviders(<PeersList />);

    await waitFor(() =>
      expect(screen.getByText("No peers match your search.")).toBeInTheDocument(),
    );
  });
});
