import { attention } from "@/lib/queries/attention";
import { watchtower, type WatchtowerQueuePages } from "@/lib/queries/watchtower";
import { watchtowerItem } from "@/routes/desk/_components/watchtower/__tests__/item-fixture";
import { QueryClient, QueryObserver } from "@tanstack/react-query";
import type { QueryKeyRoot } from "@trenova/shared/hooks/realtime-patching";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cachedQueueItems } from "../watchtower-cache";
import { createWatchtowerPatcher } from "../watchtower-realtime";

const observers: Array<() => void> = [];

function cachedQueueWith(...items: ReturnType<typeof watchtowerItem>[]) {
  const queryClient = new QueryClient();
  queryClient.setQueryData<WatchtowerQueuePages>(watchtower.queue.queryKey, {
    pageParams: [null],
    pages: [{ items, endCursor: null, hasNextPage: false, seenAt: 0 }],
  });
  return queryClient;
}

/** A queue the Watchtower page has on screen: cached and observed. */
function queueWith(...items: ReturnType<typeof watchtowerItem>[]) {
  const queryClient = cachedQueueWith(...items);
  const observer = new QueryObserver(queryClient, {
    queryKey: watchtower.queue.queryKey,
    queryFn: () => new Promise<never>(() => undefined),
    staleTime: Number.POSITIVE_INFINITY,
  });
  observers.push(observer.subscribe(() => undefined));
  return queryClient;
}

const invalidate = () => vi.fn<(roots: readonly QueryKeyRoot[], activeOnly: boolean) => void>();

const QUEUE_AND_SUMMARIES = [
  watchtower.queue.queryKey,
  watchtower.counts().queryKey,
  attention._def,
];
const SUMMARIES = [watchtower.counts().queryKey, attention._def];

afterEach(() => {
  observers.splice(0).forEach((unsubscribe) => unsubscribe());
});

const notice = (resource: string, recordId: string, action = "updated") => ({
  organizationId: "org_1",
  businessUnitId: "bu_1",
  resource,
  action,
  recordId,
});

describe("createWatchtowerPatcher", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("reads a new signal once and places it in the queue by its deadline", async () => {
    const queryClient = queueWith(watchtowerItem({ id: "wt_a", dueAt: 100 }));
    const fetchItem = vi.fn().mockResolvedValue(watchtowerItem({ id: "wt_new", dueAt: 50 }));
    const patcher = createWatchtowerPatcher({
      queryClient,
      invalidate: invalidate(),
      fetchItem,
      coalesceMs: 10,
    });

    patcher.handle(notice("watchtower", "wt_new", "created"));
    patcher.handle(notice("watchtower", "wt_new"));
    await vi.advanceTimersByTimeAsync(20);

    expect(fetchItem).toHaveBeenCalledTimes(1);
    expect(cachedQueueItems(queryClient).map((item) => item.id)).toEqual(["wt_new", "wt_a"]);
    patcher.dispose();
  });

  it("takes an item out once the reader can no longer read it", async () => {
    const queryClient = queueWith(watchtowerItem({ id: "wt_a" }));
    const fetchItem = vi.fn().mockRejectedValue(
      new GraphQLRequestError({
        kind: "graphql",
        message: "Watchtower item not found",
        status: 200,
        graphQLErrors: [
          {
            extensions: {
              code: "NOT_FOUND",
              type: "https://trenova.app/problems/resource-not-found",
            },
            code: "NOT_FOUND",
            message: "Watchtower item not found",
            type: "https://trenova.app/problems/resource-not-found",
          },
        ],
      }),
    );
    const patcher = createWatchtowerPatcher({
      queryClient,
      invalidate: invalidate(),
      fetchItem,
      coalesceMs: 10,
    });

    patcher.handle(notice("watchtower", "wt_a"));
    await vi.advanceTimersByTimeAsync(20);

    expect(cachedQueueItems(queryClient)).toEqual([]);
    patcher.dispose();
  });

  it("reads again the rows an agent run is working when the run moves", async () => {
    const working = watchtowerItem({
      id: "wt_a",
      agentWork: {
        agentId: "adef_1",
        runId: "ar_1",
        status: "Working",
        step: 0,
        steps: ["Read", "Decide", "Approve"],
        note: "",
        failed: false,
        handedAt: 1,
        agent: null,
      },
    });
    const queryClient = queueWith(working, watchtowerItem({ id: "wt_b" }));
    const advanced = { ...working, agentWork: { ...working.agentWork!, step: 1 } };
    const fetchItem = vi.fn().mockResolvedValue(advanced);
    const patcher = createWatchtowerPatcher({
      queryClient,
      invalidate: invalidate(),
      fetchItem,
      coalesceMs: 10,
    });

    patcher.handleRun(notice("agent_run", "ar_1"));
    patcher.handleRun(notice("shipments", "ar_1"));
    await vi.advanceTimersByTimeAsync(20);

    expect(fetchItem).toHaveBeenCalledExactlyOnceWith("wt_a");
    expect(cachedQueueItems(queryClient)[0].agentWork?.step).toBe(1);
    patcher.dispose();
  });

  it("keeps an item marked seen that the reader already opened", async () => {
    const queryClient = queueWith(watchtowerItem({ id: "wt_a", seen: true }));
    const fetchItem = vi.fn().mockResolvedValue(watchtowerItem({ id: "wt_a", seen: false }));
    const patcher = createWatchtowerPatcher({
      queryClient,
      invalidate: invalidate(),
      fetchItem,
      coalesceMs: 10,
    });

    patcher.handle(notice("watchtower", "wt_a"));
    await vi.advanceTimersByTimeAsync(20);

    expect(cachedQueueItems(queryClient)[0].seen).toBe(true);
    patcher.dispose();
  });
});

