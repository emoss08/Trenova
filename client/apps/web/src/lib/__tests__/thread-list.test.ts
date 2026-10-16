import { assistant } from "@/lib/queries/assistant";
import type { AssistantThread, AssistantThreadPage } from "@/types/assistant";
import { InfiniteQueryObserver, QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { waitFor } from "@testing-library/react";
import type { ThreadListFilter } from "@/services/assistant";
import {
  cachedThread,
  cachedThreadLists,
  cachedThreads,
  countCachedThread,
  findCachedThread,
  noteThreadsInView,
  patchCachedThread,
  removeCachedThread,
  takeCachedThread,
  THREAD_PAGE_SIZE,
  threadListQuery,
  threadShelfCountsQuery,
  threadShelfQuery,
  threadsOf,
  updateCachedThread,
  type ThreadPages,
} from "../thread-list";

const listThreads = vi.hoisted(() =>
  vi.fn<
    (options: {
      limit?: number;
      cursor?: string;
      until?: string;
      filter?: ThreadListFilter;
      signal?: AbortSignal;
    }) => Promise<unknown>
  >(),
);

const countThreadShelves = vi.hoisted(() =>
  vi.fn<(edges: readonly number[], options: { signal?: AbortSignal }) => Promise<unknown>>(),
);

vi.mock("@/services/api", () => ({
  apiService: { assistantService: { listThreads, countThreadShelves } },
}));

function thread(id: string, overrides: Partial<AssistantThread> = {}): AssistantThread {
  return {
    id,
    businessUnitId: "bu",
    organizationId: "org",
    userId: "u",
    agentDefinitionId: "agtd_1",
    preferredProviderId: "",
    origin: "Desk",
    pinned: false,
    subjectType: "",
    subjectId: "",
    canContinue: true,
    title: id,
    status: "Active",
    lastMessageAt: 100,
    version: 0,
    createdAt: 50,
    updatedAt: 100,
    ...overrides,
  };
}

function pages(...items: AssistantThread[][]): ThreadPages {
  const built: AssistantThreadPage[] = items.map((page, index) => ({
    items: page,
    nextCursor: index < items.length - 1 ? `c${index + 1}` : "",
  }));

  return { pages: built, pageParams: built.map((_, index) => (index === 0 ? "" : `c${index}`)) };
}

let queryClient: QueryClient;

beforeEach(() => {
  listThreads.mockReset();
  queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
});

/**
 * The person's conversations are read a page at a time; every reader of the
 * list sees every page read so far, in order, each conversation once.
 */
describe("threadsOf", () => {
  it("reads every page in order", () => {
    const data = pages([thread("a"), thread("b")], [thread("c")]);

    expect(threadsOf(data).map((item) => item.id)).toEqual(["a", "b", "c"]);
  });

  // A conversation touched between two page reads can be handed back twice;
  // the rail keys its rows by conversation, so it must appear once.
  it("keeps the first of a conversation two pages both hold", () => {
    const data = pages([thread("a"), thread("b", { title: "first" })], [thread("b"), thread("c")]);

    const threads = threadsOf(data);
    expect(threads.map((item) => item.id)).toEqual(["a", "b", "c"]);
    expect(threads[1].title).toBe("first");
  });

  it("reads one array per cache entry, and nothing when there is none", () => {
    const data = pages([thread("a")]);

    expect(threadsOf(data)).toBe(threadsOf(data));
    expect(threadsOf(undefined)).toEqual([]);
  });
});

describe("threadListQuery", () => {
  it("asks for the first page without a cursor and each next one with the last cursor", async () => {
    listThreads
      .mockResolvedValueOnce({ items: [thread("a")], nextCursor: "c1" })
      .mockResolvedValueOnce({ items: [thread("b")], nextCursor: "" });

    await queryClient.fetchInfiniteQuery({ ...threadListQuery(), pages: 2 });

    expect(listThreads).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ cursor: "", limit: THREAD_PAGE_SIZE }),
    );
    expect(listThreads).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ cursor: "c1", limit: THREAD_PAGE_SIZE }),
    );
    expect(cachedThreads(queryClient).map((item) => item.id)).toEqual(["a", "b"]);
  });

  it("has no next page once a page comes back without a cursor", () => {
    const { getNextPageParam } = threadListQuery();

    expect(getNextPageParam({ items: [], nextCursor: "" }, [], "", [])).toBeUndefined();
    expect(getNextPageParam({ items: [], nextCursor: "c9" }, [], "", [])).toBe("c9");
  });
});

