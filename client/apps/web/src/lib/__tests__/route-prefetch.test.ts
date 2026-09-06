import { queryClient } from "@/lib/query-client";
import { combineLoaders } from "@/lib/route-permission";
import {
  createPrefetchLoader,
  lazyPrefetch,
  PREFETCH_STALE_TIME_MS,
  type RoutePrefetch,
  type RoutePrefetchQuery,
} from "@/lib/route-prefetch";
import { redirect, type LoaderFunctionArgs } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";

const args = {
  request: new Request("http://localhost/shipment-management/shipments?panelType=edit"),
  params: {},
  context: undefined,
} as unknown as LoaderFunctionArgs;

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

describe("createPrefetchLoader", () => {
  it("returns null before the queries it started have resolved", async () => {
    const queryKey = ["prefetch-test", "pending"] as const;
    const queryFn = vi.fn(() => new Promise<never>(() => undefined));

    const result = await createPrefetchLoader(() => [{ queryKey, queryFn }])(args);
    expect(result).toBeNull();

    await flushMicrotasks();
    expect(queryFn).toHaveBeenCalledTimes(1);
    expect(queryClient.getQueryState(queryKey)?.fetchStatus).toBe("fetching");
  });

  it("hands the loader arguments to the prefetch so it can read the request", async () => {
    const prefetch = vi.fn<RoutePrefetch>(() => []);

    await createPrefetchLoader(prefetch)(args);
    await flushMicrotasks();

    expect(prefetch).toHaveBeenCalledWith(args);
  });

  it("accepts a prefetch that resolves its list asynchronously", async () => {
    const queryKey = ["prefetch-test", "async-list"] as const;
    const queryFn = vi.fn(async () => "data");

    await createPrefetchLoader(async () => [{ queryKey, queryFn }])(args);
    await untilSettled(queryKey);

    expect(queryClient.getQueryData(queryKey)).toBe("data");
  });

  it("swallows a prefetch that throws synchronously", async () => {
    const loader = createPrefetchLoader(() => {
      throw new Error("no list for you");
    });

    expect(await loader(args)).toBeNull();
    await flushMicrotasks();
  });

  it("swallows a prefetch that rejects", async () => {
    const loader = createPrefetchLoader(async () => {
      throw new Error("no list for you");
    });

    expect(await loader(args)).toBeNull();
    await flushMicrotasks();
  });

  it("swallows a query that fails and leaves the failure for the page to surface", async () => {
    const queryKey = ["prefetch-test", "failing"] as const;
    const failure = new Error("server said no");
    const queryFn = vi.fn(async () => {
      throw failure;
    });

    expect(await createPrefetchLoader(() => [{ queryKey, queryFn }])(args)).toBeNull();
    await untilSettled(queryKey);

    expect(queryClient.getQueryState(queryKey)?.error).toBe(failure);
  });

  it("gives each query a staleTime so the page's mount does not refetch it", async () => {
    const queryKey = ["prefetch-test", "stale-time"] as const;
    const queryFn = vi.fn(async () => "data");
    const loader = createPrefetchLoader(() => [{ queryKey, queryFn }]);

    await loader(args);
    await untilSettled(queryKey);

    expect(storedStaleTime(queryKey)).toBe(PREFETCH_STALE_TIME_MS);

    await loader(args);
    await flushMicrotasks();
    expect(queryFn).toHaveBeenCalledTimes(1);
  });

  it("keeps a longer staleTime the page declared for the query", async () => {
    const queryKey = ["prefetch-test", "page-stale-time"] as const;

    await createPrefetchLoader(() => [
      { queryKey, queryFn: async () => "data", staleTime: Infinity },
    ])(args);
    await untilSettled(queryKey);

    expect(storedStaleTime(queryKey)).toBe(Infinity);
  });

  it("starts every query in the list, not only the first", async () => {
    const first = ["prefetch-test", "first"] as const;
    const second = ["prefetch-test", "second"] as const;

    await createPrefetchLoader(() => [
      { queryKey: first, queryFn: async () => 1 },
      { queryKey: second, queryFn: async () => 2 },
    ])(args);
    await untilSettled(first);
    await untilSettled(second);

    expect(queryClient.getQueryData(first)).toBe(1);
    expect(queryClient.getQueryData(second)).toBe(2);
  });

  it("never runs when an earlier loader in the chain redirects", async () => {
    const prefetch = vi.fn<RoutePrefetch>(() => []);
    const loader = combineLoaders(async () => redirect("/login"), createPrefetchLoader(prefetch));

    const result = await loader(args);
    await flushMicrotasks();

    expect(result).toBeInstanceOf(Response);
    expect(prefetch).not.toHaveBeenCalled();
  });
});

describe("lazyPrefetch", () => {
  it("returns an empty list for a page module without a prefetch export", async () => {
    const importer = vi.fn(async () => ({ Page: () => null }));

    await expect(lazyPrefetch(importer)(args)).resolves.toEqual([]);
    expect(importer).toHaveBeenCalledTimes(1);
  });

  it("runs the page module's prefetch with the loader arguments", async () => {
    const list: RoutePrefetchQuery[] = [
      { queryKey: ["prefetch-test", "lazy"], queryFn: async () => "data" },
    ];
    const prefetch = vi.fn<RoutePrefetch>(() => list);

    await expect(lazyPrefetch(async () => ({ prefetch }))(args)).resolves.toBe(list);
    expect(prefetch).toHaveBeenCalledWith(args);
  });

  it("surfaces a failed module import to the loader, which swallows it", async () => {
    const loader = createPrefetchLoader(
      lazyPrefetch(async () => {
        throw new Error("chunk load failed");
      }),
    );

    expect(await loader(args)).toBeNull();
    await flushMicrotasks();
  });
});
