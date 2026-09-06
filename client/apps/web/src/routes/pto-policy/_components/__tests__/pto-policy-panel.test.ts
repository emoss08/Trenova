import { ptoPolicyFormSchema } from "@trenova/shared/types/pto-policy";
import { describe, expect, it } from "vitest";
import type { PTOPolicyRow } from "@/lib/graphql/pto-policy";
import { buildPtoPolicyDefaults, toPtoPolicyInput } from "../pto-policy-panel";

const row: PTOPolicyRow = {
  id: "ptop_1",
  businessUnitId: "bu_1",
  organizationId: "org_1",
  name: "Standard",
  code: "STD",
  description: null,
  status: "Active",
  isDefault: true,
  yearBasis: "CalendarYear",
  countWeekends: true,
  waitingPeriodDays: 90,
  requiresApproval: true,
  enforceBalance: true,
  allowNegative: true,
  negativeFloorDays: "-3.00",
  openAssignmentCount: 4,
  version: 2,
  createdAt: 1,
  updatedAt: 2,
  rules: [
    {
      id: "ptpr_1",
      ptoPolicyId: "ptop_1",
      ptoType: "Vacation",
      accrualMethod: "Monthly",
      accrualAmountDays: "0.83",
      maxBalanceDays: "20.00",
      carryoverCapDays: "5.00",
      carryoverExpiryDays: 90,
      tiers: [
        { minMonths: 24, accrualAmountDays: "1.25", maxBalanceDays: "25.00" },
        { minMonths: 60, accrualAmountDays: "1.67", maxBalanceDays: null },
      ],
      onTermination: "PayOut",
      sortOrder: 0,
    },
    {
      id: "ptpr_2",
      ptoPolicyId: "ptop_1",
      ptoType: "Sick",
      accrualMethod: "None",
      accrualAmountDays: "0.00",
      maxBalanceDays: null,
      carryoverCapDays: null,
      carryoverExpiryDays: 0,
      tiers: [],
      onTermination: "Forfeit",
      sortOrder: 1,
    },
  ],
};

describe("PTO policy form mapping", () => {
  it("round-trips a server row through the form and back into a mutation input", () => {
    const defaults = buildPtoPolicyDefaults(row);
    expect(defaults.negativeFloorDays).toBe("-3.00");
    expect(defaults.rules).toHaveLength(2);

    const parsed = ptoPolicyFormSchema.safeParse(defaults);
    expect(parsed.success, JSON.stringify(parsed.error?.issues)).toBe(true);

    const input = toPtoPolicyInput(defaults, row.version);
    expect(input.version).toBe(2);
    expect(input.code).toBe("STD");
    expect(input.negativeFloorDays).toBe("-3.00");
    expect(input.rules[1]).toMatchObject({
      ptoType: "Sick",
      accrualMethod: "None",
      accrualAmountDays: "0",
      maxBalanceDays: undefined,
    });
  });

  it("carries tenure tiers and the termination treatment through the form", () => {
    const defaults = buildPtoPolicyDefaults(row);
    expect(defaults.rules[0].onTermination).toBe("PayOut");
    expect(defaults.rules[0].tiers).toEqual([
      { minMonths: 24, accrualAmountDays: "1.25", maxBalanceDays: "25.00" },
      { minMonths: 60, accrualAmountDays: "1.67", maxBalanceDays: null },
    ]);
    expect(defaults.rules[1].tiers).toEqual([]);
    expect(defaults.rules[1].onTermination).toBe("Forfeit");

    const input = toPtoPolicyInput(defaults, row.version);
    expect(input.rules[0].onTermination).toBe("PayOut");
    expect(input.rules[0].tiers).toEqual([
      { minMonths: 24, accrualAmountDays: "1.25", maxBalanceDays: "25.00" },
      { minMonths: 60, accrualAmountDays: "1.67", maxBalanceDays: undefined },
    ]);
    expect(input.rules[1].tiers).toEqual([]);
  });

  it("rejects tiers that do not climb by tenure and tiers on a rule that never accrues", () => {
    const base = { ...buildPtoPolicyDefaults(row), name: "Standard", code: "STD" };

    const unordered = ptoPolicyFormSchema.safeParse({
      ...base,
      rules: [
        {
          ...base.rules[0],
          tiers: [
            { minMonths: 60, accrualAmountDays: "1.67", maxBalanceDays: null },
            { minMonths: 24, accrualAmountDays: "1.25", maxBalanceDays: null },
          ],
        },
      ],
    });
    expect(unordered.success).toBe(false);
    expect(unordered.error?.issues.map((issue) => issue.path.join("."))).toContain(
      "rules.0.tiers.1.minMonths",
    );

    const zeroMonths = ptoPolicyFormSchema.safeParse({
      ...base,
      rules: [
        {
          ...base.rules[0],
          tiers: [{ minMonths: 0, accrualAmountDays: "1", maxBalanceDays: null }],
        },
      ],
    });
    expect(zeroMonths.success).toBe(false);

    const tiersWithoutAccrual = ptoPolicyFormSchema.safeParse({
      ...base,
      rules: [
        {
          ...base.rules[1],
          tiers: [{ minMonths: 12, accrualAmountDays: "1", maxBalanceDays: null }],
        },
      ],
    });
    expect(tiersWithoutAccrual.success).toBe(false);
    expect(tiersWithoutAccrual.error?.issues[0]?.path.join(".")).toBe("rules.0.tiers");

    const perPayPeriod = ptoPolicyFormSchema.safeParse({
      ...base,
      rules: [{ ...base.rules[0], accrualMethod: "PerPayPeriod", accrualAmountDays: "0.4" }],
    });
    expect(perPayPeriod.success, JSON.stringify(perPayPeriod.error?.issues)).toBe(true);
  });

  it("zeroes the floor when negative balances are switched off", () => {
    const defaults = buildPtoPolicyDefaults(row);
    defaults.allowNegative = false;
    defaults.negativeFloorDays = null;
    expect(toPtoPolicyInput(defaults).negativeFloorDays).toBe("0");
  });

  it("rejects duplicate rule types, accruing rules without an amount, and a floor without allowNegative", () => {
    const base = { ...buildPtoPolicyDefaults(null), name: "Standard", code: "STD" };

    const duplicate = ptoPolicyFormSchema.safeParse({
      ...base,
      rules: [base.rules[0], { ...base.rules[0] }],
    });
    expect(duplicate.success).toBe(false);
    expect(duplicate.error?.issues.map((issue) => issue.path.join("."))).toContain(
      "rules.1.ptoType",
    );

    const noAmount = ptoPolicyFormSchema.safeParse({
      ...base,
      rules: [{ ...base.rules[0], accrualAmountDays: "0" }],
    });
    expect(noAmount.success).toBe(false);

    const floor = ptoPolicyFormSchema.safeParse({ ...base, negativeFloorDays: "-2" });
    expect(floor.success).toBe(false);
    expect(floor.error?.issues[0]?.path).toEqual(["negativeFloorDays"]);

    const defaultDraft = ptoPolicyFormSchema.safeParse({
      ...base,
      isDefault: true,
      status: "Draft",
    });
    expect(defaultDraft.success).toBe(false);
  });

  it("uppercases the code and starts new policies with a monthly vacation rule", () => {
    const defaults = buildPtoPolicyDefaults(null);
    expect(defaults.rules[0]).toMatchObject({ ptoType: "Vacation", accrualMethod: "Monthly" });
    defaults.code = "std-driver";
    defaults.name = "Standard";
    expect(toPtoPolicyInput(defaults).code).toBe("STD-DRIVER");
  });
});
