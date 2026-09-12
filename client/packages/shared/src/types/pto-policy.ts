import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";
import { ptoTypeSchema } from "./worker";

export const ptoPolicyStatusSchema = z.enum(["Active", "Inactive", "Draft"]);
export type PTOPolicyStatus = z.infer<typeof ptoPolicyStatusSchema>;

export const ptoYearBasisSchema = z.enum(["CalendarYear", "HireAnniversary"]);
export type PTOYearBasis = z.infer<typeof ptoYearBasisSchema>;

export const ptoAccrualMethodSchema = z.enum([
  "None",
  "FixedAnnualGrant",
  "Monthly",
  "PerPayPeriod",
]);
export type PTOAccrualMethod = z.infer<typeof ptoAccrualMethodSchema>;

export const ptoTerminationActionSchema = z.enum(["Forfeit", "PayOut"]);
export type PTOTerminationAction = z.infer<typeof ptoTerminationActionSchema>;

export const ptoLedgerEntryTypeSchema = z.enum([
  "OpeningBalance",
  "Accrual",
  "Usage",
  "Reversal",
  "Adjustment",
  "Carryover",
  "Expiry",
  "Payout",
  "Forfeiture",
]);
export type PTOLedgerEntryType = z.infer<typeof ptoLedgerEntryTypeSchema>;

export const ptoLedgerActorTypeSchema = z.enum(["User", "System"]);
export type PTOLedgerActorType = z.infer<typeof ptoLedgerActorTypeSchema>;

const DECIMAL_PATTERN = /^-?\d+(\.\d{1,2})?$/;

const decimalString = (message: string) => z.string().trim().regex(DECIMAL_PATTERN, { message });

const optionalDecimalString = (message: string) =>
  z
    .string()
    .nullable()
    .transform((value) => {
      const trimmed = value?.trim() ?? "";
      return trimmed === "" ? null : trimmed;
    })
    .pipe(z.string().regex(DECIMAL_PATTERN, { message }).nullable());

export const ptoAccrualTierFormSchema = z
  .object({
    minMonths: z.number().int().min(1, { message: "Tiers start after at least one month" }),
    accrualAmountDays: decimalString("Enter days with up to two decimals"),
    maxBalanceDays: optionalDecimalString("Enter days with up to two decimals"),
  })
  .superRefine((tier, ctx) => {
    if (!(Number(tier.accrualAmountDays) > 0)) {
      ctx.addIssue({
        code: "custom",
        path: ["accrualAmountDays"],
        message: translate("A tier must accrue more than zero"),
      });
    }
    if (tier.maxBalanceDays !== null && !(Number(tier.maxBalanceDays) > 0)) {
      ctx.addIssue({
        code: "custom",
        path: ["maxBalanceDays"],
        message: translate("Maximum balance must be above zero"),
      });
    }
  });

export type PTOAccrualTierFormValues = z.infer<typeof ptoAccrualTierFormSchema>;

export const ptoPolicyRuleFormSchema = z
  .object({
    ptoType: ptoTypeSchema,
    accrualMethod: ptoAccrualMethodSchema,
    accrualAmountDays: decimalString("Enter days with up to two decimals"),
    maxBalanceDays: optionalDecimalString("Enter days with up to two decimals"),
    carryoverCapDays: optionalDecimalString("Enter days with up to two decimals"),
    carryoverExpiryDays: z.number().int().min(0, { message: "Cannot be negative" }),
    tiers: z.array(ptoAccrualTierFormSchema),
    onTermination: ptoTerminationActionSchema,
  })
  .superRefine((rule, ctx) => {
    if (rule.accrualMethod === "None" && rule.tiers.length > 0) {
      ctx.addIssue({
        code: "custom",
        path: ["tiers"],
        message: translate("Tenure tiers only apply when the rule accrues"),
      });
    }
    rule.tiers.forEach((tier, index) => {
      const previous = rule.tiers[index - 1];
      if (previous && tier.minMonths <= previous.minMonths) {
        ctx.addIssue({
          code: "custom",
          path: ["tiers", index, "minMonths"],
          message: translate("Tiers must climb by tenure"),
        });
      }
    });
    const amount = Number(rule.accrualAmountDays);
    if (rule.accrualMethod !== "None" && !(amount > 0)) {
      ctx.addIssue({
        code: "custom",
        path: ["accrualAmountDays"],
        message: translate("Accrual amount is required when a rule accrues"),
      });
    }
    if (rule.accrualMethod === "None" && amount !== 0) {
      ctx.addIssue({
        code: "custom",
        path: ["accrualAmountDays"],
        message: translate("Set the amount to 0 when the rule does not accrue"),
      });
    }
    if (rule.maxBalanceDays !== null && !(Number(rule.maxBalanceDays) > 0)) {
      ctx.addIssue({
        code: "custom",
        path: ["maxBalanceDays"],
        message: translate("Maximum balance must be above zero"),
      });
    }
    if (rule.carryoverExpiryDays > 0 && rule.carryoverCapDays === null) {
      ctx.addIssue({
        code: "custom",
        path: ["carryoverExpiryDays"],
        message: translate("Carryover expiry needs a carryover cap"),
      });
    }
  });

