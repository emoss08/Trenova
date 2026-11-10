import type { DeskAttentionItem } from "@/lib/graphql/desk-attention";
import { describe, expect, it } from "vitest";
import { watchtowerItem } from "../../watchtower/__tests__/item-fixture";
import { groupLife, itemLife, outcomeLife } from "../life-badge";

function item(overrides: Partial<DeskAttentionItem> = {}): DeskAttentionItem {
  return { ...watchtowerItem(), deskLens: "Watchtower", aboutId: "shp_1", ...overrides };
}

const working = {
  agentId: "agd_1",
  runId: "arun_1",
  status: "Working" as const,
  step: 1,
  steps: ["Read" as const, "Decide" as const, "Approve" as const],
  note: "",
  failed: false,
  handedAt: 1_800_000_000,
  agent: null,
};

describe("where the work stands", () => {
  it("says nothing of a condition nobody has picked up", () => {
    expect(itemLife(item())).toBeNull();
  });

  it("reads an agent at work as looking into it, and a failed run as not gone through", () => {
    expect(itemLife(item({ agentWork: working }))).toBe("looking");
    expect(itemLife(item({ agentWork: { ...working, failed: true } }))).toBe("failed");
  });

  it("reads a change in the Decisions lens as waiting on the person", () => {
    expect(itemLife(item({ deskLens: "Decisions" }))).toBe("drafted");
  });

  it("puts a change waiting on the person ahead of an agent at work in a row", () => {
    expect(
      groupLife([item({ agentWork: working }), item({ id: "wt_2", deskLens: "Decisions" })]),
    ).toBe("drafted");
  });

  it("maps each settled outcome to its own label", () => {
    expect(
      ["OnItsOwn", "Approved", "Previewed", "Failed"].map((o) => outcomeLife(o as never)),
    ).toEqual(["auto", "done", "simulated", "failed"]);
  });
});
