import {
  allocationRemainder,
  allocationsValid,
  chargeLineTotal,
  nestChargeAllocations,
  splitCharges,
  summarizeBillingByPayer,
  toChargeAllocationInput,
} from "@trenova/shared/lib/charge-split";
import type { ChargeAllocation, Shipment } from "@trenova/shared/types/shipment";
import { describe, expect, it } from "vitest";

function row(
  overrides: Partial<ChargeAllocation> & { billToCustomerId: string },
): ChargeAllocation {
  return {
    method: "Percent",
    sequence: 0,
    ...overrides,
  } as ChargeAllocation;
}

describe("splitCharges", () => {
  it("folds the rounding remainder into the last row so shares sum to the charge", () => {
    const shares = splitCharges(
      100,
      [
        row({ billToCustomerId: "a", percent: 33.333333, sequence: 0 }),
        row({ billToCustomerId: "b", percent: 33.333333, sequence: 1 }),
        row({ billToCustomerId: "c", percent: 33.333334, sequence: 2 }),
      ],
      "a",
    );
    expect(shares.map((share) => share.amount)).toEqual([33.33, 33.33, 33.34]);
    expect(shares.reduce((sum, share) => sum + share.amountMinor, 0)).toBe(10000);
    expect(shares.every((share) => share.partial)).toBe(true);
  });

  it("orders rows by sequence so the remainder lands on the row shown last", () => {
    const shares = splitCharges(
      0.01,
      [
        row({ billToCustomerId: "last", percent: 50, sequence: 5 }),
        row({ billToCustomerId: "first", percent: 50, sequence: 1 }),
      ],
      "first",
    );
    expect(shares.map((share) => [share.billToCustomerId, share.amountMinor])).toEqual([
      ["first", 0],
      ["last", 1],
    ]);
  });

  it("passes amount rows through and derives their percent", () => {
    const shares = splitCharges(
      250,
      [
        row({ billToCustomerId: "a", method: "Amount", amount: 100 }),
        row({ billToCustomerId: "b", method: "Amount", amount: 150 }),
      ],
      "a",
    );
    expect(shares.map((share) => share.amount)).toEqual([100, 150]);
    expect(shares[0].percent).toBe(40);
    expect(shares[1].percent).toBe(60);
  });

  it("bills the whole charge to the default payer when nothing is allocated", () => {
    expect(splitCharges(42.42, [], "cus_default")).toEqual([
      {
        billToCustomerId: "cus_default",
        amountMinor: 4242,
        amount: 42.42,
        percent: 100,
        partial: false,
      },
    ]);
  });
});

describe("allocationRemainder", () => {
  it("reports what is left of 100% and when a split is over", () => {
    const under = allocationRemainder([row({ billToCustomerId: "a", percent: 60 })], 100);
    expect(under).toMatchObject({ method: "Percent", allocated: 60, remaining: 40, isOver: false });
    expect(under.isComplete).toBe(false);

    const over = allocationRemainder(
      [row({ billToCustomerId: "a", percent: 60 }), row({ billToCustomerId: "b", percent: 50 })],
      100,
    );
    expect(over.isOver).toBe(true);
    expect(over.remaining).toBe(-10);
  });

  it("treats 33.33 + 33.33 + 33.34 as complete", () => {
    const state = allocationRemainder(
      [
        row({ billToCustomerId: "a", percent: 33.33 }),
        row({ billToCustomerId: "b", percent: 33.33 }),
        row({ billToCustomerId: "c", percent: 33.34 }),
      ],
      100,
    );
    expect(state.isComplete).toBe(true);
  });

  it("measures amount rows against the charge total in cents", () => {
    const state = allocationRemainder(
      [
        row({ billToCustomerId: "a", method: "Amount", amount: 10.1 }),
        row({ billToCustomerId: "b", method: "Amount", amount: 10.2 }),
      ],
      20.3,
    );
    expect(state.isComplete).toBe(true);
    expect(state.remaining).toBe(0);
  });

  it("flags mixed methods, duplicate payers and blank payers", () => {
    const mixed = allocationRemainder(
      [
        row({ billToCustomerId: "a", percent: 50 }),
        row({ billToCustomerId: "b", method: "Amount", amount: 50 }),
      ],
      100,
    );
    expect(mixed.mixedMethods).toBe(true);
    expect(
      allocationRemainder(
        [row({ billToCustomerId: "a", percent: 50 }), row({ billToCustomerId: "a", percent: 50 })],
        100,
      ).duplicatePayers,
    ).toBe(true);
    expect(
      allocationRemainder([row({ billToCustomerId: "", percent: 100 })], 100).missingPayers,
    ).toBe(true);
  });
});

