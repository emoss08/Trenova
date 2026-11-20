import { describe, expect, it } from "vitest";
import { checksNetPayVariance, settlementControlSchema } from "@/types/settlement-control";
import { savedSettlementControl } from "./settlement-control-fixture";

describe("settlement control schema", () => {
  it("accepts the record the server sends, decimals and all", () => {
    expect(settlementControlSchema.parse(savedSettlementControl)).toEqual(savedSettlementControl);
  });

  // The server stores both percentages as NUMERIC(7, 4): a fifth decimal or a threshold of
  // 1,000% fails in the database, past the validator.
  it.each([
    ["varianceThresholdPct", "25.12345"],
    ["varianceThresholdPct", "1000"],
    ["varianceThresholdPct", "-1"],
    ["defaultEscrowInterestRate", "100.5"],
    ["defaultEscrowInterestRate", "-0.5"],
  ])("refuses %s of %s", (field, value) => {
    const result = settlementControlSchema.safeParse({ ...savedSettlementControl, [field]: value });

    expect(result.success).toBe(false);
    expect(result.error?.issues.map((issue) => issue.path)).toEqual([[field]]);
  });

  it.each([
    ["escrowInterestFrequencyMonths", 4],
    ["escrowInterestFrequencyMonths", 0],
    ["varianceLookbackWeeks", 53],
    ["varianceLookbackWeeks", 0],
    ["payDelayDays", 31],
    ["periodEndDayOfWeek", 7],
  ])("refuses %s of %d, as the server's validator does", (field, value) => {
    const result = settlementControlSchema.safeParse({ ...savedSettlementControl, [field]: value });

    expect(result.error?.issues.map((issue) => issue.path)).toEqual([[field]]);
  });
});

describe("checksNetPayVariance", () => {
  it.each([
    ["25", true],
    ["0", false],
    ["0.0000", false],
    ["0.0001", true],
    ["", true],
    [null, true],
  ])("reads a threshold of %j as a check that runs: %s", (threshold, runs) => {
    expect(checksNetPayVariance(threshold)).toBe(runs);
  });
});