export type PTOPolicyRuleFormValues = z.infer<typeof ptoPolicyRuleFormSchema>;

export const ptoPolicyFormSchema = z
  .object({
    name: z.string().trim().min(1, { message: "Name is required" }).max(100),
    code: z.string().trim().min(1, { message: "Code is required" }).max(50),
    description: z.string().trim().max(1000).nullable(),
    status: ptoPolicyStatusSchema,
    isDefault: z.boolean(),
    yearBasis: ptoYearBasisSchema,
    countWeekends: z.boolean(),
    waitingPeriodDays: z.number().int().min(0, { message: "Cannot be negative" }),
    requiresApproval: z.boolean(),
    enforceBalance: z.boolean(),
    allowNegative: z.boolean(),
    negativeFloorDays: optionalDecimalString("Enter days with up to two decimals"),
    rules: z.array(ptoPolicyRuleFormSchema).min(1, { message: "Add at least one PTO type" }),
  })
  .superRefine((policy, ctx) => {
    const floor = policy.negativeFloorDays === null ? 0 : Number(policy.negativeFloorDays);
    if (policy.allowNegative && !(floor < 0)) {
      ctx.addIssue({
        code: "custom",
        path: ["negativeFloorDays"],
        message: translate("Set how far below zero a balance may go"),
      });
    }
    if (!policy.allowNegative && floor !== 0) {
      ctx.addIssue({
        code: "custom",
        path: ["negativeFloorDays"],
        message: translate("Only applies when negative balances are allowed"),
      });
    }
    if (policy.isDefault && policy.status !== "Active") {
      ctx.addIssue({
        code: "custom",
        path: ["isDefault"],
        message: translate("Only an active policy can be the default"),
      });
    }
    const seen = new Set<string>();
    policy.rules.forEach((rule, index) => {
      if (seen.has(rule.ptoType)) {
        ctx.addIssue({
          code: "custom",
          path: ["rules", index, "ptoType"],
          message: translate("Each PTO type may only have one rule"),
        });
      }
      seen.add(rule.ptoType);
    });
  });

export type PTOPolicyFormValues = z.infer<typeof ptoPolicyFormSchema>;

export const assignPtoPolicyFormSchema = z.object({
  ptoPolicyId: z.string().min(1, { message: "Policy is required" }),
  effectiveFrom: z.number().int().positive({ message: "Effective date is required" }),
  note: z.string().trim().max(255).nullable(),
  openingBalances: z.array(
    z.object({
      ptoType: ptoTypeSchema,
      days: z
        .string()
        .trim()
        .regex(/^\d+(\.\d{1,2})?$/, { message: "Enter days" }),
    }),
  ),
});

export type AssignPTOPolicyFormValues = z.infer<typeof assignPtoPolicyFormSchema>;

export const adjustPtoBalanceFormSchema = z.object({
  ptoType: ptoTypeSchema,
  amountDays: z
    .string()
    .trim()
    .regex(DECIMAL_PATTERN, { message: "Enter days with up to two decimals" })
    .refine((value) => Number(value) !== 0, { message: "Amount cannot be zero" }),
  effectiveAt: z.number().int().positive({ message: "Effective date is required" }),
  note: z
    .string()
    .trim()
    .min(3, { message: "Explain the adjustment" })
    .max(255, { message: "Note must be 255 characters or fewer" }),
});

export type AdjustPTOBalanceFormValues = z.infer<typeof adjustPtoBalanceFormSchema>;

export const PTO_ACCRUAL_METHOD_LABELS: Record<PTOAccrualMethod, string> = {
  None: "No accrual",
  FixedAnnualGrant: "Annual grant",
  Monthly: "Monthly",
  PerPayPeriod: "Per pay period",
};

export const PTO_TERMINATION_ACTION_LABELS: Record<PTOTerminationAction, string> = {
  Forfeit: "Forfeit balance",
  PayOut: "Pay out balance",
};

export const PTO_YEAR_BASIS_LABELS: Record<PTOYearBasis, string> = {
  CalendarYear: "Calendar year",
  HireAnniversary: "Hire anniversary",
};

export const PTO_LEDGER_ENTRY_LABELS: Record<PTOLedgerEntryType, string> = {
  OpeningBalance: "Opening balance",
  Accrual: "Accrual",
  Usage: "Time off taken",
  Reversal: "Reversal",
  Adjustment: "Adjustment",
  Carryover: "Carryover",
  Expiry: "Expiry",
  Payout: "Paid out",
  Forfeiture: "Forfeited",
};
