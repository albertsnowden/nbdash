import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { PermissionsProvider } from "@/contexts/PermissionsContext";
import { FULL_ACCESS_PERMISSIONS_RESPONSE } from "@/test-fixtures/permissions";
import GroupsList from "./GroupsList";

function renderWithProviders(ui: ReactElement) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <PermissionsProvider>
        <MemoryRouter>{ui}</MemoryRouter>
      </PermissionsProvider>
    </QueryClientProvider>,
  );
}

describe("GroupsList", () => {
  beforeEach(() => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = typeof input === "string" ? input : input.toString();
        if (url.includes("/permissions")) {
          return new Response(JSON.stringify(FULL_ACCESS_PERMISSIONS_RESPONSE), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          });
        }
        return new Response(
          JSON.stringify({
            groups: [
              { id: "g1", name: "All", peers_count: 2, resources_count: 0, peers: [] },
              { id: "g2", name: "devs", peers_count: 1, resources_count: 0, peers: [] },
            ],
            total: 2,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }),
    );
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders groups and hides delete for the protected All group", async () => {
    renderWithProviders(<GroupsList />);

    await waitFor(() => expect(screen.getByText("devs")).toBeInTheDocument());
    expect(screen.getByText("2 groups")).toBeInTheDocument();

    const allRow = screen.getByText("All").closest("tr")!;
    const devsRow = screen.getByText("devs").closest("tr")!;
    expect(allRow.querySelector("button")).toBeNull();
    expect(devsRow.querySelector("button")).not.toBeNull();
  });
});
