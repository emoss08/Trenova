import { watchtower, type WatchtowerQueuePages } from "@/lib/queries/watchtower";
import { watchtowerItem } from "@/routes/desk/_components/watchtower/__tests__/item-fixture";
import { QueryClient } from "@tanstack/react-query";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cachedQueueItems } from "../watchtower-cache";
import { createWatchtowerPatcher } from "../watchtower-realtime";

function queueWith(...items: ReturnType<typeof watchtowerItem>[]) {
  const queryClient = new QueryClient();
  queryClient.setQueryData<WatchtowerQueuePages>(watchtower.queue.queryKey, {
    pageParams: [null],
    pages: [{ items, endCursor: null, hasNextPage: false, seenAt: 0 }],
  });
  return queryClient;
}

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
    const patcher = createWatchtowerPatcher({ queryClient, fetchItem, coalesceMs: 10 });

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
    const patcher = createWatchtowerPatcher({ queryClient, fetchItem, coalesceMs: 10 });

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
    const patcher = createWatchtowerPatcher({ queryClient, fetchItem, coalesceMs: 10 });

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
    const patcher = createWatchtowerPatcher({ queryClient, fetchItem, coalesceMs: 10 });

    patcher.handle(notice("watchtower", "wt_a"));
    await vi.advanceTimersByTimeAsync(20);

    expect(cachedQueueItems(queryClient)[0].seen).toBe(true);
    patcher.dispose();
  });
});

describe("a notice's entity", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("moves the row's version at once, before the item is read again", () => {
    const queryClient = queueWith(watchtowerItem({ id: "wt_a", version: 3 }));
    const fetchItem = vi.fn(() => new Promise<never>(() => undefined));
    const patcher = createWatchtowerPatcher({ queryClient, fetchItem, coalesceMs: 10 });

    patcher.handle({
      ...notice("watchtower", "wt_a"),
      entity: { id: "wt_a", version: 4, status: "Open", title: "Now five hours late" },
    });

    const [row] = cachedQueueItems(queryClient);
    expect(row.version).toBe(4);
    expect(row.title).toBe("Now five hours late");
    patcher.dispose();
  });

  it("takes out at once an item the notice says is resolved or snoozed", () => {
    const queryClient = queueWith(watchtowerItem({ id: "wt_a" }), watchtowerItem({ id: "wt_b" }));
    const fetchItem = vi.fn(() => new Promise<never>(() => undefined));
    const patcher = createWatchtowerPatcher({ queryClient, fetchItem, coalesceMs: 10 });

    patcher.handle({
      ...notice("watchtower", "wt_a"),
      entity: { id: "wt_a", version: 4, status: "Resolved", resolvedAt: 100 },
    });
    patcher.handle({
      ...notice("watchtower", "wt_b"),
      entity: { id: "wt_b", version: 4, status: "Snoozed", snoozedUntil: 200 },
    });

    expect(cachedQueueItems(queryClient)).toEqual([]);
    patcher.dispose();
  });

  it("ignores a notice older than the row, and a read that comes back older", async () => {
    const queryClient = queueWith(watchtowerItem({ id: "wt_a", version: 7, title: "Current" }));
    const fetchItem = vi
      .fn()
      .mockResolvedValue(watchtowerItem({ id: "wt_a", version: 6, title: "Stale read" }));
    const patcher = createWatchtowerPatcher({ queryClient, fetchItem, coalesceMs: 10 });

    patcher.handle({
      ...notice("watchtower", "wt_a"),
      entity: { id: "wt_a", version: 5, status: "Open", title: "Stale notice" },
    });
    expect(cachedQueueItems(queryClient)[0].title).toBe("Current");

    await vi.advanceTimersByTimeAsync(20);
    expect(cachedQueueItems(queryClient)[0]).toMatchObject({ version: 7, title: "Current" });
    patcher.dispose();
  });
});
