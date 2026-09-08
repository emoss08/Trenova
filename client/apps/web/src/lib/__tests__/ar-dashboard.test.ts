import { describe, expect, it } from "vitest";
import { receivablesHaveActivity } from "../ar-dashboard";

const CLEAR = { openInvoiceCount: 0, totalOpenMinor: 0, unappliedCashMinor: 0 };
const QUIET_WEEK = { billedMinor: 0, arBalanceMinor: 0 };

describe("receivablesHaveActivity", () => {
  it("is false for a tenant that has never invoiced", () => {
    expect(receivablesHaveActivity(CLEAR, [QUIET_WEEK, QUIET_WEEK])).toBe(false);
    expect(receivablesHaveActivity(CLEAR, [])).toBe(false);
  });

  it("is true while an invoice is open, even with a quiet trend", () => {
    expect(receivablesHaveActivity({ ...CLEAR, openInvoiceCount: 1 }, [QUIET_WEEK])).toBe(true);
  });

  // A credit balance is still a balance; the sign must not hide it.
  it("is true for an open balance of either sign", () => {
    expect(receivablesHaveActivity({ ...CLEAR, totalOpenMinor: -500 }, [QUIET_WEEK])).toBe(true);
  });

  it("is true when cash is waiting to be applied", () => {
    expect(receivablesHaveActivity({ ...CLEAR, unappliedCashMinor: 100 }, [QUIET_WEEK])).toBe(true);
  });

  // A clear book with history keeps its dashboard: billing in any week, or a
  // balance carried in any week, counts.
  it("is true when any week of the trend carried billing or a balance", () => {
    expect(
      receivablesHaveActivity(CLEAR, [QUIET_WEEK, { billedMinor: 1, arBalanceMinor: 0 }]),
    ).toBe(true);
    expect(receivablesHaveActivity(CLEAR, [{ billedMinor: 0, arBalanceMinor: 250 }])).toBe(true);
  });
});
