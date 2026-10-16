import type { GlobalSearchGroup, GlobalSearchResponse } from "@/services/global-search";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { PaletteScope } from "../palette-model";

const { globalSearch } = vi.hoisted(() => ({ globalSearch: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { globalSearchService: { search: globalSearch } },
}));

const { usePaletteRemoteSearch } = await import("../use-palette-data");

function group(id: string, title: string): GlobalSearchGroup {
  return {
    entityType: "customer",
    label: "Customers",
    total: 1,
    hits: [{ id, entityType: "customer", title, href: `/customers/${id}` }],
  };
}

const ACME = [group("cus_1", "Acme Freight")];
const ACME_MILLS = [group("cus_2", "Acme Mills")];

/** Each search waits until the test answers it, so the hook can be read while one is in flight. */
function answerSearchesByHand() {
  const pending = new Map<string, (response: GlobalSearchResponse) => void>();
  globalSearch.mockImplementation(
    (query: string) =>
      new Promise<GlobalSearchResponse>((resolve) => {
        pending.set(query, resolve);
      }),
  );
  return (query: string, groups: GlobalSearchGroup[]) =>
    act(async () => {
      pending.get(query)?.({ query, groups });
    });
}

function renderSearch(query: string, scope: PaletteScope = "all") {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderHook(
    (props: { query: string; scope: PaletteScope }) =>
      usePaletteRemoteSearch({ enabled: true, ...props }),
    {
      initialProps: { query, scope },
      wrapper: ({ children }) => (
        <QueryClientProvider client={client}>{children}</QueryClientProvider>
      ),
    },
  );
}

beforeEach(() => {
  globalSearch.mockReset();
});

describe("usePaletteRemoteSearch", () => {
  it("is loading only while there is nothing yet to show for the first search", async () => {
    const answer = answerSearchesByHand();
    const { result } = renderSearch("acme");

    expect(result.current).toMatchObject({ loading: true, refreshing: false, groups: [] });

    await answer("acme", ACME);

    await waitFor(() =>
      expect(result.current).toMatchObject({
        loading: false,
        refreshing: false,
        groups: ACME,
        query: "acme",
      }),
    );
  });

  it("keeps the last answer, counted and not loading, while the next search is in flight", async () => {
    const answer = answerSearchesByHand();
    const { result, rerender } = renderSearch("acme");
    await answer("acme", ACME);
    await waitFor(() => expect(result.current.groups).toBe(ACME));

    rerender({ query: "acme m", scope: "all" });

    expect(result.current).toMatchObject({
      loading: false,
      error: false,
      refreshing: true,
      groups: ACME,
      query: "acme",
    });

    await answer("acme m", ACME_MILLS);

    await waitFor(() =>
      expect(result.current).toMatchObject({
        refreshing: false,
        groups: ACME_MILLS,
        query: "acme m",
      }),
    );
  });

  it("does not carry an answer over to another scope", async () => {
    const answer = answerSearchesByHand();
    const { result, rerender } = renderSearch("acme");
    await answer("acme", ACME);
    await waitFor(() => expect(result.current.groups).toBe(ACME));

    rerender({ query: "acme", scope: "customer" });

    expect(result.current).toMatchObject({
      loading: true,
      refreshing: false,
      groups: [],
      query: "",
    });
  });

  it("hands back the same state until something about the search changes", async () => {
    const answer = answerSearchesByHand();
    const { result, rerender } = renderSearch("acme");
    await answer("acme", ACME);
    await waitFor(() => expect(result.current.groups).toBe(ACME));
    const settled = result.current;

    rerender({ query: "acme", scope: "all" });

    expect(result.current).toBe(settled);
  });

  it("searches nothing and shows nothing below the shortest useful query", () => {
    const { result } = renderSearch("a");

    expect(result.current).toMatchObject({
      ready: false,
      loading: false,
      refreshing: false,
      groups: [],
      query: "",
    });
    expect(globalSearch).not.toHaveBeenCalled();
  });
});
