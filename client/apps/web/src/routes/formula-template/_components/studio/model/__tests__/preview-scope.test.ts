import type { TestExpressionRequest } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import { scopeTarget } from "../preview-scope";

// The request leaves out blank breakdown lines (formulaTestRequest), so the
// server's breakdowns[n] is the n-th line it was sent, not the n-th in the form.
const request: TestExpressionRequest = {
  expression: "linehaul + fuel",
  schemaId: "shipment",
  variables: {},
  breakdowns: [
    { name: "linehaul", label: "Linehaul", expression: "baseRate * totalDistance" },
    { name: "fuel", label: "Fuel", expression: "weight * 0.1" },
  ],
};

const form = [
  { name: "linehaul", label: "Linehaul", expression: "baseRate * totalDistance" },
  { name: "", label: "", expression: "" },
  { name: "fuel", label: "Fuel", expression: "weight * 0.1" },
];

describe("scopeTarget", () => {
  it("puts the main expression's findings on the expression", () => {
    expect(scopeTarget("expression", request, form)).toEqual({
      path: "expression",
      checked: "linehaul + fuel",
    });
  });

  it("finds a breakdown line by its key, past a blank line the request left out", () => {
    expect(scopeTarget("breakdowns[1].expression", request, form)).toEqual({
      path: "breakdownDefinitions.2.expression",
      checked: "weight * 0.1",
    });
  });

  it("names nothing for a line the form no longer holds, or before any preview", () => {
    expect(scopeTarget("breakdowns[1].expression", request, form.slice(0, 2))).toBeNull();
    expect(scopeTarget("breakdowns[5].expression", request, form)).toBeNull();
    expect(scopeTarget("expression", null, form)).toBeNull();
  });
});
