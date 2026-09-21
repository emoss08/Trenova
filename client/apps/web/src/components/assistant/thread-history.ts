import type { AssistantMessage, AssistantMessagePage } from "@/types/assistant";

/**
 * The shape the thread's history query keeps: page 0 is the newest window of
 * the thread and every later page is the one above it, fetched as the reader
 * scrolls up. Sequence numbers are the thread's own order, so a page cut on
 * one never repeats or skips a message however many turns land meanwhile.
 */
export type ThreadHistory = {
  pages: AssistantMessagePage[];
  pageParams: unknown[];
};

/**
 * Flattens the pages into reading order, oldest first.
 *
 * Pages arrive newest first and each page is already ascending, so the older
 * pages are walked from the last fetched to the first and the newest page is
 * appended last. Ids are deduplicated defensively: a turn appended to the
 * newest page and then refetched must never appear twice.
 */
export function flattenHistory(pages: readonly AssistantMessagePage[]): AssistantMessage[] {
  const seen = new Set<string>();
  const ordered: AssistantMessage[] = [];

  for (let index = pages.length - 1; index >= 0; index -= 1) {
    for (const message of pages[index].results) {
      if (seen.has(message.id)) {
        continue;
      }
      seen.add(message.id);
      ordered.push(message);
    }
  }

  return ordered;
}

/**
 * The sequence to page up from: the smallest one loaded so far. Undefined
 * when nothing has loaded, which asks for the newest page.
 */
export function oldestSequence(pages: readonly AssistantMessagePage[]): number | undefined {
  let oldest: number | undefined;
  for (const page of pages) {
    for (const message of page.results) {
      if (oldest === undefined || message.sequence < oldest) {
        oldest = message.sequence;
      }
    }
  }

  return oldest;
}

/**
 * Whether there is a page above the ones loaded. The last fetched page is the
 * highest one, so it is the one that knows.
 */
export function hasOlderPages(pages: readonly AssistantMessagePage[]): boolean {
  if (pages.length === 0) {
    return false;
  }

  return pages[pages.length - 1].hasMore;
}

/**
 * Appends a finished turn's saved messages to the newest page in place of
 * refetching. A thread with several pages loaded would otherwise refetch every
 * one of them after each reply; the turn result already carries the rows the
 * server wrote, ids and sequence numbers included.
 *
 * Returns the history unchanged when nothing is cached yet: the query will
 * fetch fresh on its own, and inventing a page here would race it.
 */
export function appendToHistory(
  history: ThreadHistory | undefined,
  messages: readonly AssistantMessage[],
): ThreadHistory | undefined {
  if (!history || history.pages.length === 0 || messages.length === 0) {
    return history;
  }

  const [newest, ...older] = history.pages;
  const known = new Set(newest.results.map((message) => message.id));
  const fresh = messages.filter((message) => !known.has(message.id));
  if (fresh.length === 0) {
    return history;
  }

  // The rows are placed by sequence rather than trusted as they came: the
  // done payload is assembled in the runtime, and the page has to read in
  // the order the server numbered it.
  const results = [...newest.results, ...fresh].sort((a, b) => a.sequence - b.sequence);

  return {
    ...history,
    pages: [
      {
        ...newest,
        results,
        total: newest.total + fresh.length,
      },
      ...older,
    ],
  };
}

/**
 * Whether a finished turn picks up exactly where the newest page ends.
 *
 * A turn sent while the previous one was still streaming aborts that stream.
 * The server keeps what had run, the client never fetched it, and appending
 * the next turn on top would hide the gap for good: the question that was
 * answered, and the answer, gone from the thread until a full reload. The
 * sequence numbers say whether the page is still continuous; when they do
 * not, the caller fetches instead of appending.
 */
export function continuesHistory(
  history: ThreadHistory | undefined,
  messages: readonly AssistantMessage[],
): boolean {
  if (!history || history.pages.length === 0) {
    return false;
  }
  const newest = history.pages[0];
  const known = new Set(newest.results.map((message) => message.id));
  const fresh = messages.filter((message) => !known.has(message.id));
  if (fresh.length === 0) {
    return true;
  }

  let highest = -1;
  for (const message of newest.results) {
    if (message.sequence > highest) {
      highest = message.sequence;
    }
  }
  let first = Number.POSITIVE_INFINITY;
  for (const message of fresh) {
    if (message.sequence < first) {
      first = message.sequence;
    }
  }

  return first === highest + 1;
}

export type ThreadLengthState = "open" | "long" | "full";

/** The fraction of the limit at which a conversation starts to be called long. */
const LONG_THRESHOLD = 0.8;

/**
 * How close a conversation is to the point where it must be continued in a
 * new one. `full` is where the server refuses the next turn; `long` is the
 * warning before it, so nobody meets the wall mid-thought.
 */
export function threadLength(
  total: number,
  limit: number,
): { state: ThreadLengthState; remaining: number } {
  if (limit <= 0) {
    return { state: "open", remaining: Number.POSITIVE_INFINITY };
  }
  const remaining = Math.max(0, limit - total);
  if (total >= limit) {
    return { state: "full", remaining: 0 };
  }
  if (total >= Math.ceil(limit * LONG_THRESHOLD)) {
    return { state: "long", remaining };
  }

  return { state: "open", remaining };
}
