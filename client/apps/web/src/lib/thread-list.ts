import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { AssistantThread, AssistantThreadPage } from "@/types/assistant";
import { infiniteQueryOptions, type InfiniteData, type QueryClient } from "@tanstack/react-query";

/** Conversations a page of the list holds: a rail and a half. */
export const THREAD_PAGE_SIZE = 50;

/** The person's conversations as the cache holds them: every page read so far. */
export type ThreadPages = InfiniteData<AssistantThreadPage, string>;

const NO_THREADS: AssistantThread[] = [];

/** The most a page read on its own asks for; the server's own cap. */
const MAX_PAGE_SIZE = 100;

/**
 * What one tab knows about its list beyond the pages themselves: which pages
 * were left as they were when the list was last refreshed, which
 * conversations the rail is drawing, and whether the next refresh is only to
 * catch up on the stale pages that came into view.
 */
type ListState = {
  stale: Set<string>;
  inView: ReadonlySet<string>;
  catchUpNext: boolean;
};

const states = new WeakMap<QueryClient, ListState>();

function listState(client: QueryClient): ListState {
  let state = states.get(client);
  if (!state) {
    state = { stale: new Set(), inView: new Set(), catchUpNext: false };
    states.set(client, state);
  }
  return state;
}

function shows(page: AssistantThreadPage, inView: ReadonlySet<string>): boolean {
  return inView.size > 0 && page.items.some((thread) => inView.has(thread.id));
}

/**
 * Reads a page's range again: from the cursor before it down to the place
 * its last conversation held, so it keeps its boundary and the pages after it
 * still continue from where they did. The last page of a list read to its end
 * has no boundary below it, and is read as a page.
 */
function readAgain(page: AssistantThreadPage, cursor: string, signal: AbortSignal) {
  if (page.nextCursor === "") {
    return apiService.assistantService.listThreads({
      cursor,
      limit: Math.min(MAX_PAGE_SIZE, Math.max(THREAD_PAGE_SIZE, page.items.length)),
      signal,
    });
  }

  return apiService.assistantService.listThreads({ cursor, until: page.nextCursor, signal });
}

/**
 * Reads one page of the list: a new one as the rail scrolls to the end, or
 * an existing one when the list is refreshed.
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
  pageParam,
  signal,
}: {
  client: QueryClient;
  pageParam: string;
  signal: AbortSignal;
}): Promise<AssistantThreadPage> {
  const state = listState(client);
  const cached = client.getQueryData<ThreadPages>(queries.assistant.threads().queryKey);
  const index = cached ? cached.pageParams.indexOf(pageParam) : -1;
  if (!cached || index < 0) {
    if (pageParam === "") {
      state.stale.clear();
    }
    return apiService.assistantService.listThreads({
      limit: THREAD_PAGE_SIZE,
      cursor: pageParam,
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
  const wanted = state.stale.has(pageParam) && (index === 0 || shows(page, state.inView));
  if (!wanted) {
    return page;
  }
  const fresh = await readAgain(page, pageParam, signal);
  state.stale.delete(pageParam);

  return fresh;
}

/**
 * The person's conversations, a page at a time, pinned first and then newest
 * first. Each page continues below the last row of the one before it, so a
 * conversation touched while the reader scrolls is neither repeated nor
 * skipped. A refresh reads only the first page and what the rail is showing;
 * see readPage.
 */
export function threadListQuery() {
  return infiniteQueryOptions({
    queryKey: queries.assistant.threads().queryKey,
    queryFn: ({ client, pageParam, signal }) => readPage({ client, pageParam, signal }),
    initialPageParam: "",
    getNextPageParam: (last: AssistantThreadPage) => last.nextCursor || undefined,
  });
}

/**
 * Tells the list which conversations the rail is drawing. A page left stale
 * by an earlier refresh is read again as soon as one of its conversations
 * comes into view; pass an empty list when the rail goes away.
 */
export function noteThreadsInView(client: QueryClient, threadIds: readonly string[]): void {
  const state = listState(client);
  state.inView = new Set(threadIds);
  if (state.stale.size === 0 || state.inView.size === 0) {
    return;
  }
  const key = queries.assistant.threads().queryKey;
  const cached = client.getQueryData<ThreadPages>(key);
  if (!cached || client.isFetching({ queryKey: key, exact: true }) > 0) {
    return;
  }
  const behind = cached.pages.some(
    (page, index) => state.stale.has(cached.pageParams[index]) && shows(page, state.inView),
  );
  if (!behind) {
    return;
  }
  state.catchUpNext = true;
  void client.refetchQueries({ queryKey: key, exact: true, type: "active" });
}

/** Takes a conversation out of every cached page, as soon as it is gone. */
export function removeCachedThread(queryClient: QueryClient, threadId: string): void {
  queryClient.setQueryData<ThreadPages>(queries.assistant.threads().queryKey, (cached) => {
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

/** The conversations this tab has read, without asking for any. */
export function cachedThreads(queryClient: QueryClient): AssistantThread[] {
  return threadsOf(queryClient.getQueryData<ThreadPages>(queries.assistant.threads().queryKey));
}

/** Rewrites one conversation wherever the cached pages hold it. */
export function updateCachedThread(
  queryClient: QueryClient,
  threadId: string,
  update: (thread: AssistantThread) => AssistantThread,
): void {
  queryClient.setQueryData<ThreadPages>(queries.assistant.threads().queryKey, (cached) => {
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
