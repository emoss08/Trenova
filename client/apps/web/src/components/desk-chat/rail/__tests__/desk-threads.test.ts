import type { AssistantThread } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  deskShelfEdges,
  deskShelfFilter,
  deskShelfOf,
  groupDeskThreadsByRecency,
  matchesThreadSearch,
  shelveLatest,
  threadTouchedAt,
} from "../desk-threads";

const DAY = 24 * 60 * 60;
// A Wednesday at 15:00 UTC, so "today" and "yesterday" are unambiguous.
const NOW = Date.UTC(2026, 8, 16, 15, 0, 0) / 1000;

// A conversation is made before its first message, so it is never created after its last one.
function thread(overrides: Partial<AssistantThread> & { id: string }): AssistantThread {
  const lastMessageAt = overrides.lastMessageAt ?? NOW - 60;
  return {
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
    title: "Blocked invoices",
    status: "Active",
    lastMessageAt,
    version: 0,
    createdAt: Math.min(NOW - 120, lastMessageAt),
    updatedAt: NOW - 60,
    ...overrides,
  };
}

const shelved = (groups: ReturnType<typeof groupDeskThreadsByRecency>) =>
  groups.map((group) => [group.key, group.threads.map((item) => item.id)]);

/**
 * The rail is the conversation's table of contents, scanned for "the one
 * from this morning" far more often than searched, so it shelves by how
 * recently each conversation was touched, with what the person pinned held
 * above the calendar. Empty shelves are left out: a single "Older" heading
 * over everything says nothing.
 */
describe("groupDeskThreadsByRecency", () => {
  it("shelves by today, yesterday, this week, this month and older, newest first on each", () => {
    const groups = groupDeskThreadsByRecency(
      [
        thread({ id: "old", lastMessageAt: NOW - 40 * DAY }),
        thread({ id: "month", lastMessageAt: NOW - 12 * DAY }),
        thread({ id: "today-early", lastMessageAt: NOW - 3 * 60 * 60 }),
        thread({ id: "week", lastMessageAt: NOW - 4 * DAY }),
        thread({ id: "yesterday", lastMessageAt: NOW - 1 * DAY }),
        thread({ id: "today-late", lastMessageAt: NOW - 60 }),
        thread({ id: "month-edge", lastMessageAt: NOW - 30 * DAY }),
        thread({ id: "week-edge", lastMessageAt: NOW - 7 * DAY }),
      ],
      NOW,
      { pinnedFirst: true, timezone: "UTC" },
    );

    expect(shelved(groups)).toEqual([
      ["today", ["today-late", "today-early"]],
      ["yesterday", ["yesterday"]],
      ["week", ["week", "week-edge"]],
      ["month", ["month", "month-edge"]],
      ["older", ["old"]],
    ]);
  });

  it("holds pinned conversations above the calendar, newest first, and off the other shelves", () => {
    const groups = groupDeskThreadsByRecency(
      [
        thread({ id: "a", pinned: true, lastMessageAt: NOW - 20 * DAY }),
        thread({ id: "b", lastMessageAt: NOW - 60 }),
        thread({ id: "c", pinned: true, lastMessageAt: NOW - 30 }),
      ],
      NOW,
      { pinnedFirst: true, timezone: "UTC" },
    );

    expect(shelved(groups)).toEqual([
      ["pinned", ["c", "a"]],
      ["today", ["b"]],
    ]);
  });

  it("shelves pinned conversations with the rest when they are not held first", () => {
    const groups = groupDeskThreadsByRecency(
      [
        thread({ id: "a", pinned: true, lastMessageAt: NOW - 20 * DAY }),
        thread({ id: "b", lastMessageAt: NOW - 60 }),
      ],
      NOW,
      { pinnedFirst: false, timezone: "UTC" },
    );

    expect(shelved(groups)).toEqual([
      ["today", ["b"]],
      ["month", ["a"]],
    ]);
  });

  it("omits shelves with nothing on them and returns nothing for nothing", () => {
    expect(groupDeskThreadsByRecency([], NOW, { pinnedFirst: true, timezone: "UTC" })).toEqual([]);

    const groups = groupDeskThreadsByRecency(
      [thread({ id: "a", lastMessageAt: NOW - 90 * DAY })],
      NOW,
      { pinnedFirst: true, timezone: "UTC" },
    );
    expect(groups.map((group) => group.key)).toEqual(["older"]);
  });

  // A thread that has never been written to has no last message; it was just
  // created, which is the most recent thing that could have happened to it.
  it("shelves a conversation with no messages by when it was created", () => {
    const groups = groupDeskThreadsByRecency(
      [thread({ id: "fresh", lastMessageAt: 0, createdAt: NOW - 10 })],
      NOW,
      { pinnedFirst: true, timezone: "UTC" },
    );

    expect(groups[0]?.key).toBe("today");
  });

  // A clock a little ahead of the reader's must not file a conversation
  // under a shelf that does not exist.
  it("shelves a conversation touched a moment from now under today", () => {
    const groups = groupDeskThreadsByRecency(
      [thread({ id: "ahead", lastMessageAt: NOW + 90 })],
      NOW,
      { pinnedFirst: true, timezone: "UTC" },
    );

    expect(groups[0]?.key).toBe("today");
  });

  // 23:50 last night is yesterday to the reader even though it was fewer
  // than 24 hours ago, so the shelf follows their calendar, not the clock.
  it("counts days on the reader's calendar", () => {
    const lateLastNight = Date.UTC(2026, 8, 15, 23, 50, 0) / 1000;
    const groups = groupDeskThreadsByRecency(
      [thread({ id: "late", lastMessageAt: lateLastNight })],
      NOW,
      { pinnedFirst: true, timezone: "UTC" },
    );

    expect(groups[0]?.key).toBe("yesterday");
  });
});

