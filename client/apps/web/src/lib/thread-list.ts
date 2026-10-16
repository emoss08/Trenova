import { threadTouchedAt } from "@/components/desk-chat/rail/desk-threads";
import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import type { ThreadListFilter } from "@/services/assistant";
import type {
  AssistantThread,
  AssistantThreadPage,
  AssistantThreadShelfCounts,
} from "@/types/assistant";
import {
  hashKey,
  infiniteQueryOptions,
  queryOptions,
  type DefaultError,
  type InfiniteData,
  type Query,
  type QueryClient,
  type QueryKey,
} from "@tanstack/react-query";

/** Conversations a page of the list holds: a rail and a half. */
export const THREAD_PAGE_SIZE = 50;

/** The person's conversations as the cache holds them: every page read so far. */
export type ThreadPages = InfiniteData<AssistantThreadPage, string>;

const NO_THREADS: AssistantThread[] = [];

/** The most a page read on its own asks for; the server's own cap. */
const MAX_PAGE_SIZE = 100;

/**
 * What one list in one tab knows beyond its pages: which pages were left as
 * they were when it was last refreshed, and whether its next refresh is only
 * to catch up on the stale pages that came into view.
 */
type ListState = {
  stale: Set<string>;
  catchUpNext: boolean;
};

/**
 * What one tab knows about the person's lists of conversations: the whole
 * list and each shelf of the rail are lists of their own, and the rail says
 * once which conversations it is drawing, whichever list each came from.
 */
type ClientState = {
  inView: ReadonlySet<string>;
  lists: Map<string, ListState>;
};

const states = new WeakMap<QueryClient, ClientState>();

function clientState(client: QueryClient): ClientState {
  let state = states.get(client);
  if (!state) {
    state = { inView: new Set(), lists: new Map() };
    states.set(client, state);
  }
  return state;
}

function listState(client: QueryClient, queryKey: QueryKey): ListState {
  const lists = clientState(client).lists;
  const hash = hashKey(queryKey);
  let state = lists.get(hash);
  if (!state) {
    state = { stale: new Set(), catchUpNext: false };
    lists.set(hash, state);
  }
  return state;
}

/** The key every list of conversations, and the shelf counts, are cached under. */
function threadListRoot(): QueryKey {
  return assistant.threads().queryKey;
}

/** Whether a cache entry under the list's key holds pages of conversations rather than counts. */
function isThreadPages(data: unknown): data is ThreadPages {
  return (
    typeof data === "object" &&
    data !== null &&
    Array.isArray((data as Partial<ThreadPages>).pages) &&
    Array.isArray((data as Partial<ThreadPages>).pageParams)
  );
}

const THREAD_LISTS = {
  queryKey: threadListRoot(),
  predicate: (query: Query) => isThreadPages(query.state.data),
};

function shows(page: AssistantThreadPage, inView: ReadonlySet<string>): boolean {
  return inView.size > 0 && page.items.some((thread) => inView.has(thread.id));
}

type ReadOptions = {
  filter: ThreadListFilter | undefined;
  signal: AbortSignal;
};

/**
 * Reads a page's range again: from the cursor before it down to the place
 * its last conversation held, so it keeps its boundary and the pages after it
 * still continue from where they did. The last page of a list read to its end
 * has no boundary below it, and is read as a page.
 */
function readAgain(page: AssistantThreadPage, cursor: string, { filter, signal }: ReadOptions) {
  if (page.nextCursor === "") {
    return apiService.assistantService.listThreads({
      cursor,
      limit: Math.min(MAX_PAGE_SIZE, Math.max(THREAD_PAGE_SIZE, page.items.length)),
      filter,
      signal,
    });
  }

  return apiService.assistantService.listThreads({
    cursor,
    until: page.nextCursor,
    filter,
    signal,
  });
}

/**
 * Reads one page of a list: a new one as the rail scrolls to the end, or an
 * existing one when the list is refreshed.
 *
 * A refresh reads the first page and the pages whose conversations the rail
 * is drawing, and leaves the rest as they are, marked stale; each is read
 * again when it comes into view (noteThreadsInView). Every page owns the
 * range of the list between its boundary and the one before, so reading some
 * pages and not others never repeats or loses a conversation: one that moved
 * shows once, from the page read most recently above it.
 */