describe("a burst of notices", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("refreshes the counts and the attention summary once for every item read in a window", async () => {
    const queryClient = queueWith(watchtowerItem({ id: "wt_a" }), watchtowerItem({ id: "wt_b" }));
    const fetchItem = vi.fn((id: string) => Promise.resolve(watchtowerItem({ id })));
    const refresh = invalidate();
    const patcher = createWatchtowerPatcher({
      queryClient,
      invalidate: refresh,
      fetchItem,
      coalesceMs: 10,
    });

    patcher.handle(notice("watchtower", "wt_a"));
    patcher.handle(notice("watchtower", "wt_b"));
    await vi.advanceTimersByTimeAsync(20);

    expect(fetchItem.mock.calls.map(([id]) => id)).toEqual(["wt_a", "wt_b"]);
    expect(refresh.mock.calls).toEqual([[SUMMARIES, true]]);
    patcher.dispose();
  });

  it("reads the queue again in one request instead of every item past the limit", async () => {
    const queryClient = queueWith(watchtowerItem({ id: "wt_a" }));
    const fetchItem = vi.fn((id: string) => Promise.resolve(watchtowerItem({ id })));
    const refresh = invalidate();
    const patcher = createWatchtowerPatcher({
      queryClient,
      invalidate: refresh,
      fetchItem,
      coalesceMs: 10,
      burstLimit: 3,
    });

    for (const id of ["wt_1", "wt_2", "wt_3", "wt_4"]) {
      patcher.handle(notice("watchtower", id));
    }
    await vi.advanceTimersByTimeAsync(20);

    expect(fetchItem).not.toHaveBeenCalled();
    expect(refresh.mock.calls).toEqual([[QUEUE_AND_SUMMARIES, true]]);
    patcher.dispose();
  });

  it("reads nothing while the queue is off screen and only marks it stale", async () => {
    const queryClient = cachedQueueWith(watchtowerItem({ id: "wt_a" }));
    const fetchItem = vi.fn((id: string) => Promise.resolve(watchtowerItem({ id })));
    const refresh = invalidate();
    const patcher = createWatchtowerPatcher({
      queryClient,
      invalidate: refresh,
      fetchItem,
      coalesceMs: 10,
    });

    patcher.handle(notice("watchtower", "wt_a"));
    patcher.handle(notice("watchtower", "wt_b", "created"));
    await vi.advanceTimersByTimeAsync(20);

    expect(fetchItem).not.toHaveBeenCalled();
    expect(refresh.mock.calls).toEqual([[QUEUE_AND_SUMMARIES, true]]);
    patcher.dispose();
  });

  it("answers a sweep's bulk notice with one refresh of the queue and its summaries", () => {
    const queryClient = queueWith(watchtowerItem({ id: "wt_a" }));
    const fetchItem = vi.fn();
    const refresh = invalidate();
    const patcher = createWatchtowerPatcher({
      queryClient,
      invalidate: refresh,
      fetchItem,
      coalesceMs: 10,
    });

    patcher.handle({
      organizationId: "org_1",
      businessUnitId: "bu_1",
      resource: "watchtower",
      action: "bulk_swept",
    });

    expect(fetchItem).not.toHaveBeenCalled();
    expect(refresh.mock.calls).toEqual([[QUEUE_AND_SUMMARIES, true]]);
    patcher.dispose();
  });

  it("drops what is waiting once disposed", async () => {
    const queryClient = queueWith(watchtowerItem({ id: "wt_a" }));
    const fetchItem = vi.fn();
    const refresh = invalidate();
    const patcher = createWatchtowerPatcher({
      queryClient,
      invalidate: refresh,
      fetchItem,
      coalesceMs: 10,
    });

    patcher.handle(notice("watchtower", "wt_a"));
    patcher.dispose();
    await vi.advanceTimersByTimeAsync(20);

    expect(fetchItem).not.toHaveBeenCalled();
    expect(refresh).not.toHaveBeenCalled();
  });
});

describe("a notice", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("changes nothing until the item is read, because it names the item and nothing else", async () => {
    const queryClient = queueWith(watchtowerItem({ id: "wt_a", version: 3, title: "Before" }));
    const fetchItem = vi
      .fn()
      .mockResolvedValue(watchtowerItem({ id: "wt_a", version: 4, title: "After" }));
    const patcher = createWatchtowerPatcher({
      queryClient,
      invalidate: invalidate(),
      fetchItem,
      coalesceMs: 10,
    });

    patcher.handle({ ...notice("watchtower", "wt_a"), entity: { id: "wt_a" } });
    expect(cachedQueueItems(queryClient)[0].title).toBe("Before");

    await vi.advanceTimersByTimeAsync(20);
    expect(cachedQueueItems(queryClient)[0]).toMatchObject({ version: 4, title: "After" });
    patcher.dispose();
  });

  it("ignores a read that comes back older than the row", async () => {
    const queryClient = queueWith(watchtowerItem({ id: "wt_a", version: 7, title: "Current" }));
    const fetchItem = vi
      .fn()
      .mockResolvedValue(watchtowerItem({ id: "wt_a", version: 6, title: "Stale read" }));
    const patcher = createWatchtowerPatcher({
      queryClient,
      invalidate: invalidate(),
      fetchItem,
      coalesceMs: 10,
    });

    patcher.handle(notice("watchtower", "wt_a"));

    await vi.advanceTimersByTimeAsync(20);
    expect(cachedQueueItems(queryClient)[0]).toMatchObject({ version: 7, title: "Current" });
    patcher.dispose();
  });
});
