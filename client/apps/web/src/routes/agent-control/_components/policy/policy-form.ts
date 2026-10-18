import { centsToDecimal, decimalToCents } from "@/lib/decimal-cents";
import type { AgentControl } from "@/lib/graphql/agent-control";
import type { AgentControlInput } from "@trenova/graphql/generated/graphql";
import { formatNumber } from "@trenova/shared/i18n/format";
import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";

/** The most an agent money limit may be, in whole units of the currency. */
const MAX_MONEY_LIMIT = 1_000_000_000;
/** The most money changes a run or a day may be allowed. */
const MAX_CHANGE_LIMIT = 10_000;

function moneyLimit() {
  return z
    .number({ error: () => translate("Enter an amount") })
    .int()
    .min(0, { error: () => translate("An amount cannot be below 0") })
    .max(MAX_MONEY_LIMIT * 100, {
      error: () => translate("An amount can be at most {0}", formatNumber(MAX_MONEY_LIMIT)),
    });
}

function changeLimit() {
  return z
    .number({ error: () => translate("Enter a number of changes") })
    .int({ error: () => translate("Enter a whole number of changes") })
    .min(0, { error: () => translate("A number of changes cannot be below 0") })
    .max(MAX_CHANGE_LIMIT, {
      error: () => translate("At most {0} changes", formatNumber(MAX_CHANGE_LIMIT)),
    });
}

/** The organization-wide settings as the policy editor holds them. */
export const policyFormSchema = z.object({
  earnedAutonomy: z.boolean(),
  promotionThreshold: z.number().int().min(1).max(1000),
  learnsFromWork: z.boolean(),
  personMonthlyMessages: z.number().int().min(0),
  agentMoney: z
    .object({
      callCeilingCents: moneyLimit(),
      runCalls: changeLimit(),
      runAmountCents: moneyLimit(),
      dayCalls: changeLimit(),
      dayAmountCents: moneyLimit(),
    })
    .superRefine((money, ctx) => {
      if (money.callCeilingCents > money.runAmountCents) {
        ctx.addIssue({
          code: "custom",
          path: ["callCeilingCents"],
          message: translate("One change cannot be more than a run's total"),
        });
      }
      if (money.runAmountCents > money.dayAmountCents) {
        ctx.addIssue({
          code: "custom",
          path: ["runAmountCents"],
          message: translate("A run's total cannot be more than the day's"),
        });
      }
      if (money.runCalls > money.dayCalls) {
        ctx.addIssue({
          code: "custom",
          path: ["runCalls"],
          message: translate("A run's number of changes cannot be more than the day's"),
        });
      }
    }),
  aiTrainingConsent: z.boolean(),
  version: z.number(),
});

export type PolicyFormValues = z.infer<typeof policyFormSchema>;

export function toPolicyForm(control: AgentControl): PolicyFormValues {
  return {
    earnedAutonomy: control.earnedAutonomy,
    promotionThreshold: control.promotionThreshold,
    learnsFromWork: !control.learningOff,
    personMonthlyMessages: control.personMonthlyMessages,
    agentMoney: {
      callCeilingCents: decimalToCents(control.agentMoneyCallCeiling),
      runCalls: control.agentMoneyRunCalls,
      runAmountCents: decimalToCents(control.agentMoneyRunAmount),
      dayCalls: control.agentMoneyDayCalls,
      dayAmountCents: decimalToCents(control.agentMoneyDayAmount),
    },
    aiTrainingConsent: control.aiTrainingConsent,
    version: control.version,
  };
}

/**
 * The update the editor sends. The pause switch is not the editor's and is
 * sent as it stands; training consent only when it changed, so saving anything
 * else never re-records who consented.
 */
export function toControlInput(
  values: PolicyFormValues,
  loaded: PolicyFormValues,
  shadowMode: boolean,
): AgentControlInput {
  return {
    shadowMode,
    earnedAutonomy: values.earnedAutonomy,
    promotionThreshold: values.promotionThreshold,
    learningOff: !values.learnsFromWork,
    personMonthlyMessages: values.personMonthlyMessages,
    agentMoneyCallCeiling: centsToDecimal(values.agentMoney.callCeilingCents),
    agentMoneyRunCalls: values.agentMoney.runCalls,
    agentMoneyRunAmount: centsToDecimal(values.agentMoney.runAmountCents),
    agentMoneyDayCalls: values.agentMoney.dayCalls,
    agentMoneyDayAmount: centsToDecimal(values.agentMoney.dayAmountCents),
    version: values.version,
    ...(values.aiTrainingConsent === loaded.aiTrainingConsent
      ? {}
      : { aiTrainingConsent: values.aiTrainingConsent }),
  };
}

/**
 * Whether saving makes tools that already earned it move up: earned autonomy
 * turned on, or its threshold lowered while on.
 */
export function promotesOnSave(values: PolicyFormValues, loaded: PolicyFormValues): boolean {
  if (!values.earnedAutonomy) {
    return false;
  }
  return !loaded.earnedAutonomy || values.promotionThreshold < loaded.promotionThreshold;
}
