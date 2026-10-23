import { describe, expect, it } from "vitest";
import { receivablesHaveActivity } from "../ar-dashboard";

const NEVER_INVOICED = { hasInvoiced: false, unappliedCashMinor: 0 };

describe("receivablesHaveActivity", () => {
  it("is false for a tenant that has never invoiced", () => {
    expect(receivablesHaveActivity(NEVER_INVOICED)).toBe(false);
  });

  // A clear book with history keeps its dashboard: having invoiced at all is
  // enough, whatever is open today.
  it("is true once any invoice has posted, even with nothing open", () => {
    expect(receivablesHaveActivity({ ...NEVER_INVOICED, hasInvoiced: true })).toBe(true);
  });

  it("is true when cash is waiting to be applied before any invoice", () => {
    expect(receivablesHaveActivity({ ...NEVER_INVOICED, unappliedCashMinor: 100 })).toBe(true);
  });
});
