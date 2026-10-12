import type { AssistantMessage, AssistantMessagePage, MemoryNote } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  appendToHistory,
  continuesHistory,
  flattenHistory,
  hasOlderPages,
  oldestSequence,
  replaceMemoryNote,
  threadLength,
} from "../thread-history";

function message(sequence: number, id = `amsg_${sequence}`): AssistantMessage {
  return {
    id,
    threadId: "thr_1",
    kind: "Message",
    sequence,
    role: sequence % 2 === 0 ? "User" : "Assistant",
    content: `m${sequence}`,
    toolCalls: null,
    toolCallId: "",
    toolName: "",
    toolFailed: false,
    scopeStage: "",
    scopeCategory: "",
    scopeReason: "",
    refused: false,
    model: "",
    inputTokens: 0,
    outputTokens: 0,
    createdAt: 0,
  };
}

function page(sequences: number[], hasMore = false, total = 0): AssistantMessagePage {
  return { results: sequences.map((sequence) => message(sequence)), hasMore, total, limit: 400 };
}

describe("flattenHistory", () => {
  // Pages arrive newest first as the reader scrolls up; the thread reads
  // oldest first. Sequence order across pages is what the keyset guarantees.
  it("reads pages fetched newest-first in thread order", () => {
    const pages = [page([6, 7, 8]), page([3, 4, 5]), page([0, 1, 2])];

    expect(flattenHistory(pages).map((m) => m.sequence)).toEqual([0, 1, 2, 3, 4, 5, 6, 7, 8]);
  });

  it("drops a message that appears in two pages", () => {
    const pages = [page([4, 5]), page([3, 4])];

    expect(flattenHistory(pages).map((m) => m.sequence)).toEqual([3, 4, 5]);
  });

  it("is empty for no pages", () => {
    expect(flattenHistory([])).toEqual([]);
  });
});

describe("oldestSequence / hasOlderPages", () => {
  it("pages up from the smallest loaded sequence", () => {
    expect(oldestSequence([page([6, 7]), page([3, 4, 5])])).toBe(3);
    expect(oldestSequence([])).toBeUndefined();
  });

  it("asks the highest page whether more exists", () => {
    expect(hasOlderPages([page([6, 7], false), page([3, 4], true)])).toBe(true);
    expect(hasOlderPages([page([6, 7], true), page([3, 4], false)])).toBe(false);
    expect(hasOlderPages([])).toBe(false);
  });
});

describe("appendToHistory", () => {
  // A thread with three pages loaded used to refetch all three after every
  // reply. The turn result carries the saved rows, so they are appended.
  it("appends a finished turn to the newest page and counts it", () => {
    const history = {
      pages: [page([4, 5], false, 6), page([0, 1, 2, 3], false, 6)],
      pageParams: [],
    };

    const next = appendToHistory(history, [message(6), message(7)]);

    expect(next?.pages[0].results.map((m) => m.sequence)).toEqual([4, 5, 6, 7]);
    expect(next?.pages[0].total).toBe(8);
    expect(next?.pages[1]).toBe(history.pages[1]);
  });

  it("does not append a message the page already holds", () => {
    const history = { pages: [page([4, 5], false, 6)], pageParams: [] };

    const next = appendToHistory(history, [message(5)]);

    expect(next).toBe(history);
  });

  it("leaves an empty cache alone so the query fetches fresh", () => {
    expect(appendToHistory(undefined, [message(1)])).toBeUndefined();
    const empty = { pages: [], pageParams: [] };
    expect(appendToHistory(empty, [message(1)])).toBe(empty);
  });
});