describe("matchesThreadSearch", () => {
  it("matches the title or the agent, case ignored", () => {
    expect(matchesThreadSearch(thread({ id: "a" }), "Billing desk", "BLOCKED")).toBe(true);
    expect(matchesThreadSearch(thread({ id: "a" }), "Billing desk", "billing")).toBe(true);
    expect(matchesThreadSearch(thread({ id: "a" }), "Billing desk", "dispatch")).toBe(false);
    expect(matchesThreadSearch(thread({ id: "a" }), "Billing desk", "  ")).toBe(true);
  });
});

describe("threadTouchedAt", () => {
  it("is the last message, or when the conversation was made if it has none", () => {
    expect(threadTouchedAt({ lastMessageAt: 500, createdAt: 100 })).toBe(500);
    expect(threadTouchedAt({ lastMessageAt: 0, createdAt: 100 })).toBe(100);
  });
});

/**
 * Each shelf is read from the server on its own, so the edges it is read by
 * must be the ones it is drawn by: a conversation is asked for on the shelf
 * it will be shown on, and on no other.
 */
describe("deskShelfEdges", () => {
  it("begins each shelf at the reader's midnight", () => {
    const edges = deskShelfEdges(NOW, "UTC");

    expect(edges).toEqual({
      today: Date.UTC(2026, 8, 16) / 1000,
      yesterday: Date.UTC(2026, 8, 15) / 1000,
      week: Date.UTC(2026, 8, 9) / 1000,
      month: Date.UTC(2026, 7, 17) / 1000,
    });
  });

  it("follows the reader's own calendar, not the browser's", () => {
    // 15:00 UTC is 11:00 in New York, still on the 16th, where midnight was 04:00 UTC.
    expect(deskShelfEdges(NOW, "America/New_York").today).toBe(Date.UTC(2026, 8, 16, 4) / 1000);
    // In Tokyo it is already midnight past on the 17th.
    expect(deskShelfEdges(NOW, "Asia/Tokyo").today).toBe(Date.UTC(2026, 8, 16, 15) / 1000);
  });

  // The day the clocks go back is 25 hours long; yesterday still begins at
  // the midnight before it, not 24 hours before today's.
  it("keeps to midnight across a change of the clocks", () => {
    const afterFallBack = Date.UTC(2026, 10, 2, 17) / 1000;
    const edges = deskShelfEdges(afterFallBack, "America/New_York");

    expect(edges.today).toBe(Date.UTC(2026, 10, 2, 5) / 1000);
    expect(edges.yesterday).toBe(Date.UTC(2026, 10, 1, 4) / 1000);
  });
});

