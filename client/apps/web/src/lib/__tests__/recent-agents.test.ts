import { describe, expect, it } from "vitest";
import { agentRecency } from "../recent-agents";

const thread = (agentDefinitionId: string, lastMessageAt: number, createdAt = 1) => ({
  agentDefinitionId,
  lastMessageAt,
  createdAt,
});

describe("agentRecency", () => {
  it("orders agents by their most recent conversation", () => {
    const recency = agentRecency(
      [thread("agdef_a", 100), thread("agdef_b", 300), thread("agdef_a", 500)],
      { limit: 5 },
    );

    expect(recency.ids).toEqual(["agdef_a", "agdef_b"]);
    expect(recency.lastUsedAt.get("agdef_a")).toBe(500);
  });

  it("counts an empty conversation from when it was opened", () => {
    const recency = agentRecency([thread("agdef_a", 0, 900), thread("agdef_b", 400)], {
      limit: 5,
    });

    expect(recency.ids).toEqual(["agdef_a", "agdef_b"]);
  });

  it("puts the agent last asked first, even without a kept conversation", () => {
    const recency = agentRecency([thread("agdef_a", 100)], { limit: 5, preferId: "agdef_z" });

    expect(recency.ids).toEqual(["agdef_z", "agdef_a"]);
  });

  it("keeps to the limit and skips conversations with no agent", () => {
    const recency = agentRecency(
      [thread("", 900), thread("agdef_a", 100), thread("agdef_b", 200), thread("agdef_c", 300)],
      { limit: 2 },
    );

    expect(recency.ids).toEqual(["agdef_c", "agdef_b"]);
  });
});