describe("updateCachedThread", () => {
  it("rewrites the conversation on whichever page holds it", () => {
    queryClient.setQueryData(assistant.threads().queryKey, pages([thread("a")], [thread("b")]));
    const before = queryClient.getQueryData<ThreadPages>(assistant.threads().queryKey);

    updateCachedThread(queryClient, "b", (item) => ({ ...item, title: "renamed" }));

    const after = queryClient.getQueryData<ThreadPages>(assistant.threads().queryKey);
    expect(after?.pages[1].items[0].title).toBe("renamed");
    expect(after?.pages[0]).toBe(before?.pages[0]);
  });

  it("leaves the cache as it was when no page holds the conversation", () => {
    const data = pages([thread("a")]);
    queryClient.setQueryData(assistant.threads().queryKey, data);

    updateCachedThread(queryClient, "missing", (item) => ({ ...item, title: "x" }));

    expect(queryClient.getQueryData(assistant.threads().queryKey)).toBe(data);
  });

  it("does nothing before the list has been read", () => {
    updateCachedThread(queryClient, "a", (item) => ({ ...item, title: "x" }));

    expect(queryClient.getQueryData(assistant.threads().queryKey)).toBeUndefined();
  });
});

/**
 * A person far down the rail has many pages read. A refresh reads the first
 * page and the pages the rail is drawing, and leaves the rest until they come
 * into view, so a reply landing does not download the whole list again.
 */
describe("refreshing a long list", () => {
  let server: AssistantThread[] = [];
  const key = assistant.threads().queryKey;
  const id = (index: number) => `t${String(index).padStart(3, "0")}`;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Number.POSITIVE_INFINITY } },
    });
    server = Array.from({ length: 160 }, (_, index) => thread(id(index)));
    // The server pages by cursor, or reads one page's range again with until.
    listThreads.mockImplementation(({ cursor = "", until, limit = THREAD_PAGE_SIZE }) => {
      const start = cursor === "" ? 0 : server.findIndex((item) => item.id === cursor) + 1;
      if (until) {
        const end = server.findIndex((item) => item.id === until);
        return Promise.resolve({ items: server.slice(start, end + 1), nextCursor: until });
      }
      const items = server.slice(start, start + limit);
      const more = start + limit < server.length;
      return Promise.resolve({ items, nextCursor: more ? (items.at(-1)?.id ?? "") : "" });
    });
  });

  async function readThreePages() {
    await queryClient.fetchInfiniteQuery({ ...threadListQuery(), pages: 3 });
    const observer = new InfiniteQueryObserver(queryClient, threadListQuery());
    const stop = observer.subscribe(() => undefined);
    listThreads.mockClear();
    return stop;
  }

  const calls = () =>
    listThreads.mock.calls.map(([options]) => ({
      cursor: options.cursor ?? "",
      until: options.until ?? "",
    }));

  it("reads the first page and the page in view, and not the one between", async () => {
    const stop = await readThreePages();
    noteThreadsInView(queryClient, [id(120)]);

    await queryClient.refetchQueries({ queryKey: key, exact: true });

    expect(calls()).toEqual([
      { cursor: "", until: id(49) },
      { cursor: id(99), until: id(149) },
    ]);
    expect(cachedThreads(queryClient)).toHaveLength(150);
    stop();
  });

  it("reads a page left behind once one of its conversations comes into view", async () => {
    const stop = await readThreePages();
    noteThreadsInView(queryClient, [id(120)]);
    await queryClient.refetchQueries({ queryKey: key, exact: true });
    listThreads.mockClear();

    noteThreadsInView(queryClient, [id(60), id(61)]);

    await waitFor(() => expect(calls()).toEqual([{ cursor: id(49), until: id(99) }]));
    await waitFor(() => expect(queryClient.isFetching({ queryKey: key })).toBe(0));
    listThreads.mockClear();

    // Read now: scrolling within it asks for nothing more.
    noteThreadsInView(queryClient, [id(62)]);
    await Promise.resolve();
    expect(calls()).toEqual([]);
    stop();
  });

  it("shows a conversation that moved to the top once, from the page read last", async () => {
    const stop = await readThreePages();
    const moved = { ...server[130], title: "Moved up" };
    server = [moved, ...server.filter((item) => item.id !== moved.id)];

    await queryClient.refetchQueries({ queryKey: key, exact: true });

    const threads = cachedThreads(queryClient);
    expect(threads[0]).toMatchObject({ id: id(130), title: "Moved up" });
    expect(threads.filter((item) => item.id === id(130))).toHaveLength(1);
    expect(calls()).toEqual([{ cursor: "", until: id(49) }]);
    stop();
  });

  it("keeps each page's boundary, so the next page still continues where it did", async () => {
    const stop = await readThreePages();
    server = [thread("new"), ...server];

    await queryClient.refetchQueries({ queryKey: key, exact: true });
    const pages = queryClient.getQueryData<ThreadPages>(key);

    expect(pages?.pages[0].items).toHaveLength(51);
    expect(pages?.pages[0].nextCursor).toBe(id(49));
    expect(pages?.pageParams).toEqual(["", id(49), id(99)]);
    stop();
  });

  it("takes a deleted conversation out of every page at once", async () => {
    const stop = await readThreePages();

    removeCachedThread(queryClient, id(120));

    expect(cachedThreads(queryClient).some((item) => item.id === id(120))).toBe(false);
    expect(cachedThreads(queryClient)).toHaveLength(149);
    stop();
  });
});

