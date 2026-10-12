import { describe, expect, it } from "vitest";
import { INITIAL_DRY_RUN, reduceDryRun, type DryRunState } from "../dry-run-model";

function fold(events: [string, unknown][]): DryRunState {
  return events.reduce(
    (state, [event, data]) =>
      reduceDryRun(state, event, data === undefined ? "" : JSON.stringify(data)),
    INITIAL_DRY_RUN,
  );
}

describe("reduceDryRun", () => {
  it("shows each call while it runs, then what it came to", () => {
    const state = fold([
      ["tool_started", { callId: "c1", name: "search_shipments" }],
      ["tool_started", { callId: "c2", name: "assign_driver" }],
      ["tool_finished", { callId: "c1", name: "search_shipments" }],
      ["delta", { text: "Marcus Reed " }],
      ["delta", { text: "is closest." }],
    ]);
    expect(state.status).toBe("running");
    expect(state.steps).toEqual([
      { callId: "c1", tool: "search_shipments", outcome: "Runs", summary: "" },
      { callId: "c2", tool: "assign_driver", outcome: null, summary: "" },
    ]);
    expect(state.reply).toBe("Marcus Reed is closest.");
  });

  it("lets the closing account replace the outcomes, keeping the streamed reply when it has none", () => {
    const state = fold([
      ["tool_started", { callId: "c1", name: "search_shipments" }],
      ["tool_started", { callId: "c2", name: "assign_driver" }],
      ["delta", { text: "Streamed." }],
      [
        "dry_run_steps",
        {
          steps: [
            { callId: "c1", tool: "search_shipments", outcome: "Runs" },
            { callId: "c2", tool: "assign_driver", outcome: "AskFirst", summary: "Assign Marcus" },
            { callId: "c3", tool: "tender", outcome: "SomethingNew" },
          ],
          reply: "",
        },
      ],
      ["done", { reply: "" }],
    ]);
    expect(state.status).toBe("done");
    expect(state.steps.map((step) => step.outcome)).toEqual(["Runs", "AskFirst", "Runs"]);
    expect(state.steps[1].summary).toBe("Assign Marcus");
    expect(state.reply).toBe("Streamed.");
  });

  it("swaps the streamed reply for the corrected one, without a second copy", () => {
    const state = fold([
      ["delta", { text: "PRO-1 (ID shp_01) is late." }],
      ["reply_regrounded", { action: "strip_ids", reason: "ids" }],
      ["reply_replaced", { text: "PRO-1 is late.", reason: "ids" }],
    ]);
    expect(state.reply).toBe("PRO-1 is late.");
    expect(state.status).toBe("running");
  });

  it("leaves out what a delegate did, and a call it already has", () => {
    const state = fold([
      ["tool_started", { callId: "c1", name: "search_shipments" }],
      ["tool_started", { callId: "c1", name: "search_shipments" }],
      ["tool_started", { callId: "d1", name: "get_invoice", agentId: "agdef_other" }],
      ["delta", { text: "their words", agentId: "agdef_other" }],
    ]);
    expect(state.steps).toHaveLength(1);
    expect(state.reply).toBe("");
  });

  it("marks a failed call, and a failure that ends the run stays failed after done", () => {
    const state = fold([
      ["tool_finished", { callId: "c9", name: "assign_driver", failed: true }],
      ["error", { message: "No model answered" }],
      ["done", undefined],
    ]);
    expect(state.steps[0].outcome).toBe("Failed");
    expect(state.status).toBe("failed");
    expect(state.error).toBe("No model answered");
  });

  it("ignores events it does not know and data it cannot read", () => {
    const before = fold([["delta", { text: "a" }]]);
    expect(reduceDryRun(before, "context", "{}")).toBe(before);
    expect(reduceDryRun(before, "delta", "not json")).toBe(before);
    expect(reduceDryRun(before, "error", "not json").error).toBe(
      "The dry run could not finish. Try again in a moment.",
    );
  });
});