describe("continuesHistory", () => {
  // A turn answered while the previous one was still streaming aborts that
  // stream; the server keeps what ran, the client never fetched it, and
  // appending the next turn on top would hide the gap for good. The
  // sequence numbers say whether the newest page is still continuous.
  it("accepts a turn that picks up where the page left off", () => {
    const history = { pages: [page([4, 5], false, 6)], pageParams: [] };

    expect(continuesHistory(history, [message(6), message(7)])).toBe(true);
  });

  it("refuses a turn that skips messages the page never saw", () => {
    const history = { pages: [page([4, 5], false, 6)], pageParams: [] };

    expect(continuesHistory(history, [message(9)])).toBe(false);
  });

  it("ignores messages the page already holds when judging continuity", () => {
    const history = { pages: [page([4, 5], false, 6)], pageParams: [] };

    expect(continuesHistory(history, [message(5), message(6)])).toBe(true);
    expect(continuesHistory(history, [message(5)])).toBe(true);
  });

  it("starts an empty thread at its first message and nowhere else", () => {
    const empty = { pages: [page([], false, 0)], pageParams: [] };

    expect(continuesHistory(empty, [message(0), message(1)])).toBe(true);
    expect(continuesHistory(empty, [message(3)])).toBe(false);
  });

  it("cannot vouch for a history that was never fetched", () => {
    expect(continuesHistory(undefined, [message(0)])).toBe(false);
    expect(continuesHistory({ pages: [], pageParams: [] }, [message(0)])).toBe(false);
  });
});

describe("threadLength", () => {
  it("warns before the wall and closes at it", () => {
    expect(threadLength(10, 400)).toEqual({ state: "open", remaining: 390 });
    expect(threadLength(320, 400)).toEqual({ state: "long", remaining: 80 });
    expect(threadLength(400, 400)).toEqual({ state: "full", remaining: 0 });
    expect(threadLength(450, 400)).toEqual({ state: "full", remaining: 0 });
  });

  it("never closes a thread the server has not bounded", () => {
    expect(threadLength(9999, 0).state).toBe("open");
  });
});

// The done payload is assembled in the runtime, not read back in order, so
// the appended rows are placed by sequence rather than trusted as they came.
describe("appendToHistory ordering", () => {
  it("places appended rows by sequence whatever order they arrived in", () => {
    const history = {
      pages: [{ results: [message(0), message(1)], total: 2, limit: 50, hasMore: false }],
      pageParams: [undefined],
    };

    const appended = appendToHistory(history, [message(4), message(2), message(3)]);

    expect(appended?.pages[0].results.map((row) => row.sequence)).toEqual([0, 1, 2, 3, 4]);
  });
});

describe("replaceMemoryNote", () => {
  const note = (id: string, status = "Suggested"): MemoryNote => ({
    id,
    content: "Read the move first.",
    kind: "Procedure",
    scope: "Agent",
    roleId: null,
    roleName: null,
    status,
    source: "Reflection",
    sourceTitle: null,
    createdAt: 1,
    version: 1,
    editable: true,
    reason: null,
    replaces: null,
    replacedBy: null,
  });
  const withNotes = (sequence: number, notes: MemoryNote[]): AssistantMessage => ({
    ...message(sequence),
    memories: notes,
  });
  const history = (...pages: AssistantMessage[][]) => ({
    pages: pages.map((results) => ({ ...page([]), results })),
    pageParams: pages.map(() => undefined),
  });

  it("updates every message that carries the memory, across pages", () => {
    const cached = history(
      [withNotes(3, [note("amem_1")]), message(4)],
      [withNotes(1, [note("amem_1"), note("amem_2")])],
    );

    const next = replaceMemoryNote(cached, "amem_1", { status: "Active", version: 2 });

    expect(next?.pages[0].results[0].memories?.[0]).toMatchObject({ status: "Active", version: 2 });
    expect(next?.pages[1].results[0].memories?.[0]).toMatchObject({ status: "Active" });
    expect(next?.pages[1].results[0].memories?.[1].status).toBe("Suggested");
    expect(next?.pages[0].results[1]).toBe(cached.pages[0].results[1]);
  });

  it("returns the same history when no message carries the memory", () => {
    const cached = history([withNotes(1, [note("amem_2")])]);

    expect(replaceMemoryNote(cached, "amem_1", { status: "Active" })).toBe(cached);
  });

  it("leaves a thread that is not cached alone", () => {
    expect(replaceMemoryNote(undefined, "amem_1", { status: "Active" })).toBeUndefined();
  });
});