async function readPage({
  client,
  queryKey,
  pageParam,
  filter,
  signal,
}: ReadOptions & {
  client: QueryClient;
  queryKey: QueryKey;
  pageParam: string;
}): Promise<AssistantThreadPage> {
  const state = listState(client, queryKey);
  const cached = client.getQueryData<ThreadPages>(queryKey);
  const index = cached ? cached.pageParams.indexOf(pageParam) : -1;
  if (!cached || index < 0) {
    if (pageParam === "") {
      state.stale.clear();
    }
    return apiService.assistantService.listThreads({
      limit: THREAD_PAGE_SIZE,
      cursor: pageParam,
      filter,
      signal,
    });
  }

  // A refresh always begins with the first page. One that is not catching up
  // leaves every page it does not read stale.
  if (index === 0) {
    const catchingUp = state.catchUpNext;
    state.catchUpNext = false;
    if (!catchingUp) {
      for (const param of cached.pageParams) {
        state.stale.add(param);
      }
    }
  }

  const page = cached.pages[index];
  const wanted =
    state.stale.has(pageParam) && (index === 0 || shows(page, clientState(client).inView));
  if (!wanted) {
    return page;
  }
  const fresh = await readAgain(page, pageParam, { filter, signal });
  state.stale.delete(pageParam);

  return fresh;
}

/** Where each page of a list continues: below the last row of the one before. */
const PAGED_BY_CURSOR = {
  initialPageParam: "",
  getNextPageParam: (last: AssistantThreadPage) => last.nextCursor || undefined,
};

/**
 * The person's conversations, a page at a time, pinned first and then most
 * recently touched first. Each page continues below the last row of the one
 * before it, so a conversation touched while the reader scrolls is neither
 * repeated nor skipped. A refresh reads only the first page and what the rail
 * is showing; see readPage.
 */
export function threadListQuery() {
  return infiniteQueryOptions<
    AssistantThreadPage,
    DefaultError,
    ThreadPages,
    ReturnType<typeof assistant.threads>["queryKey"],
    string
  >({
    queryKey: assistant.threads().queryKey,
    queryFn: ({ client, queryKey, pageParam, signal }) =>
      readPage({ client, queryKey, pageParam, filter: undefined, signal }),
    ...PAGED_BY_CURSOR,
  });
}

/**
 * One shelf of the rail, read on its own a page at a time: the server keeps
 * to the shelf's range, so a shelf the reader has folded away costs nothing
 * until it is opened. Refreshed as the whole list is; see readPage.
 */
export function threadShelfQuery(filter: ThreadListFilter) {
  return infiniteQueryOptions<
    AssistantThreadPage,
    DefaultError,
    ThreadPages,
    ReturnType<ReturnType<typeof assistant.threads>["_ctx"]["shelf"]>["queryKey"],
    string
  >({
    queryKey: assistant.threads()._ctx.shelf(filter).queryKey,
    queryFn: ({ client, queryKey, pageParam, signal }) =>
      readPage({ client, queryKey, pageParam, filter, signal }),
    ...PAGED_BY_CURSOR,
  });
}

/**
 * How many conversations each shelf holds, banded by the instants each shelf
 * begins at, newest first. Cached under the list's key, so whatever refreshes
 * the list refreshes the counts.
 */
export function threadShelfCountsQuery(edges: readonly number[]) {
  return queryOptions({
    ...assistant.threads()._ctx.shelfCounts(edges),
    queryFn: ({ signal }) => apiService.assistantService.countThreadShelves(edges, { signal }),
  });
}

/**
 * Tells the lists which conversations the rail is drawing. A page left stale
 * by an earlier refresh is read again as soon as one of its conversations
 * comes into view; pass an empty list when the rail goes away.
 */
export function noteThreadsInView(client: QueryClient, threadIds: readonly string[]): void {
  const state = clientState(client);
  state.inView = new Set(threadIds);
  if (state.inView.size === 0) {
    return;
  }

  for (const query of client.getQueryCache().findAll(THREAD_LISTS)) {
    const data = query.state.data;
    if (!isThreadPages(data) || query.state.fetchStatus !== "idle") {
      continue;
    }
    const list = listState(client, query.queryKey);
    if (list.stale.size === 0) {
      continue;
    }
    const behind = data.pages.some(
      (page, index) => list.stale.has(data.pageParams[index]) && shows(page, state.inView),
    );
    if (!behind) {
      continue;
    }
    list.catchUpNext = true;
    void client.refetchQueries({ queryKey: query.queryKey, exact: true, type: "active" });
  }
}

/** Takes a conversation out of every cached page of every list, as soon as it is gone. */
export function removeCachedThread(queryClient: QueryClient, threadId: string): void {
  queryClient.setQueriesData<ThreadPages>(THREAD_LISTS, (cached) => {
    if (!cached) {
      return cached;
    }
    let changed = false;
    const pages = cached.pages.map((page) => {
      if (!page.items.some((thread) => thread.id === threadId)) {
        return page;
      }
      changed = true;
      return { ...page, items: page.items.filter((thread) => thread.id !== threadId) };
    });

    return changed ? { ...cached, pages } : cached;
  });
}