describe("deskShelfFilter", () => {
  const edges = deskShelfEdges(NOW, "UTC");

  it("reads the pinned shelf by its pin alone", () => {
    expect(deskShelfFilter("pinned", edges)).toEqual({ pinned: true });
  });

  it("reads each dated shelf between its own edge and the one above it", () => {
    expect(deskShelfFilter("today", edges)).toEqual({ pinned: false, touchedFrom: edges.today });
    expect(deskShelfFilter("yesterday", edges)).toEqual({
      pinned: false,
      touchedFrom: edges.yesterday,
      touchedBefore: edges.today,
    });
    expect(deskShelfFilter("week", edges)).toEqual({
      pinned: false,
      touchedFrom: edges.week,
      touchedBefore: edges.yesterday,
    });
    expect(deskShelfFilter("month", edges)).toEqual({
      pinned: false,
      touchedFrom: edges.month,
      touchedBefore: edges.week,
    });
    expect(deskShelfFilter("older", edges)).toEqual({
      pinned: false,
      touchedBefore: edges.month,
    });
  });

  it("puts every conversation on the shelf whose range holds it", () => {
    const within = (at: number, filter: ReturnType<typeof deskShelfFilter>) =>
      (filter.touchedFrom === undefined || at >= filter.touchedFrom) &&
      (filter.touchedBefore === undefined || at < filter.touchedBefore);

    for (const at of [
      NOW + 90,
      edges.today,
      edges.today - 1,
      edges.yesterday,
      edges.week,
      edges.week - 1,
      edges.month,
      edges.month - 1,
      1,
    ]) {
      const subject = thread({ id: "t", lastMessageAt: at });
      const key = deskShelfOf(subject, edges, true);
      expect(within(at, deskShelfFilter(key, edges))).toBe(true);
    }
  });
});

/**
 * Each shelf is read at its own time, so a conversation touched since an
 * older shelf was read is held by two lists until that one is read again.
 */
describe("shelveLatest", () => {
  const edges = deskShelfEdges(NOW, "UTC");
  const ids = (shelf: AssistantThread[] | undefined) => shelf?.map((item) => item.id) ?? [];

  it("keeps a conversation once, in its latest copy, on the shelf that copy belongs on", () => {
    const stale = thread({ id: "x", lastMessageAt: NOW - 40 * DAY, updatedAt: NOW - 40 * DAY });
    const fresh = thread({ id: "x", lastMessageAt: NOW - 60, updatedAt: NOW - 60, title: "Now" });

    const shelves = shelveLatest([[fresh], [stale]], edges);

    expect(ids(shelves.get("today"))).toEqual(["x"]);
    expect(shelves.get("today")?.[0].title).toBe("Now");
    expect(shelves.has("older")).toBe(false);
  });

  it("takes the later version when two copies were written in the same second", () => {
    const first = thread({ id: "x", version: 1, title: "First" });
    const second = thread({ id: "x", version: 2, title: "Second" });

    expect(shelveLatest([[second], [first]], edges).get("today")?.[0].title).toBe("Second");
    expect(shelveLatest([[first], [second]], edges).get("today")?.[0].title).toBe("Second");
  });

  it("adds what is known some other way, such as the open conversation", () => {
    const open = thread({ id: "open", lastMessageAt: NOW - 90 * DAY });

    expect(ids(shelveLatest([], edges, [open]).get("older"))).toEqual(["open"]);
  });

  it("holds what the person pinned on the pinned shelf, newest first", () => {
    const shelves = shelveLatest(
      [[thread({ id: "a", pinned: true, lastMessageAt: NOW - 9 * DAY }), thread({ id: "b" })]],
      edges,
    );

    expect(ids(shelves.get("pinned"))).toEqual(["a"]);
    expect(ids(shelves.get("today"))).toEqual(["b"]);
  });
});
