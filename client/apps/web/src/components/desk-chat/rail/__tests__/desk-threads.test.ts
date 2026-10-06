import type { AssistantThread } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { groupDeskThreadsByRecency, matchesThreadSearch } from "../desk-threads";

const DAY = 24 * 60 * 60;
// A Wednesday at 15:00 UTC, so "today" and "yesterday" are unambiguous.
const NOW = Date.UTC(2026, 8, 16, 15, 0, 0) / 1000;

function thread(overrides: Partial<AssistantThread> & { id: string }): AssistantThread {
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
    lastMessageAt: NOW - 60,
    version: 0,
    createdAt: NOW - 120,
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
