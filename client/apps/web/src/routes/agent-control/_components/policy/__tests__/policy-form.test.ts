import { describe, expect, it } from "vitest";
import type { AgentControl } from "@/lib/graphql/agent-control";
import {
  policyFormSchema,
  promotesOnSave,
  toControlInput,
  toPolicyForm,
  type PolicyFormValues,
} from "../policy-form";

const loaded: PolicyFormValues = {
  earnedAutonomy: false,
  promotionThreshold: 10,
  learnsFromWork: true,
  personMonthlyMessages: 0,
  agentMoney: {
    callCeilingCents: 100_000,
    runCalls: 10,
    runAmountCents: 500_000,
    dayCalls: 50,
    dayAmountCents: 2_500_000,
  },
  aiTrainingConsent: false,
  version: 4,
};

describe("toControlInput", () => {
  it("keeps the pause switch and leaves consent out unless it changed", () => {
    const input = toControlInput({ ...loaded, learnsFromWork: false }, loaded, true);

    expect(input).toEqual({
      shadowMode: true,
      earnedAutonomy: false,
      promotionThreshold: 10,
      learningOff: true,
      personMonthlyMessages: 0,
      agentMoneyCallCeiling: "1000.00",
      agentMoneyRunCalls: 10,
      agentMoneyRunAmount: "5000.00",
      agentMoneyDayCalls: 50,
      agentMoneyDayAmount: "25000.00",
      version: 4,
    });
  });

  it("sends consent when it changed", () => {
    expect(toControlInput({ ...loaded, aiTrainingConsent: true }, loaded, false)).toMatchObject({
      aiTrainingConsent: true,
    });
  });
});

describe("promotesOnSave", () => {
  it("is true when earned autonomy turns on or its threshold drops", () => {
    expect(promotesOnSave({ ...loaded, earnedAutonomy: true }, loaded)).toBe(true);
    const on = { ...loaded, earnedAutonomy: true };
    expect(promotesOnSave({ ...on, promotionThreshold: 5 }, on)).toBe(true);
    expect(promotesOnSave({ ...on, promotionThreshold: 25 }, on)).toBe(false);
    expect(promotesOnSave(loaded, loaded)).toBe(false);
  });
});

describe("money limits", () => {
  it("reads the Decimal limits as cents and sends them back as Decimals", () => {
    const form = toPolicyForm({
      earnedAutonomy: false,
      promotionThreshold: 10,
      learningOff: false,
      personMonthlyMessages: 0,
      agentMoneyCallCeiling: "1250.50",
      agentMoneyRunCalls: 3,
      agentMoneyRunAmount: "2500.00",
      agentMoneyDayCalls: 20,
      agentMoneyDayAmount: "10000.00",
      aiTrainingConsent: false,
      version: 2,
    } as AgentControl);

    expect(form.agentMoney).toEqual({
      callCeilingCents: 125_050,
      runCalls: 3,
      runAmountCents: 250_000,
      dayCalls: 20,
      dayAmountCents: 1_000_000,
    });
    expect(toControlInput(form, form, false)).toMatchObject({
      agentMoneyCallCeiling: "1250.50",
      agentMoneyRunCalls: 3,
      agentMoneyRunAmount: "2500.00",
      agentMoneyDayCalls: 20,
      agentMoneyDayAmount: "10000.00",
    });
  });

  it("accepts zero everywhere, which holds every money change for a person", () => {
    const zero = {
      ...loaded,
      agentMoney: {
        callCeilingCents: 0,
        runCalls: 0,
        runAmountCents: 0,
        dayCalls: 0,
        dayAmountCents: 0,
      },
    };

    expect(policyFormSchema.safeParse(zero).success).toBe(true);
  });

  it("refuses limits that do not nest inside each other", () => {
    const issues = (agentMoney: PolicyFormValues["agentMoney"]) =>
      (policyFormSchema.safeParse({ ...loaded, agentMoney }).error?.issues ?? []).map((issue) =>
        issue.path.join("."),
      );

    expect(issues({ ...loaded.agentMoney, callCeilingCents: 500_001 })).toEqual([
      "agentMoney.callCeilingCents",
    ]);
    expect(issues({ ...loaded.agentMoney, runAmountCents: 2_500_001 })).toEqual([
      "agentMoney.runAmountCents",
    ]);
    expect(issues({ ...loaded.agentMoney, runCalls: 51 })).toEqual(["agentMoney.runCalls"]);
    expect(issues({ ...loaded.agentMoney, dayCalls: -1 })).toContain("agentMoney.dayCalls");
  });

  it("says what is wrong with a limit in words, never in cents", () => {
    const messages = (agentMoney: Record<string, unknown>) =>
      Object.fromEntries(
        (
          policyFormSchema.safeParse({
            ...loaded,
            agentMoney: { ...loaded.agentMoney, ...agentMoney },
          }).error?.issues ?? []
        ).map((issue) => [issue.path.join("."), issue.message]),
      );

    expect(messages({ dayAmountCents: 100_000_000_001 })).toEqual({
      "agentMoney.dayAmountCents": "An amount can be at most 1,000,000,000",
    });
    expect(messages({ dayCalls: 10_001 })).toEqual({
      "agentMoney.dayCalls": "At most 10,000 changes",
    });
    expect(messages({ callCeilingCents: Number.NaN })).toEqual({
      "agentMoney.callCeilingCents": "Enter an amount",
    });
    expect(messages({ runCalls: undefined })).toEqual({
      "agentMoney.runCalls": "Enter a number of changes",
    });
    expect(messages({ runCalls: 2.5 })).toEqual({
      "agentMoney.runCalls": "Enter a whole number of changes",
    });
    expect(messages({ runAmountCents: -1 })["agentMoney.runAmountCents"]).toBe(
      "An amount cannot be below 0",
    );
  });
});
