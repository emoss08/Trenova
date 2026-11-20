import { describe, expect, it } from "vitest";
import { carrierSettlementControlSchema } from "@/types/carrier-settlement-control";
import { savedCarrierSettlementControl } from "./carrier-settlement-control-fixture";

function issues(overrides: Partial<Record<string, unknown>>) {
  const result = carrierSettlementControlSchema.safeParse({
    ...savedCarrierSettlementControl,
    ...overrides,
  });
  return result.success ? [] : result.error.issues.map((issue) => issue.message);
}

describe("carrier settlement control schema", () => {
  it("reads a saved record as the query sends it", () => {
    expect(issues({})).toEqual([]);
  });

  // The server refuses a pay delay outside 0 to 30 days.
  it.each([
    [30, []],
    [31, ["Pay delay cannot exceed 30 days"]],
    [-1, ["Pay delay cannot be negative"]],
  ])("takes a pay delay of %i days as the server does", (payDelayDays, expected) => {
    expect(issues({ payDelayDays })).toEqual(expected);
  });

  it("refuses a negative variance tolerance", () => {
    expect(issues({ varianceToleranceMinor: -1 })).toEqual([
      "Variance tolerance cannot be negative",
    ]);
  });

  it("refuses a pay trigger the server does not know", () => {
    expect(issues({ payTrigger: "Dispatched" })).toHaveLength(1);
  });
});
