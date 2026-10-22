import {
  allocationRemainders,
  chargeTotals,
  splitAllocations,
  type AllocationChargesInput,
  type AllocationSplitInput,
} from "@trenova/shared/rules/allocation";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { corpusCase, expectCorpusOutput } from "@trenova/shared/rules/__tests__/corpus";
import { describe, expect, it } from "vitest";

function splitCase(name: string) {
  const testCase = corpusCase("allocation", name);
  expectCorpusOutput(
    testCase,
    splitAllocations(rulesSync(), testCase.input as AllocationSplitInput),
  );
}

function chargesCase(name: string, run: typeof chargeTotals | typeof remainderOutput) {
  const testCase = corpusCase("allocation", name);
  expectCorpusOutput(testCase, run(rulesSync(), testCase.input as AllocationChargesInput));
}

function remainderOutput(...args: Parameters<typeof allocationRemainders>) {
  return { charges: allocationRemainders(...args) };
}

describe("splitAllocations", () => {
  it("splits a half-cent charge into shares that add up to its rounded cents", () => {
    splitCase("fix: a sub-cent total still splits into shares that add up to its rounded cents");
  });

  it("refuses a charge split by percent on one row and amount on another", () => {
    splitCase("resolver: mixed methods on one charge are refused");
  });

  it("sums each payer's freight and accessorial shares, the shipment's payer first", () => {
    splitCase(
      "TS summarizeBillingByPayer: freight and accessorial shares merge per payer, the shipment's payer first",
    );
  });

  it("keeps the positions of empty rows so each share names its row", () => {
    splitCase("null allocation entries are skipped and positions are kept");
  });

  it("reads half-typed form values as not filled in instead of refusing the request", () => {
    const split = splitAllocations(rulesSync(), {
      defaultPayerId: "cus_a",
      charges: [
        {
          kind: "Freight",
          total: Number.NaN,
          allocations: [
            { billToCustomerId: "cus_a", method: "Percent", percent: "abc", sequence: 0 },
            { billToCustomerId: "cus_b", method: "Percent", percent: 100, sequence: 1.7 },
          ],
        },
      ],
    });
    expect(split.charges).toEqual([
      { total: "0", error: { message: "Percent allocations for this charge must total 100" } },
    ]);
    expect(split.payers).toBeNull();
  });
});

describe("allocationRemainders", () => {
  it("measures amount rows against the charge in cents", () => {
    chargesCase("remainder: amount rows are measured against the charge in cents", remainderOutput);
  });

  it("flags mixed methods, duplicate payers and blank payers in that order", () => {
    chargesCase(
      "remainder: mixed methods, duplicate payers and blank payers are flagged in that order",
      remainderOutput,
    );
  });

  it("accepts no rows and a full 100%, and refuses 99.99% and a zero row", () => {
    chargesCase(
      "remainder (TS allocationsValid): empty is valid, 99.99 is not, 100 is, a zero row is not",
      remainderOutput,
    );
  });
});

describe("chargeTotals", () => {
  it("prices flat, per-unit and percentage lines, a per-unit line with no units at zero", () => {
    chargesCase(
      "chargeTotal: flat, per unit and percentage lines (Go TestAdditionalChargeTotal)",
      chargeTotals,
    );
  });

  it("prices decimal strings exactly", () => {
    chargesCase("chargeTotal: TS chargeLineTotal cases", chargeTotals);
  });
});
