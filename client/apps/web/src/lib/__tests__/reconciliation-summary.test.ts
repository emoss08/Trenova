import { describe, expect, it } from "vitest";
import { matchRate, reconciliationHasActivity } from "../reconciliation-summary";
import type { ReconciliationSummary } from "@/types/bank-receipt";

function summary(over: Partial<ReconciliationSummary> = {}): ReconciliationSummary {
  return {
    asOfDate: 1_780_000_000,
    importedCount: 0,
    importedAmount: 0,
    matchedCount: 0,
    matchedAmount: 0,
    exceptionCount: 0,
    exceptionAmount: 0,
    activeWorkItemCount: 0,
    assignedWorkItemCount: 0,
    inReviewWorkItemCount: 0,
    exceptionAging: { currentCount: 0, days1To3Count: 0, days4To7Count: 0, daysOver7Count: 0 },
    ...over,
  };
}

describe("reconciliationHasActivity", () => {
  it("reads a summary of zeros as a programme nobody has opened", () => {
    expect(reconciliationHasActivity(summary())).toBe(false);
  });

  // The contract lets the counts disagree: a work item can be open with
  // nothing imported this period, and an amount can be zero with a count
  // above it. Any count above zero is activity; amounts alone are not.
  it("treats any count above zero as activity, whichever it is", () => {
    expect(reconciliationHasActivity(summary({ importedCount: 1 }))).toBe(true);
    expect(reconciliationHasActivity(summary({ inReviewWorkItemCount: 1 }))).toBe(true);
    expect(reconciliationHasActivity(summary({ exceptionCount: 2 }))).toBe(true);
    expect(reconciliationHasActivity(summary({ importedAmount: 12_500 }))).toBe(false);
  });
});

describe("matchRate", () => {
  it("is zero rather than a division by nothing before anything is imported", () => {
    expect(matchRate(summary())).toBe(0);
  });

  it("rounds matched over imported to a whole percentage", () => {
    expect(matchRate(summary({ importedCount: 3, matchedCount: 2 }))).toBe(67);
    expect(matchRate(summary({ importedCount: 8, matchedCount: 8 }))).toBe(100);
  });
});
