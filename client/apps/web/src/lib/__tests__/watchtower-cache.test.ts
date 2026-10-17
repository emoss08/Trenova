import { watchtower, type WatchtowerQueuePages } from "@/lib/queries/watchtower";
import { watchtowerItem } from "@/routes/desk/_components/watchtower/__tests__/item-fixture";
import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";
import {
  belongsInQueue,
  cachedQueueItems,
  placeInQueue,
  shiftCleared,
  shiftQueueCounts,
} from "../watchtower-cache";

function cacheWith(pages: { ids: [string, number | null][]; hasNextPage: boolean }[]) {
  const queryClient = new QueryClient();
  queryClient.setQueryData<WatchtowerQueuePages>(watchtower.queue.queryKey, {
    pageParams: pages.map((_, index) => (index === 0 ? null : `cursor-${index}`)),
    pages: pages.map((page) => ({
      items: page.ids.map(([id, dueAt]) => watchtowerItem({ id, dueAt })),
      endCursor: page.hasNextPage ? "next" : null,
      hasNextPage: page.hasNextPage,
      seenAt: 0,
    })),
  });
  return queryClient;
}

const ids = (queryClient: QueryClient) => cachedQueueItems(queryClient).map((item) => item.id);

describe("belongsInQueue", () => {
  it("keeps only what is open and awake", () => {
    expect(belongsInQueue({ resolvedAt: null, status: "Open" })).toBe(true);
    expect(belongsInQueue({ resolvedAt: null, status: "InProgress" })).toBe(true);
    expect(belongsInQueue({ resolvedAt: null, status: "Snoozed" })).toBe(false);
    expect(belongsInQueue({ resolvedAt: 10, status: "Resolved" })).toBe(false);
  });
});

describe("placeInQueue", () => {
  it("puts a new signal where its deadline sorts", () => {
    const queryClient = cacheWith([
      {
        ids: [
          ["wt_a", 100],
          ["wt_c", 300],
          ["wt_fyi", null],
        ],
        hasNextPage: false,
      },
    ]);

    placeInQueue(queryClient, watchtowerItem({ id: "wt_b", dueAt: 200 }));

    expect(ids(queryClient)).toEqual(["wt_a", "wt_b", "wt_c", "wt_fyi"]);
  });

  it("moves a row whose deadline moved, and takes out one that was resolved or snoozed", () => {
    const queryClient = cacheWith([
      {
        ids: [
          ["wt_a", 100],
          ["wt_b", 200],
          ["wt_c", 300],
        ],
        hasNextPage: false,
      },
    ]);

    placeInQueue(queryClient, watchtowerItem({ id: "wt_a", dueAt: 400 }));
    expect(ids(queryClient)).toEqual(["wt_b", "wt_c", "wt_a"]);

    placeInQueue(queryClient, watchtowerItem({ id: "wt_b", status: "Snoozed" }));
    placeInQueue(queryClient, watchtowerItem({ id: "wt_c", resolvedAt: 10, status: "Resolved" }));
    expect(ids(queryClient)).toEqual(["wt_a"]);
  });

  it("leaves a row that sorts past what is loaded for the next page to bring", () => {
    const queryClient = cacheWith([
      {
        ids: [
          ["wt_a", 100],
          ["wt_b", 200],
        ],
        hasNextPage: true,
      },
    ]);

    placeInQueue(queryClient, watchtowerItem({ id: "wt_z", dueAt: 900 }));

    expect(ids(queryClient)).toEqual(["wt_a", "wt_b"]);
  });

  it("places a row in a later page when that is where it sorts", () => {
    const queryClient = cacheWith([
      {
        ids: [
          ["wt_a", 100],
          ["wt_b", 200],
        ],
        hasNextPage: true,
      },
      { ids: [["wt_d", 400]], hasNextPage: false },
    ]);

    placeInQueue(queryClient, watchtowerItem({ id: "wt_c", dueAt: 300 }));

    expect(ids(queryClient)).toEqual(["wt_a", "wt_b", "wt_c", "wt_d"]);
  });
});

describe("the counts the rail and the queue read", () => {
  it("move by one item, the critical count only for a critical one, never below zero", () => {
    const queryClient = new QueryClient();
    queryClient.setQueryData(watchtower.counts().queryKey, {
      active: 1,
      activeCritical: 0,
    });

    shiftQueueCounts(queryClient, { severity: "Critical" }, -1);
    shiftQueueCounts(queryClient, { severity: "Warning" }, -1);

    expect(queryClient.getQueryData(watchtower.counts().queryKey)).toEqual({
      active: 0,
      activeCritical: 0,
    });

    queryClient.setQueryData(watchtower.cleared(100).queryKey, 2);
    shiftCleared(queryClient, 100, 1);
    expect(queryClient.getQueryData(watchtower.cleared(100).queryKey)).toBe(3);
  });
});
