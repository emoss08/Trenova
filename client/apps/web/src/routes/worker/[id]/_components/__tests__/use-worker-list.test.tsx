import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import type { Connection } from "@/lib/graphql/connection";
import { useWorkerList, workerListKey, type ConnectionPage } from "../use-worker-list";

type Row = { id: string };

function connection(ids: string[], endCursor: string | null): Connection<Row> {
  return {
    edges: ids.map((id) => ({ node: { id } })),
    pageInfo: { hasNextPage: endCursor !== null, endCursor },
    totalCount: null,
  };
}

/** Two pages: a, b then c; the second is held until the test lets it go. */
function twoPages() {
  let release: () => void = () => undefined;
  const fetchPage = vi.fn(async ({ after }: ConnectionPage) => {
    if (after === null) return connection(["a", "b"], "cursor-b");
    await new Promise<void>((done) => {
      release = done;
    });
    return connection(["c"], null);
  });
  return { fetchPage, release: () => release() };
}

function setup(fetchPage: (page: ConnectionPage) => Promise<Connection<Row>>) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  const hook = renderHook(
    () => useWorkerList({ root: "worker-things", workerId: "wrk_1", pageSize: 2, fetchPage }),
    { wrapper },
  );
  return { client, ...hook };
}

describe("useWorkerList", () => {
  it("reads the first page with the page size and the read's signal", async () => {
    const { fetchPage } = twoPages();
    const { result } = setup(fetchPage);

    await waitFor(() => expect(result.current.rows).toEqual([{ id: "a" }, { id: "b" }]));
    expect(fetchPage).toHaveBeenCalledWith({
      first: 2,
      after: null,
      signal: expect.any(AbortSignal),
    });
    expect(result.current.hasMore).toBe(true);
  });

  it("loads the next page once however often it is asked while it is in flight", async () => {
    const pages = twoPages();
    const { result } = setup(pages.fetchPage);
    await waitFor(() => expect(result.current.hasMore).toBe(true));

    act(() => result.current.loadMore());
    await waitFor(() => expect(result.current.loadingMore).toBe(true));
    act(() => result.current.loadMore());
    act(() => result.current.loadMore());
    await act(async () => pages.release());

    await waitFor(() => expect(result.current.rows.map((row) => row.id)).toEqual(["a", "b", "c"]));
    expect(pages.fetchPage).toHaveBeenCalledTimes(2);
    expect(result.current.hasMore).toBe(false);
  });

  it("keeps the same rows array between renders while the pages do not change", async () => {
    const { fetchPage } = twoPages();
    const { result, rerender } = setup(fetchPage);
    await waitFor(() => expect(result.current.rows).toHaveLength(2));

    const before = result.current.rows;
    rerender();

    expect(result.current.rows).toBe(before);
  });

  it("refetches every loaded page in place when the area is invalidated", async () => {
    const pages = twoPages();
    const { result, client } = setup(pages.fetchPage);
    await waitFor(() => expect(result.current.hasMore).toBe(true));
    act(() => result.current.loadMore());
    await waitFor(() => expect(result.current.loadingMore).toBe(true));
    await act(async () => pages.release());
    await waitFor(() => expect(result.current.rows).toHaveLength(3));
    pages.fetchPage.mockClear();
    pages.fetchPage.mockImplementation(async ({ after }: ConnectionPage) =>
      after === null ? connection(["a", "b"], "cursor-b") : connection(["c"], null),
    );

    await act(() => client.invalidateQueries({ queryKey: ["worker-things", "wrk_1"] }));

    expect(pages.fetchPage.mock.calls.map(([page]) => page.after)).toEqual([null, "cursor-b"]);
    expect(result.current.rows.map((row) => row.id)).toEqual(["a", "b", "c"]);
  });

  it("keys the read under the area's root and the worker, as the invalidation table does", () => {
    expect(workerListKey("worker-things", "wrk_1", ["open"])).toEqual([
      "worker-things",
      "wrk_1",
      "list",
      "open",
    ]);
  });
});
