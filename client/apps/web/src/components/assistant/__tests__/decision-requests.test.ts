import { describe, expect, it } from "vitest";
import type { ToolStep } from "../activity";
import { decisionRequestOf, decisionRequestsFromSteps } from "../decision-requests";

describe("decisionRequestsFromSteps", () => {
  it("reads a live turn's finished calls only, each once", () => {
    const step = (
      id: string,
      status: ToolStep["status"],
      args: Record<string, unknown>,
    ): ToolStep => ({
      id,
      name: "request_decision",
      arguments: args,
      status,
      content: "",
      effect: "ask",
      summary: "",
      durationSeconds: null,
    });

    expect(
      decisionRequestsFromSteps([
        step("1", "done", { proposalId: "aprop_1" }),
        step("2", "running", { proposalId: "aprop_2" }),
        step("3", "failed", { proposalId: "aprop_3" }),
        step("4", "done", { proposalId: "aprop_1" }),
        step("5", "done", { planId: "apl_1" }),
        step("6", "done", { proposalIds: ["aprop_4", "aprop_5", "aprop_4", 7] }),
      ]),
    ).toEqual([
      { callId: "1", proposalIds: ["aprop_1"], planId: "" },
      { callId: "5", proposalIds: [], planId: "apl_1" },
      { callId: "6", proposalIds: ["aprop_4", "aprop_5"], planId: "" },
    ]);
  });

  it("reads an artifact's payload the same way", () => {
    expect(decisionRequestOf({ proposalId: "aprop_1", proposalIds: [] })).toEqual({
      proposalIds: ["aprop_1"],
      planId: "",
    });
    expect(decisionRequestOf({})).toBeNull();
  });
});
