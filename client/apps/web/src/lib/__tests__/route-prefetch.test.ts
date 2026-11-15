import { queryClient } from "@/lib/query-client";
import {
  pageLoader,
  PREFETCH_STALE_TIME_MS,
  type RoutePrefetch,
  type RoutePrefetchQuery,
} from "@/lib/route-prefetch";
import { afterEach, describe, expect, it, vi } from "vitest";

// The router's loader context for a first visit to the shipments page with a panel open.
const args = {
  params: {},
  location: { searchStr: "?panelType=edit" },
  cause: "enter",
  context: { queryClient },
} as const;

// Lets the loader's fire-and-forget chain reach the query cache without awaiting it
// from the test the way the loader must never do.
async function flushMicrotasks(): Promise<void> {
  await new Promise((resolve) => setTimeout(resolve, 0));
}

async function untilSettled(queryKey: RoutePrefetchQuery["queryKey"]): Promise<void> {
  for (let attempt = 0; attempt < 50; attempt += 1) {
    const state = queryClient.getQueryState(queryKey);
    if (state && state.fetchStatus === "idle" && state.status !== "pending") {
      return;
    }
    await flushMicrotasks();
  }
  throw new Error(`query ${JSON.stringify(queryKey)} never settled`);
}

// The cache's Query type does not surface the observer options it holds, so the stored
// staleTime is read through the shape prefetchQuery was handed.
function storedStaleTime(queryKey: RoutePrefetchQuery["queryKey"]): number | undefined {
  const options: object | undefined = queryClient.getQueryCache().find({ queryKey })?.options;
  if (!options || !("staleTime" in options) || typeof options.staleTime !== "number") {
    return undefined;
  }
  return options.staleTime;
}

afterEach(() => {
  queryClient.clear();
});

describe("pageLoader", () => {
  it("returns before the queries it started have resolved", async () => {
    const queryKey = ["prefetch-test", "pending"] as const;
    const queryFn = vi.fn(() => new Promise<never>(() => undefined));

    expect(pageLoader(() => [{ queryKey, queryFn }])(args)).toBeUndefined();

    await flushMicrotasks();
    expect(queryFn).toHaveBeenCalledTimes(1);
    expect(queryClient.getQueryState(queryKey)?.fetchStatus).toBe("fetching");
  });

  it("hands the prefetch the path's params and the query, so it can read the panel asked for", async () => {
    const prefetch = vi.fn<RoutePrefetch>(() => []);

    pageLoader(prefetch)({ ...args, params: { id: "shp_1" } });
    await flushMicrotasks();

    const [received] = prefetch.mock.calls[0] ?? [];
    expect(received?.params).toEqual({ id: "shp_1" });
    expect(received?.searchParams.get("panelType")).toBe("edit");
  });

  it("accepts a prefetch that resolves its list asynchronously", async () => {
    const queryKey = ["prefetch-test", "async-list"] as const;
    const queryFn = vi.fn(async () => "data");

    pageLoader(async () => [{ queryKey, queryFn }])(args);
    await untilSettled(queryKey);

    expect(queryClient.getQueryData(queryKey)).toBe("data");
  });

  it("swallows a prefetch that throws synchronously", async () => {
    const loader = pageLoader(() => {
      throw new Error("no list for you");
    });

    expect(loader(args)).toBeUndefined();
    await flushMicrotasks();
  });

  it("swallows a prefetch that rejects", async () => {
    const loader = pageLoader(async () => {
      throw new Error("no list for you");
    });

    expect(loader(args)).toBeUndefined();
    await flushMicrotasks();
  });

  it("swallows a query that fails and leaves the failure for the page to surface", async () => {
    const queryKey = ["prefetch-test", "failing"] as const;
    const failure = new Error("server said no");
    const queryFn = vi.fn(async () => {
      throw failure;
    });

    expect(pageLoader(() => [{ queryKey, queryFn }])(args)).toBeUndefined();
    await untilSettled(queryKey);

    expect(queryClient.getQueryState(queryKey)?.error).toBe(failure);
  });

  it("gives each query a staleTime so the page's mount does not refetch it", async () => {
    const queryKey = ["prefetch-test", "stale-time"] as const;
    const queryFn = vi.fn(async () => "data");
    const loader = pageLoader(() => [{ queryKey, queryFn }]);

    loader(args);
    await untilSettled(queryKey);

    expect(storedStaleTime(queryKey)).toBe(PREFETCH_STALE_TIME_MS);

    loader(args);
    await flushMicrotasks();
    expect(queryFn).toHaveBeenCalledTimes(1);
  });

  it("keeps a longer staleTime the page declared for the query", async () => {
    const queryKey = ["prefetch-test", "page-stale-time"] as const;

    pageLoader(() => [{ queryKey, queryFn: async () => "data", staleTime: Infinity }])(args);
    await untilSettled(queryKey);

    expect(storedStaleTime(queryKey)).toBe(Infinity);
  });

  it("starts every query in the list, not only the first", async () => {
    const first = ["prefetch-test", "first"] as const;
    const second = ["prefetch-test", "second"] as const;

    pageLoader(() => [
      { queryKey: first, queryFn: async () => 1 },
      { queryKey: second, queryFn: async () => 2 },
    ])(args);
    await untilSettled(first);
    await untilSettled(second);

    expect(queryClient.getQueryData(first)).toBe(1);
    expect(queryClient.getQueryData(second)).toBe(2);
  });

  it("warms nothing when the navigation only changes the page's query", async () => {
    const prefetch = vi.fn<RoutePrefetch>(() => []);

    pageLoader(prefetch)({ ...args, cause: "stay" });
    await flushMicrotasks();

    expect(prefetch).not.toHaveBeenCalled();
  });
});

