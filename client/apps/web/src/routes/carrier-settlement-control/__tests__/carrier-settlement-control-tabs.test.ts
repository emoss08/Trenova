import { describe, expect, it } from "vitest";
import { carrierSettlementControlSchema } from "@/types/carrier-settlement-control";
import { carrierSettlementControlTabs } from "../carrier-settlement-control-tabs";

const RECORD_KEYS = new Set(["id", "version", "organizationId", "businessUnitId"]);

describe("carrier settlement control tabs", () => {
  it("puts every setting the form edits on exactly one tab", () => {
    const placed = carrierSettlementControlTabs().flatMap((tab) => tab.fields);
    const editable = Object.keys(carrierSettlementControlSchema.shape).filter(
      (key) => !RECORD_KEYS.has(key),
    );

    expect([...placed].sort()).toEqual([...editable].sort());
    expect(new Set(placed).size).toBe(placed.length);
  });

  it("offers the pay period, invoice matching, and approval and posting, in that order", () => {
    expect(carrierSettlementControlTabs().map((tab) => tab.value)).toEqual([
      "payPeriod",
      "matching",
      "approval",
    ]);
  });
});
