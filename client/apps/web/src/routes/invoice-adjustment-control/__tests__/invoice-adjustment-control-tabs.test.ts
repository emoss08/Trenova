import { describe, expect, it } from "vitest";
import { invoiceAdjustmentControlSchema } from "@/types/invoice-adjustment-control";
import { invoiceAdjustmentControlTabs } from "../invoice-adjustment-control-tabs";

const RECORD_KEYS = new Set([
  "id",
  "version",
  "createdAt",
  "updatedAt",
  "organizationId",
  "businessUnitId",
]);

describe("invoice adjustment control tabs", () => {
  it("puts every setting the form edits on exactly one tab", () => {
    const placed = invoiceAdjustmentControlTabs().flatMap((tab) => tab.fields);
    const editable = Object.keys(invoiceAdjustmentControlSchema.shape).filter(
      (key) => !RECORD_KEYS.has(key),
    );

    expect([...placed].sort()).toEqual([...editable].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  it("offers every tab to every organization, eligibility first", () => {
    expect(invoiceAdjustmentControlTabs().map((tab) => [tab.value, tab.label])).toEqual([
      ["eligibility", "Eligibility"],
      ["approval", "Approval"],
      ["credits", "Credits & rebills"],
    ]);
  });
});
