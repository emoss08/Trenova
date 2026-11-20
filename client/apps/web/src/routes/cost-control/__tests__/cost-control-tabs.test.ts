import { describe, expect, it } from "vitest";
import { costControlSchema } from "@/types/cost-control";
import { costControlTabs } from "../cost-control-tabs";

const RECORD_KEYS = new Set(["version"]);

describe("cost control tabs", () => {
  it("puts every setting the form edits on exactly one tab", () => {
    const placed = costControlTabs({ accounting: true }).flatMap((tab) => tab.fields);
    const editable = Object.keys(costControlSchema.shape).filter((key) => !RECORD_KEYS.has(key));

    expect([...placed].sort()).toEqual([...editable].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  it("offers every tab, in order, when the plan includes accounting", () => {
    expect(costControlTabs({ accounting: true }).map((tab) => tab.value)).toEqual([
      "basis",
      "rates",
      "actuals",
    ]);
  });

  it("leaves out GL actuals when the plan has no ledger to read them from", () => {
    expect(costControlTabs({ accounting: false }).map((tab) => tab.value)).toEqual([
      "basis",
      "rates",
    ]);
  });
});
