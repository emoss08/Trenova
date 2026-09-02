import { describe, expect, it } from "vitest";
import { breakdownErrorsByPath, reconcileBreakdown } from "../breakdown-reconcile";

const lines = [
  { name: "linehaul", label: "Linehaul", amount: 1000 },
  { name: "fuel", label: "Fuel", amount: 180 },
];

describe("reconcileBreakdown", () => {
  it("reports the sum and what the total leaves unallocated", () => {
    const result = reconcileBreakdown({ total: 1250, rawAmount: 1250, lines });
    expect(result.sum).toBe(1180);
    expect(result.residual).toBe(70);
    expect(result.balanced).toBe(false);
    expect(result.clampMismatch).toBe(false);
  });

  it("is balanced when the lines add up to the total within a cent", () => {
    const result = reconcileBreakdown({
      total: 1180.004,
      rawAmount: 1180.004,
      lines,
    });
    expect(result.balanced).toBe(true);
    expect(result.residual).toBe(0);
  });

  it("flags lines that still sum to the raw amount after a guardrail clamp", () => {
    const result = reconcileBreakdown({
      total: 1500,
      rawAmount: 1180,
      guardrailApplied: true,
      lines,
    });
    expect(result.clampMismatch).toBe(true);
    expect(result.residual).toBe(320);
  });

  it("ignores lines that failed to evaluate and says so", () => {
    const result = reconcileBreakdown({
      total: 1000,
      rawAmount: 1000,
      lines: [...lines, { name: "broken", label: "Broken", amount: 0, error: "boom" }],
    });
    expect(result.sum).toBe(1180);
    expect(result.failedCount).toBe(1);
  });
});

describe("breakdownErrorsByPath", () => {
  it("maps a failed line to the form path of the matching definition", () => {
    const definitions = [
      { name: "linehaul", label: "Linehaul", expression: "1" },
      { name: "fuel", label: "Fuel", expression: "lookup('x', 1)" },
    ];
    const items = [
      { name: "linehaul", label: "Linehaul", amount: 1 },
      { name: "fuel", label: "Fuel", amount: 0, error: "rate table not found" },
    ];
    expect(breakdownErrorsByPath(items, definitions)).toEqual({
      "breakdownDefinitions.1.expression": "rate table not found",
    });
  });
});
