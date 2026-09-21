import type { AssistantThread } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import { groupDeskThreads, matchesThreadSearch } from "../desk-threads";

function thread(overrides: Partial<AssistantThread>): AssistantThread {
  return {
    id: "athr_1",
    agentDefinitionId: "agtd_1",
    title: "Blocked invoices",
    pinned: false,
    origin: "Desk",
    lastMessageAt: 10,
    createdAt: 1,
    ...overrides,
  } as AssistantThread;
}

const agentNames = new Map([
  ["agtd_1", "Billing desk"],
  ["agtd_2", "Dispatch desk"],
]);

/**
 * The rail shows what a person pinned on top, then the rest grouped by the
 * agent they were talking to, newest first inside each group, so a day's
 * conversations with the billing desk sit together.
 */
describe("groupDeskThreads", () => {
  it("puts pinned conversations first, newest first", () => {
    const groups = groupDeskThreads(
      [
        thread({ id: "a", pinned: true, lastMessageAt: 5 }),
        thread({ id: "b", pinned: false, lastMessageAt: 9 }),
        thread({ id: "c", pinned: true, lastMessageAt: 7 }),
      ],
      agentNames,
    );
    expect(groups[0].kind).toBe("pinned");
    expect(groups[0].threads.map((item) => item.id)).toEqual(["c", "a"]);
  });

  it("groups the rest by agent, the busiest group first", () => {
    const groups = groupDeskThreads(
      [
        thread({ id: "a", agentDefinitionId: "agtd_2", lastMessageAt: 3 }),
        thread({ id: "b", agentDefinitionId: "agtd_1", lastMessageAt: 9 }),
        thread({ id: "c", agentDefinitionId: "agtd_2", lastMessageAt: 8 }),
      ],
      agentNames,
    );
    expect(groups.map((group) => group.label)).toEqual(["Dispatch desk", "Billing desk"]);
    expect(groups[0].threads.map((item) => item.id)).toEqual(["c", "a"]);
  });

  it("names a group for an agent it cannot find", () => {
    const groups = groupDeskThreads([thread({ agentDefinitionId: "gone" })], agentNames);
    expect(groups[0].label).toBe("");
    expect(groups[0].agentId).toBe("gone");
  });

  it("returns nothing for nothing", () => {
    expect(groupDeskThreads([], agentNames)).toEqual([]);
  });
});

describe("matchesThreadSearch", () => {
  it("matches the title or the agent, case ignored", () => {
    expect(matchesThreadSearch(thread({}), "Billing desk", "BLOCKED")).toBe(true);
    expect(matchesThreadSearch(thread({}), "Billing desk", "billing")).toBe(true);
    expect(matchesThreadSearch(thread({}), "Billing desk", "dispatch")).toBe(false);
    expect(matchesThreadSearch(thread({}), "Billing desk", "  ")).toBe(true);
  });
});
