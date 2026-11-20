import { describe, expect, it } from "vitest";
import { billingControlSchema } from "@/types/billing-control";
import { billingControlTabs } from "../billing-control-tabs";

const RECORD_KEYS = new Set([
  "id",
  "version",
  "createdAt",
  "updatedAt",
  "organizationId",
  "businessUnitId",
]);

describe("billing control tabs", () => {
  it("puts every setting the form edits on exactly one tab", () => {
    const placed = billingControlTabs().flatMap((tab) => tab.fields);
    const editable = Object.keys(billingControlSchema.shape).filter((key) => !RECORD_KEYS.has(key));

    expect([...placed].sort()).toEqual([...editable].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  it("offers one tab per job, invoices first", () => {
    expect(billingControlTabs().map((tab) => [tab.value, tab.label])).toEqual([
      ["invoices", "Invoices"],
      ["automation", "Automation"],
      ["exceptions", "Exceptions"],
      ["rating", "Rating"],
      ["collections", "Collections"],
    ]);
  });
});
