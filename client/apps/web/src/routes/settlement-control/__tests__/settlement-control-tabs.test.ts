import { describe, expect, it } from "vitest";
import { settlementControlSchema } from "@/types/settlement-control";
import { settlementControlTabs } from "../settlement-control-tabs";

const RECORD_KEYS = new Set(["id", "version", "organizationId", "businessUnitId"]);

describe("settlement control tabs", () => {
  it("puts every setting the form edits on exactly one tab", () => {
    const placed = settlementControlTabs().flatMap((tab) => tab.fields);
    const editable = Object.keys(settlementControlSchema.shape).filter(
      (key) => !RECORD_KEYS.has(key),
    );

    expect([...placed].sort()).toEqual([...editable].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  it("offers the pay period, review and advances tabs, in that order", () => {
    expect(settlementControlTabs().map((tab) => [tab.value, tab.label])).toEqual([
      ["payPeriod", "Pay period"],
      ["review", "Review & approval"],
      ["deductions", "Advances & escrow"],
    ]);
  });
});
