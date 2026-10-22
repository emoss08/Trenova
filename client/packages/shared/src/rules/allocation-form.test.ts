import {
  withChargeAllocations,
  withShipmentAllocations,
} from "@trenova/shared/rules/allocation-form";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { chargeAllocationSchema, shipmentCreateSchema } from "@trenova/shared/types/shipment";
import { describe, expect, it } from "vitest";
import { z } from "zod";

function issuesAt(result: { success: boolean; error?: z.ZodError }, prefix: string) {
  return (result.error?.issues ?? [])
    .filter((issue) => issue.path.join(".").startsWith(prefix))
    .map((issue) => ({ path: issue.path.join("."), message: issue.message }));
}

describe("withShipmentAllocations", () => {
  const schema = withShipmentAllocations(shipmentCreateSchema, rulesSync());

  it("puts each charge's first problem on its allocation list while other fields still fail", () => {
    const result = schema.safeParse({
      freightChargeAmount: 1000,
      freightAllocations: [
        { billToCustomerId: "cus_a", method: "Percent", percent: 60, sequence: 0 },
        { billToCustomerId: "cus_b", method: "Percent", percent: 30, sequence: 1 },
      ],
      additionalCharges: [
        {
          accessorialChargeId: "acc_1",
          method: "Flat",
          amount: 150,
          unit: 2,
          allocations: [
            { billToCustomerId: "cus_a", method: "Amount", amount: 100, sequence: 0 },
            { billToCustomerId: "cus_b", method: "Amount", amount: 200, sequence: 1 },
          ],
        },
        {
          accessorialChargeId: "acc_2",
          method: "Flat",
          amount: 10,
          unit: 1,
          allocations: [
            { billToCustomerId: "cus_a", method: "Percent", percent: 50, sequence: 0 },
            { billToCustomerId: "cus_a", method: "Percent", percent: 50, sequence: 1 },
          ],
        },
      ],
    });

    expect([
      ...issuesAt(result, "freightAllocations"),
      ...issuesAt(result, "additionalCharges"),
    ]).toEqual([
      { path: "freightAllocations", message: "Percents must add up to 100" },
      { path: "additionalCharges.1.allocations", message: "Each payer may appear only once" },
    ]);
  });

  it("judges only percent rows on a percentage accessorial, whose total is not known here", () => {
    const result = schema.safeParse({
      freightChargeAmount: 1000,
      additionalCharges: [
        {
          accessorialChargeId: "acc_1",
          method: "Percentage",
          amount: 10,
          unit: 1,
          allocations: [
            { billToCustomerId: "cus_a", method: "Amount", amount: 40, sequence: 0 },
            { billToCustomerId: "cus_b", method: "Amount", amount: 70, sequence: 1 },
          ],
        },
      ],
    });
    expect(issuesAt(result, "additionalCharges")).toEqual([]);
  });

  it("accepts a split whose rows add up and a charge with no rows", () => {
    const result = schema.safeParse({
      freightChargeAmount: 100,
      freightAllocations: [
        { billToCustomerId: "cus_a", method: "Percent", percent: 33.33, sequence: 0 },
        { billToCustomerId: "cus_b", method: "Percent", percent: 33.33, sequence: 1 },
        { billToCustomerId: "cus_c", method: "Percent", percent: 33.34, sequence: 2 },
      ],
      additionalCharges: [
        { accessorialChargeId: "acc_1", method: "Flat", amount: 5, unit: 1, allocations: [] },
      ],
    });
    expect([
      ...issuesAt(result, "freightAllocations"),
      ...issuesAt(result, "additionalCharges"),
    ]).toEqual([]);
  });
});

describe("withChargeAllocations", () => {
  it("measures the rows against the total the form holds", () => {
    const schema = withChargeAllocations(
      z.object({ amount: z.number(), allocations: z.array(chargeAllocationSchema) }),
      rulesSync(),
      (values) => values.amount,
    );

    const result = schema.safeParse({
      amount: 0.01,
      allocations: [
        { billToCustomerId: "cus_a", method: "Amount", amount: 0.005, sequence: 0 },
        { billToCustomerId: "cus_b", method: "Amount", amount: 0.005, sequence: 1 },
      ],
    });
    expect(issuesAt(result, "allocations")).toEqual([
      { path: "allocations", message: "Amounts must add up to the charge" },
    ]);
  });
});