/**
 * Each shelf of the rail is a list of its own, read from the server only for
 * its range. It sits under the list's key, so whatever refreshes, rewrites or
 * empties the list does the same to every shelf, and leaves the counts kept
 * beside them alone.
 */
describe("shelves", () => {
  const today: ThreadListFilter = { pinned: false, touchedFrom: 1_000 };
  const older: ThreadListFilter = { pinned: false, touchedBefore: 500 };
  const shelfKey = (filter: ThreadListFilter) => assistant.threads()._ctx.shelf(filter).queryKey;

  it("reads a shelf's pages with its range, under the list's key", async () => {
    listThreads
      .mockResolvedValueOnce({ items: [thread("a")], nextCursor: "c1" })
      .mockResolvedValueOnce({ items: [thread("b")], nextCursor: "" });

    await queryClient.fetchInfiniteQuery({ ...threadShelfQuery(today), pages: 2 });

    expect(listThreads).toHaveBeenNthCalledWith(
      1,
      expect.objectContaining({ cursor: "", filter: today, limit: THREAD_PAGE_SIZE }),
    );
    expect(listThreads).toHaveBeenNthCalledWith(
      2,
      expect.objectContaining({ cursor: "c1", filter: today }),
    );
    expect(shelfKey(today).slice(0, 3)).toEqual(assistant.threads().queryKey);
  });

  it("keeps each shelf apart from the others and from the whole list", async () => {
    listThreads.mockImplementation(({ filter }) =>
      Promise.resolve({
        items: [thread(filter === undefined ? "whole" : filter === today ? "today" : "older")],
        nextCursor: "",
      }),
    );

    await queryClient.fetchInfiniteQuery(threadListQuery());
    await queryClient.fetchInfiniteQuery(threadShelfQuery(today));
    await queryClient.fetchInfiniteQuery(threadShelfQuery(older));

    expect(threadsOf(queryClient.getQueryData(shelfKey(today))).map((item) => item.id)).toEqual([
      "today",
    ]);
    expect(threadsOf(queryClient.getQueryData(shelfKey(older))).map((item) => item.id)).toEqual([
      "older",
    ]);
    expect(cachedThreadLists(queryClient)).toHaveLength(3);
  });

  it("reads the counts for the edges given and keeps them under the list's key", async () => {
    countThreadShelves.mockResolvedValueOnce({ pinned: 2, bands: [1, 0, 3, 4, 5] });

    const counts = await queryClient.fetchQuery(threadShelfCountsQuery([4, 3, 2, 1]));

    expect(countThreadShelves).toHaveBeenCalledWith([4, 3, 2, 1], expect.anything());
    expect(counts).toEqual({ pinned: 2, bands: [1, 0, 3, 4, 5] });
    expect(assistant.threads()._ctx.shelfCounts([4, 3, 2, 1]).queryKey.slice(0, 3)).toEqual(
      assistant.threads().queryKey,
    );
  });

  it("rewrites and removes a conversation on every shelf that holds it, and not the counts", () => {
    const counts = { pinned: 0, bands: [1, 0, 0, 0, 1] };
    queryClient.setQueryData(shelfKey(today), pages([thread("a"), thread("b")]));
    queryClient.setQueryData(shelfKey(older), pages([thread("b")]));
    queryClient.setQueryData(assistant.threads()._ctx.shelfCounts([2, 1]).queryKey, counts);

    updateCachedThread(queryClient, "b", (item) => ({ ...item, title: "renamed" }));

    expect(threadsOf(queryClient.getQueryData(shelfKey(today)))[1].title).toBe("renamed");
    expect(threadsOf(queryClient.getQueryData(shelfKey(older)))[0].title).toBe("renamed");

    removeCachedThread(queryClient, "b");

    expect(threadsOf(queryClient.getQueryData(shelfKey(today))).map((item) => item.id)).toEqual([
      "a",
    ]);
    expect(threadsOf(queryClient.getQueryData(shelfKey(older)))).toEqual([]);
    expect(queryClient.getQueryData(assistant.threads()._ctx.shelfCounts([2, 1]).queryKey)).toBe(
      counts,
    );
  });

  it("knows every conversation any list holds, each once", () => {
    queryClient.setQueryData(assistant.threads().queryKey, pages([thread("a")]));
    queryClient.setQueryData(shelfKey(today), pages([thread("a"), thread("b")]));
    queryClient.setQueryData(shelfKey(older), pages([thread("c", { title: "far back" })]));

    expect(cachedThreads(queryClient).map((item) => item.id)).toEqual(["a", "b", "c"]);
    expect(findCachedThread(queryClient, "c")?.title).toBe("far back");
    expect(findCachedThread(queryClient, "missing")).toBeNull();
    expect(findCachedThread(queryClient, null)).toBeNull();
  });

  it("reads a shelf's page left behind once one of its conversations comes into view", async () => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false, staleTime: Number.POSITIVE_INFINITY } },
    });
    const server = Array.from({ length: 120 }, (_, index) => thread(`o${index}`));
    listThreads.mockImplementation(({ cursor = "", until, limit = THREAD_PAGE_SIZE }) => {
      const start = cursor === "" ? 0 : server.findIndex((item) => item.id === cursor) + 1;
      if (until) {
        const end = server.findIndex((item) => item.id === until);
        return Promise.resolve({ items: server.slice(start, end + 1), nextCursor: until });
      }
      const items = server.slice(start, start + limit);
      const more = start + limit < server.length;
      return Promise.resolve({ items, nextCursor: more ? (items.at(-1)?.id ?? "") : "" });
    });
    await queryClient.fetchInfiniteQuery({ ...threadShelfQuery(older), pages: 2 });
    const observer = new InfiniteQueryObserver(queryClient, threadShelfQuery(older));
    const stop = observer.subscribe(() => undefined);
    noteThreadsInView(queryClient, []);
    await queryClient.refetchQueries({ queryKey: shelfKey(older), exact: true });
    listThreads.mockClear();

    noteThreadsInView(queryClient, ["o60"]);

    await waitFor(() =>
      expect(listThreads).toHaveBeenCalledWith(
        expect.objectContaining({ cursor: "o49", until: "o99", filter: older }),
      ),
    );
    stop();
  });
});

