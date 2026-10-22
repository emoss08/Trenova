import {
  hasAnyAllocations,
  nestChargeAllocations,
  toChargeAllocationInput,
  toMinorUnits,
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

describe("hasAnyAllocations", () => {
  it("is true when freight or any charge carries a row, and false when none does", () => {
    expect(hasAnyAllocations({ freightAllocations: [], additionalCharges: [] })).toBe(false);
    expect(hasAnyAllocations({ freightAllocations: null, additionalCharges: null })).toBe(false);
    expect(hasAnyAllocations({ freightAllocations: [{}] })).toBe(true);
    expect(
      hasAnyAllocations({ additionalCharges: [{ allocations: [] }, { allocations: [{}] }] }),
    ).toBe(true);
  });
});

describe("toMinorUnits", () => {
  it("turns an amount typed with two places into whole cents", () => {
    expect(toMinorUnits(0.29)).toBe(29);
    expect(toMinorUnits(1250.5)).toBe(125050);
    expect(toMinorUnits(-12.34)).toBe(-1234);
  });
});