const flattened = new WeakMap<ThreadPages, AssistantThread[]>();

/**
 * Every conversation the pages hold, in list order, each once. Read once per
 * cache entry, so every reader of the same data shares one array.
 */
export function threadsOf(data: ThreadPages | undefined): AssistantThread[] {
  if (!data) {
    return NO_THREADS;
  }
  const known = flattened.get(data);
  if (known) {
    return known;
  }

  const seen = new Set<string>();
  const threads: AssistantThread[] = [];
  for (const page of data.pages) {
    for (const thread of page.items) {
      if (!seen.has(thread.id)) {
        seen.add(thread.id);
        threads.push(thread);
      }
    }
  }
  flattened.set(data, threads);

  return threads;
}

/** One conversation from the pages, as threadsOf would give it: the first page holding it wins. */
export function findThread(data: ThreadPages | undefined, threadId: string | null) {
  if (!data || threadId === null) {
    return null;
  }
  for (const page of data.pages) {
    const found = page.items.find((thread) => thread.id === threadId);
    if (found) {
      return found;
    }
  }
  return null;
}

/**
 * The person's most recent conversations: the list's first page, which holds
 * what they pinned and what they touched last. What recency is read from,
 * such as which agents to offer first, needs nothing older.
 */
export function recentThreadsOf(data: ThreadPages | undefined): AssistantThread[] {
  return data?.pages[0]?.items ?? NO_THREADS;
}

/** Every list of conversations this tab holds: the whole list and each shelf read so far. */
export function cachedThreadLists(queryClient: QueryClient): ThreadPages[] {
  const lists: ThreadPages[] = [];
  for (const [, data] of queryClient.getQueriesData(THREAD_LISTS)) {
    if (isThreadPages(data)) {
      lists.push(data);
    }
  }

  return lists;
}

/** The conversations this tab has read, in any list, each once, without asking for any. */
export function cachedThreads(queryClient: QueryClient): AssistantThread[] {
  const seen = new Set<string>();
  const threads: AssistantThread[] = [];
  for (const list of cachedThreadLists(queryClient)) {
    for (const thread of threadsOf(list)) {
      if (!seen.has(thread.id)) {
        seen.add(thread.id);
        threads.push(thread);
      }
    }
  }

  return threads;
}

/** One conversation as any list this tab holds has it, without asking for it. */
export function findCachedThread(
  queryClient: QueryClient,
  threadId: string | null,
): AssistantThread | null {
  if (threadId === null) {
    return null;
  }
  for (const list of cachedThreadLists(queryClient)) {
    const found = findThread(list, threadId);
    if (found) {
      return found;
    }
  }

  return null;
}

/** Rewrites one conversation wherever the cached pages of any list hold it. */
export function updateCachedThread(
  queryClient: QueryClient,
  threadId: string,
  update: (thread: AssistantThread) => AssistantThread,
): void {
  queryClient.setQueriesData<ThreadPages>(THREAD_LISTS, (cached) => {
    if (!cached) {
      return cached;
    }
    let changed = false;
    const pages = cached.pages.map((page) => {
      if (!page.items.some((thread) => thread.id === threadId)) {
        return page;
      }
      changed = true;
      return {
        ...page,
        items: page.items.map((thread) => (thread.id === threadId ? update(thread) : thread)),
      };
    });

    return changed ? { ...cached, pages } : cached;
  });
}

/** One conversation as this tab holds it, in any list or read on its own, without asking for it. */
export function cachedThread(queryClient: QueryClient, threadId: string): AssistantThread | null {
  return (
    findCachedThread(queryClient, threadId) ??
    queryClient.getQueryData<AssistantThread>(assistant.thread(threadId).queryKey) ??
    null
  );
}

/**
 * Writes changes into one conversation wherever this tab holds it, in every
 * list and read on its own, as soon as they are asked for, and hands back
 * what takes them out again if the server refuses them. Taking them out
 * leaves alone a field a later change has rewritten since.
 */
export function patchCachedThread(
  queryClient: QueryClient,
  threadId: string,
  changes: Partial<AssistantThread>,
): () => void {
  const before = cachedThread(queryClient, threadId);
  writeThread(queryClient, threadId, (thread) => ({ ...thread, ...changes }));
  if (!before) {
    return () => undefined;
  }

  const keys = Object.keys(changes) as (keyof AssistantThread)[];
  const undo = Object.fromEntries(
    keys.map((key) => [key, before[key]]),
  ) as Partial<AssistantThread>;

  return () =>
    writeThread(queryClient, threadId, (thread) =>
      keys.every((key) => Object.is(thread[key], changes[key])) ? { ...thread, ...undo } : thread,
    );
}

