import type { AssistantThread } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { groupThreadsByRecency } from "../thread-grouping";

const DAY = 24 * 60 * 60;
// A Wednesday at 15:00 UTC, so "today" and "yesterday" are unambiguous.
const NOW = Date.UTC(2026, 8, 16, 15, 0, 0) / 1000;

function thread(id: string, lastMessageAt: number): AssistantThread {
  return {
    id,
    businessUnitId: "bu",
    organizationId: "org",
    userId: "u",
    agentDefinitionId: "agdef_1",
    preferredProviderId: "",
    title: id,
    status: "Active",
    lastMessageAt,
    version: 0,
    createdAt: lastMessageAt,
    updatedAt: lastMessageAt,
  };
}

/**
 * The sidebar is scanned for "the one from this morning" far more often than
 * searched, so conversations are shelved by how recently they were touched.
 * Empty shelves are left out; a list with a single "Older" heading over
 * everything says nothing.
 */
describe("groupThreadsByRecency", () => {
  it("shelves by today, yesterday, this week and older, newest first within each", () => {
    const groups = groupThreadsByRecency(
      [
        thread("old", NOW - 40 * DAY),
        thread("today-early", NOW - 3 * 60 * 60),
        thread("week", NOW - 4 * DAY),
        thread("yesterday", NOW - 1 * DAY),
        thread("today-late", NOW - 60),
      ],
      NOW,
      "UTC",
    );

    expect(groups.map((group) => [group.label, group.threads.map((t) => t.id)])).toEqual([
      ["Today", ["today-late", "today-early"]],
      ["Yesterday", ["yesterday"]],
      ["Previous 7 days", ["week"]],
      ["Older", ["old"]],
    ]);
  });

  it("omits shelves with nothing on them", () => {
    const groups = groupThreadsByRecency([thread("a", NOW - 30 * DAY)], NOW, "UTC");

    expect(groups.map((group) => group.label)).toEqual(["Older"]);
  });

  // A thread that has never been written to has no last message; it was just
  // created, which is the most recent thing that could have happened to it.
  it("treats a thread with no messages as touched when it was created", () => {
    const fresh = { ...thread("fresh", NOW - 10), lastMessageAt: 0 };
    const groups = groupThreadsByRecency([fresh], NOW, "UTC");

    expect(groups[0]?.label).toBe("Today");
  });
});
