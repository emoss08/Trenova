import { describe, expect, it } from "vitest";
import { additionalChargeSchema } from "@trenova/shared/types/shipment";

/**
 * `additional_charges.unit` is a smallint (`AdditionalCharge.Unit` an int16), so a charge
 * bills at most 32,767 units. `amount` is numeric(19, 4), and the server caps a rate at
 * 99,999,999,999.9999 (`shipment.MaxAdditionalChargeRate`), the most that column holds
 * which a JavaScript number still carries exactly with four decimals.
 */
const charge = {
  accessorialChargeId: "acc_det",
  method: "PerUnit",
  amount: "75",
  unit: 1,
};

function issues(input: Record<string, unknown>) {
  const result = additionalChargeSchema.safeParse({ ...charge, ...input });
  return result.success ? [] : result.error.issues.map((issue) => [issue.path, issue.message]);
}

describe("additionalChargeSchema unit", () => {
  it("accepts every quantity a smallint holds, from one to 32,767", () => {
    expect(additionalChargeSchema.parse({ ...charge, unit: 1 }).unit).toBe(1);
    expect(additionalChargeSchema.parse({ ...charge, unit: 1500 }).unit).toBe(1500);
    expect(additionalChargeSchema.parse({ ...charge, unit: 32_767 }).unit).toBe(32_767);
  });

  it("refuses a quantity past 32,767 on the unit field", () => {
    expect(issues({ unit: 32_768 })).toEqual([[["unit"], "Unit must be at most 32,767"]]);
  });
});

describe("additionalChargeSchema amount", () => {
  it("accepts a rate well past $100,000, up to the server's cap", () => {
    expect(additionalChargeSchema.parse({ ...charge, amount: "250000.00" }).amount).toBe(250_000);
    expect(additionalChargeSchema.parse({ ...charge, amount: "99999999999.9999" }).amount).toBe(
      99_999_999_999.9999,
    );
  });

  it("refuses a rate past 99,999,999,999.9999 on the amount field", () => {
    expect(issues({ amount: "100000000000" })).toEqual([
      [["amount"], "Amount must be at most 99,999,999,999.9999"],
    ]);
  });
});