function writeThread(
  queryClient: QueryClient,
  threadId: string,
  update: (thread: AssistantThread) => AssistantThread,
): void {
  updateCachedThread(queryClient, threadId, update);
  queryClient.setQueryData<AssistantThread>(assistant.thread(threadId).queryKey, (cached) =>
    cached ? update(cached) : cached,
  );
}

/** Where a conversation sat in one cached list, so it can be put back there. */
type CachedPlace = {
  queryKey: QueryKey;
  pageParam: string;
  index: number;
};

/** A conversation taken out of every cached list, and what puts it back. */
export type TakenThread = {
  /** The conversation as the lists held it; null when none did. */
  thread: AssistantThread | null;
  /** Puts it back where each list held it, unless that list holds it again already. */
  restore: () => void;
};

/**
 * Takes a conversation out of every cached page of every list the moment it
 * is deleted, remembering where each held it so a refused delete puts it back
 * in the same place rather than waiting for the list to be read again.
 */
export function takeCachedThread(queryClient: QueryClient, threadId: string): TakenThread {
  let taken: AssistantThread | null = null;
  const places: CachedPlace[] = [];
  for (const query of queryClient.getQueryCache().findAll(THREAD_LISTS)) {
    const data = query.state.data;
    if (!isThreadPages(data)) {
      continue;
    }
    for (const [pageIndex, page] of data.pages.entries()) {
      const index = page.items.findIndex((thread) => thread.id === threadId);
      if (index < 0) {
        continue;
      }
      taken ??= page.items[index];
      places.push({ queryKey: query.queryKey, pageParam: data.pageParams[pageIndex], index });
    }
  }
  removeCachedThread(queryClient, threadId);

  const thread = taken;
  const restore = () => {
    if (thread === null) {
      return;
    }
    for (const place of places) {
      queryClient.setQueryData<ThreadPages>(place.queryKey, (cached) =>
        cached ? putBack(cached, thread, place) : cached,
      );
    }
  };

  return { thread, restore };
}

function putBack(cached: ThreadPages, thread: AssistantThread, place: CachedPlace): ThreadPages {
  const pageIndex = cached.pageParams.indexOf(place.pageParam);
  if (pageIndex < 0 || threadsOf(cached).some((item) => item.id === thread.id)) {
    return cached;
  }
  const page = cached.pages[pageIndex];
  const items = [...page.items];
  items.splice(Math.min(place.index, items.length), 0, thread);
  const pages = [...cached.pages];
  pages[pageIndex] = { ...page, items };

  return { ...cached, pages };
}

function isShelfCounts(data: unknown): data is AssistantThreadShelfCounts {
  return (
    typeof data === "object" &&
    data !== null &&
    typeof (data as Partial<AssistantThreadShelfCounts>).pinned === "number" &&
    Array.isArray((data as Partial<AssistantThreadShelfCounts>).bands)
  );
}

/** The edges a cache entry of shelf counts was read for: the last part of its key. */
function shelfCountEdges(queryKey: QueryKey): readonly number[] | null {
  const edges = queryKey.at(-1);
  return Array.isArray(edges) && edges.every((edge) => typeof edge === "number") ? edges : null;
}

/**
 * Counts a conversation onto (1) or off (-1) the shelf it sits on, in every
 * set of shelf counts this tab holds, so the rail's numbers follow a pin or a
 * delete before the counts are read again: the pinned shelf when it is
 * pinned, else the band its last touch falls in.
 */
export function countCachedThread(
  queryClient: QueryClient,
  thread: AssistantThread,
  delta: 1 | -1,
): void {
  for (const query of queryClient.getQueryCache().findAll({ queryKey: threadListRoot() })) {
    const edges = shelfCountEdges(query.queryKey);
    if (edges === null || !isShelfCounts(query.state.data)) {
      continue;
    }
    queryClient.setQueryData<AssistantThreadShelfCounts>(query.queryKey, (counts) => {
      if (!counts) {
        return counts;
      }
      if (thread.pinned) {
        return { ...counts, pinned: Math.max(0, counts.pinned + delta) };
      }
      const touched = threadTouchedAt(thread);
      const band = edges.findIndex((edge) => touched >= edge);
      const at = band < 0 ? edges.length : band;
      if (at >= counts.bands.length) {
        return counts;
      }
      const bands = [...counts.bands];
      bands[at] = Math.max(0, bands[at] + delta);

      return { ...counts, bands };
    });
  }
}