/**
 * An infinite query stores `{ pages, pageParams }` where a plain one stores the
 * payload itself. Warming one through prefetchQuery seeds a cache entry the
 * page then cannot read — its first render throws on `data.pages` — so the
 * paged reports library only survives its route's prefetch if `warm` reaches
 * for prefetchInfiniteQuery instead.
 */
describe("pageLoader with a paged query", () => {
  const infiniteEntry = (
    queryKey: readonly unknown[],
    queryFn: (context: { pageParam?: unknown }) => unknown,
  ) => ({
    queryKey,
    queryFn,
    initialPageParam: null as string | null,
    getNextPageParam: () => undefined,
  });

  it("stores a warmed page under the shape an infinite query reads", async () => {
    const queryKey = ["prefetch-test", "paged"] as const;
    const queryFn = vi.fn(() => Promise.resolve({ rows: ["a"] }));

    pageLoader(() => [infiniteEntry(queryKey, queryFn)])(args);
    await untilSettled(queryKey);

    expect(queryClient.getQueryData(queryKey)).toEqual({
      pages: [{ rows: ["a"] }],
      pageParams: [null],
    });
  });

  it("hands the first page param to the query function", async () => {
    const queryKey = ["prefetch-test", "paged-param"] as const;
    const queryFn = vi.fn((context: { pageParam?: unknown }) => {
      void context;
      return Promise.resolve({ rows: [] });
    });

    pageLoader(() => [infiniteEntry(queryKey, queryFn)])(args);
    await untilSettled(queryKey);

    expect(queryFn.mock.calls[0]?.[0]).toMatchObject({ pageParam: null });
  });

  it("still warms a plain query as a plain query", async () => {
    const queryKey = ["prefetch-test", "plain"] as const;
    const queryFn = vi.fn(() => Promise.resolve({ rows: ["a"] }));

    pageLoader(() => [{ queryKey, queryFn }])(args);
    await untilSettled(queryKey);

    expect(queryClient.getQueryData(queryKey)).toEqual({ rows: ["a"] });
  });

  it("gives a paged query the same default staleTime", async () => {
    const queryKey = ["prefetch-test", "paged-stale"] as const;

    pageLoader(() => [infiniteEntry(queryKey, () => Promise.resolve({ rows: [] }))])(args);
    await untilSettled(queryKey);

    expect(storedStaleTime(queryKey)).toBe(PREFETCH_STALE_TIME_MS);
  });
});
