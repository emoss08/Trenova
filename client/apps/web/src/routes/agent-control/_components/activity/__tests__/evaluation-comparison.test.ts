import { describe, expect, it } from "vitest";
import { readComparison, summarizeComparison } from "../evaluation-comparison";

const t = (text: string, ...args: (string | number)[]) =>
  text.replace(/\{(\d+)\}/g, (_m, i) => String(args[Number(i)]));

describe("readComparison", () => {
  it("reads the stored shape and drops what it cannot place", () => {
    const comparison = readComparison({
      matches: [
        { toolName: "assign_move", verdict: "Agreed", originalOutcome: "accepted" },
        { toolName: "cancel_shipment", verdict: "Made up" },
        "nonsense",
        {
          toolName: "assign_move",
          verdict: "Changed",
          changes: [{ field: "primaryWorkerId", from: "wrk_a", to: "wrk_b" }],
        },
      ],
      agreed: 1,
      changed: 1,
      decided: 1,
      score: 1,
    });

    expect(comparison).not.toBeNull();
    expect(comparison?.matches.map((match) => match.verdict)).toEqual(["Agreed", "Changed"]);
    expect(comparison?.matches[1].changes).toEqual([
      { field: "primaryWorkerId", from: "wrk_a", to: "wrk_b" },
    ]);
    expect(comparison?.regressed).toBe(0);
    expect(comparison?.score).toBe(1);
  });

  it("is null for an evaluation that has not finished", () => {
    expect(readComparison(null)).toBeNull();
    expect(readComparison("x")).toBeNull();
  });
});

describe("summarizeComparison", () => {
  // Regressions are the reason anyone opens the list, so they come first.
  it("puts the bad news first and says when there is nothing", () => {
    const comparison = readComparison({ agreed: 2, regressed: 1, added: 1, matches: [] });
    expect(summarizeComparison(comparison!, t)).toBe("1 regressed · 2 agreed · 1 added");
    expect(summarizeComparison(readComparison({ matches: [] })!, t)).toBe("Nothing to compare");
  });
});