describe("allocationsValid", () => {
  it("accepts an empty split and rejects one that does not add up", () => {
    expect(allocationsValid([], 100)).toBe(true);
    expect(allocationsValid([row({ billToCustomerId: "a", percent: 99.99 })], 100)).toBe(false);
    expect(allocationsValid([row({ billToCustomerId: "a", percent: 100 })], 100)).toBe(true);
    expect(allocationsValid([row({ billToCustomerId: "a", percent: 0 })], 100)).toBe(false);
  });
});

describe("chargeLineTotal", () => {
  it("multiplies flat and per-unit charges and takes a percent of freight", () => {
    expect(chargeLineTotal({ method: "Flat", amount: 25, unit: 2 })).toBe(50);
    expect(chargeLineTotal({ method: "PerUnit", amount: "1.5", unit: 4 })).toBe(6);
    expect(chargeLineTotal({ method: "Percentage", amount: 10, unit: 1 }, 2450)).toBe(245);
  });
});

describe("summarizeBillingByPayer", () => {
  it("merges freight and accessorial shares per payer with the shipment's payer first", () => {
    const rows = summarizeBillingByPayer(
      {
        customerId: "intel",
        billToCustomerId: null,
        freightChargeAmount: 1000,
        freightAllocations: [
          row({ billToCustomerId: "amd", percent: 40, sequence: 1 }),
          row({ billToCustomerId: "intel", percent: 60, sequence: 0 }),
        ],
        additionalCharges: [
          {
            method: "Flat",
            amount: 150,
            unit: 1,
            allocations: [row({ billToCustomerId: "amd", percent: 100 })],
          },
          { method: "Percentage", amount: 10, unit: 1, allocations: [] },
        ],
      },
      (id) => (id === "amd" ? { name: "AMD", code: "AMD1" } : null),
    );

    expect(rows.map((r) => r.payerId)).toEqual(["intel", "amd"]);
    expect(rows[0]).toMatchObject({
      isPrimary: true,
      freightAmount: 600,
      accessorialAmount: 100,
      totalAmount: 700,
      isSplit: true,
    });
    expect(rows[1]).toMatchObject({
      payerName: "AMD",
      payerCode: "AMD1",
      isPrimary: false,
      freightAmount: 400,
      accessorialAmount: 150,
      totalAmount: 550,
    });
  });

  it("reports a single unsplit payer when nothing is allocated", () => {
    const rows = summarizeBillingByPayer({
      customerId: "intel",
      billToCustomerId: "reseller",
      freightChargeAmount: 100,
      additionalCharges: [],
    });
    expect(rows).toHaveLength(1);
    expect(rows[0]).toMatchObject({ payerId: "reseller", isPrimary: true, isSplit: false });
  });
});

describe("nestChargeAllocations / toChargeAllocationInput", () => {
  it("regroups flat rows under their charge and keeps a new charge's own rows", () => {
    const freight = row({ billToCustomerId: "a", chargeKind: "Freight", percent: 100 });
    const forCharge = row({
      billToCustomerId: "b",
      chargeKind: "Accessorial",
      additionalChargeId: "ac_1",
      percent: 100,
    });
    const nested = nestChargeAllocations({
      chargeAllocations: [freight, forCharge],
      additionalCharges: [
        { id: "ac_1", allocations: [] },
        { id: undefined, allocations: [row({ billToCustomerId: "c", percent: 100 })] },
      ] as NonNullable<Shipment["additionalCharges"]>,
    });
    expect(nested.freightAllocations).toEqual([freight]);
    expect(nested.additionalCharges[0].allocations).toEqual([forCharge]);
    expect(nested.additionalCharges[1].allocations).toHaveLength(1);
  });

  it("sends decimals as strings, drops the snapshot and numbers rows by position", () => {
    const input = toChargeAllocationInput(
      { billToCustomerId: "a", method: "Percent", percent: 12.5, sequence: 9, id: "" },
      3,
    );
    expect(input).toEqual({
      id: undefined,
      billToCustomerId: "a",
      method: "Percent",
      percent: "12.5",
      amount: null,
      sequence: 3,
      version: undefined,
    });
    expect("billToCustomer" in input).toBe(false);
  });
});
