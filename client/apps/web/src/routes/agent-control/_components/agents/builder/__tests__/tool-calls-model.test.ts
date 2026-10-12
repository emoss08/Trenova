import type { AgentToolVerdictRow } from "@/lib/graphql/agent-scorecard";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { describe, expect, it } from "vitest";
import {
  formatFailingShare,
  toolCallRecords,
  toolCallTotals,
  verdictLabel,
  verdictTone,
  wentThrough,
} from "../tool-calls-model";

/** One row as AgentScorecard.toolVerdicts carries it: a tool, a verdict, a count and its reasons. */
function row(
  toolName: string,
  verdict: string,
  calls: number,
  topReasons: AgentToolVerdictRow["topReasons"] = [],
): AgentToolVerdictRow {
  return { toolName, verdict, calls, topReasons };
}

const t: TranslateFn = (message) => message ?? "";

describe("toolCallRecords", () => {
  it("gathers each tool's verdicts and counts the calls that did not go through", () => {
    const [record] = toolCallRecords([
      row("update_shipment", "ran", 6),
      row("update_shipment", "invalid", 3, [{ reason: "stop 2 has no location", calls: 3 }]),
      row("update_shipment", "proposed", 2),
      row("update_shipment", "failed", 1, [{ reason: "version conflict", calls: 1 }]),
    ]);

    expect(record.toolName).toBe("update_shipment");
    expect(record.calls).toBe(12);
    expect(record.failing).toBe(4);
    expect(record.failingShare).toBeCloseTo(4 / 12);
    expect(record.verdicts.map((tally) => tally.verdict)).toEqual([
      "ran",
      "proposed",
      "invalid",
      "failed",
    ]);
  });

  it("counts a preview as going through, and a refusal, failure or unknown outcome as not", () => {
    const [record] = toolCallRecords([
      row("assign_tractor", "simulated", 4),
      row("assign_tractor", "denied", 1),
      row("assign_tractor", "duplicate", 1),
      row("assign_tractor", "over_budget", 1),
      row("assign_tractor", "unknown", 1),
    ]);

    expect(record.calls).toBe(8);
    expect(record.failing).toBe(4);
  });

  it("puts the tool with the most calls that did not go through first, whatever its volume", () => {
    const records = toolCallRecords([
      row("list_shipments", "ran", 400),
      row("list_shipments", "invalid", 2),
      row("update_shipment", "ran", 3),
      row("update_shipment", "invalid", 5),
      row("get_customer", "ran", 50),
    ]);

    expect(records.map((record) => record.toolName)).toEqual([
      "update_shipment",
      "list_shipments",
      "get_customer",
    ]);
  });

  it("breaks a tie on failures by the higher share, then the busier tool, then the name", () => {
    const records = toolCallRecords([
      row("b_tool", "failed", 2),
      row("b_tool", "ran", 2),
      row("a_tool", "failed", 2),
      row("a_tool", "ran", 2),
      row("c_tool", "failed", 2),
      row("c_tool", "ran", 8),
      row("d_tool", "failed", 2),
    ]);

    expect(records.map((record) => record.toolName)).toEqual([
      "d_tool",
      "a_tool",
      "b_tool",
      "c_tool",
    ]);
  });

  it("folds two rows for the same verdict into one, adding up a reason given in both", () => {
    const [record] = toolCallRecords([
      row("update_shipment", "invalid", 3, [
        { reason: "missing stop", calls: 2 },
        { reason: "bad date", calls: 1 },
      ]),
      row("update_shipment", "invalid", 2, [
        { reason: "bad date", calls: 2 },
      ]),
    ]);

    expect(record.verdicts).toEqual([
      {
        verdict: "invalid",
        calls: 5,
        reasons: [
          { reason: "bad date", calls: 3 },
          { reason: "missing stop", calls: 2 },
        ],
      },
    ]);
  });

  it("orders a verdict's reasons most frequent first", () => {
    const [record] = toolCallRecords([
      row("update_shipment", "denied", 4, [
        { reason: "no permission to update shipments", calls: 1 },
        { reason: "outside the agent's data access", calls: 3 },
      ]),
    ]);

    expect(record.verdicts[0].reasons.map((reason) => reason.reason)).toEqual([
      "outside the agent's data access",
      "no permission to update shipments",
    ]);
  });

  it("leaves out rows with no calls and has nothing to say for an empty scorecard", () => {
    expect(toolCallRecords([row("get_customer", "ran", 0)])).toEqual([]);
    expect(toolCallRecords([])).toEqual([]);
  });

  it("sorts a verdict this client has not heard of after the ones it knows, and counts it as failing", () => {
    const [record] = toolCallRecords([
      row("get_customer", "rate_limited", 1),
      row("get_customer", "failed", 1),
      row("get_customer", "ran", 1),
    ]);

    expect(record.verdicts.map((tally) => tally.verdict)).toEqual([
      "ran",
      "failed",
      "rate_limited",
    ]);
    expect(record.failing).toBe(2);
  });
});

describe("toolCallTotals", () => {
  it("adds every tool's calls and failures", () => {
    const records = toolCallRecords([
      row("a", "ran", 3),
      row("a", "failed", 1),
      row("b", "invalid", 2),
    ]);

    expect(toolCallTotals(records)).toEqual({ calls: 6, failing: 3 });
    expect(toolCallTotals([])).toEqual({ calls: 0, failing: 0 });
  });
});

describe("verdicts", () => {
  it("says only run, proposed and simulated calls went through", () => {
    expect(["ran", "proposed", "simulated"].every(wentThrough)).toBe(true);
    expect(
      ["denied", "invalid", "duplicate", "over_budget", "failed", "unknown", ""].some(wentThrough),
    ).toBe(false);
  });

  it("names refusals in the words the conversation uses", () => {
    expect(verdictLabel("invalid", t)).toBe("Not accepted");
    expect(verdictLabel("denied", t)).toBe("Not permitted");
    expect(verdictLabel("over_budget", t)).toBe("Out of budget");
    expect(verdictLabel("duplicate", t)).toBe("Skipped (repeat)");
    expect(verdictLabel("ran", t)).toBe("Ran");
    expect(verdictLabel("unknown", t)).toBe("Outcome unknown");
    expect(verdictLabel("rate_limited", t)).toBe("Rate limited");
  });

  it("tones a failure worst, a refusal to fix as a warning and a skipped repeat as neutral", () => {
    expect(verdictTone("ran")).toBe("success");
    expect(verdictTone("proposed")).toBe("info");
    expect(verdictTone("simulated")).toBe("neutral");
    expect(verdictTone("duplicate")).toBe("neutral");
    expect(verdictTone("invalid")).toBe("warning");
    expect(verdictTone("failed")).toBe("danger");
    expect(verdictTone("unknown")).toBe("danger");
    expect(verdictTone("rate_limited")).toBe("danger");
  });
});

describe("formatFailingShare", () => {
  it("never rounds a share above zero down to nothing", () => {
    expect(formatFailingShare(0)).toBe("0%");
    expect(formatFailingShare(0.001)).toBe("1%");
    expect(formatFailingShare(0.25)).toBe("25%");
    expect(formatFailingShare(1)).toBe("100%");
    expect(formatFailingShare(Number.NaN)).toBe("0%");
  });
});