/**
 * A rename, a pin or a delete shows in every list the moment it is asked for,
 * and a refused one is taken back out without reading the lists again.
 */
describe("optimistic changes", () => {
  const today: ThreadListFilter = { pinned: false, touchedFrom: 1_000 };
  const pinnedShelf: ThreadListFilter = { pinned: true };
  const shelfKey = (filter: ThreadListFilter) => assistant.threads()._ctx.shelf(filter).queryKey;
  const countsKey = (edges: number[]) => assistant.threads()._ctx.shelfCounts(edges).queryKey;
  const ids = (key: readonly unknown[]) =>
    threadsOf(queryClient.getQueryData<ThreadPages>(key)).map((item) => item.id);

  it("patches a conversation in every list and read on its own, and takes the patch back", () => {
    queryClient.setQueryData(assistant.threads().queryKey, pages([thread("a", { title: "old" })]));
    queryClient.setQueryData(shelfKey(today), pages([thread("a", { title: "old" })]));
    queryClient.setQueryData(assistant.thread("a").queryKey, thread("a", { title: "old" }));

    const undo = patchCachedThread(queryClient, "a", { title: "new" });

    expect(threadsOf(queryClient.getQueryData(assistant.threads().queryKey))[0].title).toBe("new");
    expect(threadsOf(queryClient.getQueryData(shelfKey(today)))[0].title).toBe("new");
    expect(queryClient.getQueryData<AssistantThread>(assistant.thread("a").queryKey)?.title).toBe(
      "new",
    );

    undo();

    expect(threadsOf(queryClient.getQueryData(assistant.threads().queryKey))[0].title).toBe("old");
    expect(threadsOf(queryClient.getQueryData(shelfKey(today)))[0].title).toBe("old");
    expect(queryClient.getQueryData<AssistantThread>(assistant.thread("a").queryKey)?.title).toBe(
      "old",
    );
  });

  it("leaves a later change in place when an earlier one is taken back", () => {
    queryClient.setQueryData(assistant.threads().queryKey, pages([thread("a", { pinned: false })]));

    const undoFirst = patchCachedThread(queryClient, "a", { pinned: true });
    patchCachedThread(queryClient, "a", { pinned: false });
    undoFirst();

    expect(cachedThread(queryClient, "a")?.pinned).toBe(false);

    const undoTitle = patchCachedThread(queryClient, "a", { title: "first" });
    patchCachedThread(queryClient, "a", { title: "second" });
    undoTitle();

    expect(cachedThread(queryClient, "a")?.title).toBe("second");
  });

  it("reads a conversation from a list, else on its own, else null", () => {
    queryClient.setQueryData(assistant.thread("solo").queryKey, thread("solo", { title: "alone" }));
    queryClient.setQueryData(
      assistant.threads().queryKey,
      pages([thread("a", { title: "listed" })]),
    );

    expect(cachedThread(queryClient, "a")?.title).toBe("listed");
    expect(cachedThread(queryClient, "solo")?.title).toBe("alone");
    expect(cachedThread(queryClient, "missing")).toBeNull();
  });

  it("takes a conversation out of every list and puts it back where each held it", () => {
    queryClient.setQueryData(
      assistant.threads().queryKey,
      pages([thread("a"), thread("b", { title: "held" }), thread("c")], [thread("d")]),
    );
    queryClient.setQueryData(shelfKey(today), pages([thread("x")], [thread("y"), thread("b")]));

    const taken = takeCachedThread(queryClient, "b");

    expect(taken.thread?.title).toBe("held");
    expect(ids(assistant.threads().queryKey)).toEqual(["a", "c", "d"]);
    expect(ids(shelfKey(today))).toEqual(["x", "y"]);

    taken.restore();

    const whole = queryClient.getQueryData<ThreadPages>(assistant.threads().queryKey);
    expect(whole?.pages.map((page) => page.items.map((t) => t.id))).toEqual([
      ["a", "b", "c"],
      ["d"],
    ]);
    const shelf = queryClient.getQueryData<ThreadPages>(shelfKey(today));
    expect(shelf?.pages.map((page) => page.items.map((t) => t.id))).toEqual([["x"], ["y", "b"]]);
  });

  it("does not put a conversation back into a list that holds it again", () => {
    queryClient.setQueryData(assistant.threads().queryKey, pages([thread("a"), thread("b")]));
    const taken = takeCachedThread(queryClient, "b");
    queryClient.setQueryData(assistant.threads().queryKey, pages([thread("b"), thread("a")]));

    taken.restore();

    expect(ids(assistant.threads().queryKey)).toEqual(["b", "a"]);
  });

  it("hands back nothing to restore when no list holds the conversation", () => {
    const data = pages([thread("a")]);
    queryClient.setQueryData(assistant.threads().queryKey, data);

    const taken = takeCachedThread(queryClient, "missing");
    taken.restore();

    expect(taken.thread).toBeNull();
    expect(queryClient.getQueryData(assistant.threads().queryKey)).toBe(data);
  });

  it("counts a conversation on and off the shelf its pin or last touch puts it on", () => {
    queryClient.setQueryData(countsKey([300, 200]), { pinned: 1, bands: [4, 5, 6] });
    queryClient.setQueryData(shelfKey(pinnedShelf), pages([thread("p", { pinned: true })]));

    countCachedThread(queryClient, thread("t", { lastMessageAt: 350 }), -1);
    countCachedThread(queryClient, thread("y", { lastMessageAt: 250 }), -1);
    countCachedThread(queryClient, thread("o", { lastMessageAt: 100, createdAt: 50 }), 1);
    countCachedThread(queryClient, thread("p", { pinned: true }), 1);

    expect(queryClient.getQueryData(countsKey([300, 200]))).toEqual({
      pinned: 2,
      bands: [3, 4, 7],
    });
    expect(ids(shelfKey(pinnedShelf))).toEqual(["p"]);
  });

  it("never counts a shelf below nothing", () => {
    queryClient.setQueryData(countsKey([300, 200]), { pinned: 0, bands: [0, 0, 0] });

    countCachedThread(queryClient, thread("p", { pinned: true }), -1);
    countCachedThread(queryClient, thread("t", { lastMessageAt: 350 }), -1);

    expect(queryClient.getQueryData(countsKey([300, 200]))).toEqual({
      pinned: 0,
      bands: [0, 0, 0],
    });
  });
});
