import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { useRoleOptions } from "../use-role-options";

const { fetchGraphQLSelectOptions } = vi.hoisted(() => ({ fetchGraphQLSelectOptions: vi.fn() }));

vi.mock("@/lib/graphql/select-options", () => ({ fetchGraphQLSelectOptions }));

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("useRoleOptions", () => {
  it("reads every role in one ROLE select-options page and names each by its label", async () => {
    fetchGraphQLSelectOptions.mockResolvedValue({
      count: 2,
      next: null,
      prev: null,
      results: [
        { id: "rol_dispatch", label: "Dispatcher", description: "Plans loads", meta: null },
        { id: "rol_billing", label: "Billing clerk", description: null, meta: null },
      ],
    });

    const { result } = renderHook(() => useRoleOptions(), { wrapper });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(result.current.data).toEqual([
      { id: "rol_dispatch", name: "Dispatcher" },
      { id: "rol_billing", name: "Billing clerk" },
    ]);
    expect(fetchGraphQLSelectOptions).toHaveBeenCalledWith(
      { resource: "ROLE", initialLimit: 100 },
      expect.objectContaining({ signal: expect.any(AbortSignal) }),
    );
  });
});
