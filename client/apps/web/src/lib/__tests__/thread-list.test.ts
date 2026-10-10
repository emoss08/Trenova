import { queries } from "@/lib/queries";
import type { AssistantThread, AssistantThreadPage } from "@/types/assistant";
import { InfiniteQueryObserver, QueryClient } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { waitFor } from "@testing-library/react";
import {
  cachedThreads,
  noteThreadsInView,
  removeCachedThread,
  THREAD_PAGE_SIZE,
  threadListQuery,
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
      signal?: AbortSignal;
    }) => Promise<unknown>
  >(),
);

vi.mock("@/services/api", () => ({
  apiService: { assistantService: { listThreads } },
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
    queryClient.setQueryData(
      queries.assistant.threads().queryKey,
      pages([thread("a")], [thread("b")]),
    );
    const before = queryClient.getQueryData<ThreadPages>(queries.assistant.threads().queryKey);

    updateCachedThread(queryClient, "b", (item) => ({ ...item, title: "renamed" }));

    const after = queryClient.getQueryData<ThreadPages>(queries.assistant.threads().queryKey);
    expect(after?.pages[1].items[0].title).toBe("renamed");
    expect(after?.pages[0]).toBe(before?.pages[0]);
  });

  it("leaves the cache as it was when no page holds the conversation", () => {
    const data = pages([thread("a")]);
    queryClient.setQueryData(queries.assistant.threads().queryKey, data);

    updateCachedThread(queryClient, "missing", (item) => ({ ...item, title: "x" }));

    expect(queryClient.getQueryData(queries.assistant.threads().queryKey)).toBe(data);
  });

  it("does nothing before the list has been read", () => {
    updateCachedThread(queryClient, "a", (item) => ({ ...item, title: "x" }));

    expect(queryClient.getQueryData(queries.assistant.threads().queryKey)).toBeUndefined();
  });
});

/**
 * A person far down the rail has many pages read. A refresh reads the first
 * page and the pages the rail is drawing, and leaves the rest until they come
 * into view, so a reply landing does not download the whole list again.
 */
describe("refreshing a long list", () => {
  let server: AssistantThread[] = [];
  const key = queries.assistant.threads().queryKey;
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
