import type { AssistantThread } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { assistantShelves } from "../thread-grouping";

const DAY = 24 * 60 * 60;
// A Wednesday at 15:00 UTC, so "today" and "yesterday" are unambiguous.
const NOW = Date.UTC(2026, 8, 16, 15, 0, 0) / 1000;

function thread(
  id: string,
  lastMessageAt: number,
  overrides: Partial<AssistantThread> = {},
): AssistantThread {
  return {
    id,
    businessUnitId: "bu",
    organizationId: "org",
    userId: "u",
    agentDefinitionId: "agdef_1",
    preferredProviderId: "",
    origin: "Desk",
    pinned: false,
    subjectType: "",
    subjectId: "",
    canContinue: true,
    title: id,
    status: "Active",
    lastMessageAt,
    version: 0,
    createdAt: lastMessageAt,
    updatedAt: lastMessageAt,
    ...overrides,
  };
}

const waiting = (pendingDecisions: number) => ({
  attention: { pendingDecisions, lastTurnFailed: false, unread: false },
});

const agentName = (candidate: AssistantThread) =>
  candidate.agentDefinitionId === "agdef_2" ? "Dispatch desk" : "Billing exceptions";

const shelve = (threads: AssistantThread[], query = "") =>
  assistantShelves(threads, { now: NOW, timezone: "UTC", query, agentName });

/**
 * The assistant's history and its full-screen sidebar list conversations the
 * way the Desk's rail shelves them, by when they were last touched, with one
 * shelf ahead of the calendar: the conversations a change is waiting on.
 */
describe("assistantShelves", () => {
  it("shelves by recency, newest first, leaving empty shelves out", () => {
    const shelves = shelve([
      thread("older", NOW - 3 * DAY),
      thread("morning", NOW - 2 * 60 * 60),
      thread("yesterday", NOW - DAY),
      thread("noon", NOW - 60 * 60),
    ]);

    expect(shelves.map((shelf) => shelf.key)).toEqual(["today", "yesterday", "week"]);
    expect(shelves[0].threads.map((item) => item.id)).toEqual(["noon", "morning"]);
  });

  it("lifts the conversations a change waits on onto their own shelf, first", () => {
    const shelves = shelve([
      thread("quiet", NOW - 60),
      thread("asks", NOW - 2 * DAY, waiting(1)),
      thread("asks-more", NOW - 3 * 60, waiting(3)),
    ]);

    expect(shelves[0]).toEqual({
      key: "waiting",
      threads: [
        expect.objectContaining({ id: "asks-more" }),
        expect.objectContaining({ id: "asks" }),
      ],
    });
    expect(shelves.slice(1).flatMap((shelf) => shelf.threads.map((item) => item.id))).toEqual([
      "quiet",
    ]);
  });

  it("does not shelve a pinned conversation apart; pins are the Desk's", () => {
    const shelves = shelve([thread("pinned", NOW - 60, { pinned: true })]);

    expect(shelves.map((shelf) => shelf.key)).toEqual(["today"]);
  });

  it("finds conversations by title or agent, ignoring case", () => {
    const threads = [
      thread("Stuck loads", NOW - 60),
      thread("Payments due", NOW - 120, { agentDefinitionId: "agdef_2" }),
    ];

    expect(shelve(threads, "STUCK").flatMap((shelf) => shelf.threads.map((t) => t.id))).toEqual([
      "Stuck loads",
    ]);
    expect(shelve(threads, "dispatch").flatMap((shelf) => shelf.threads.map((t) => t.id))).toEqual([
      "Payments due",
    ]);
    expect(shelve(threads, "nothing like it")).toEqual([]);
  });

  it("lists nothing when there is nothing", () => {
    expect(shelve([])).toEqual([]);
  });
});
