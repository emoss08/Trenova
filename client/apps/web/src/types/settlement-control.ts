import { translate } from "@trenova/shared/i18n/runtime";
import { compareDecimalStrings, decimalString } from "@trenova/shared/types/decimal";
import {
  payPeriodFrequencySchema,
  settlementPayTriggerSchema,
} from "@trenova/shared/types/driver-pay";
import { z } from "zod";

/** The server keeps both percentages as NUMERIC(7, 4). */
export const SETTLEMENT_PERCENT_SCALE = 4;

const percentSchema = (label: string, max: string, missing: () => string) =>
  z.string({ error: missing }).pipe(
    decimalString(
      SETTLEMENT_PERCENT_SCALE,
      translate("{0} must be a number with at most 4 decimals", translate(label)),
    )
      .refine((value) => compareDecimalStrings(value, "0") >= 0, {
        error: () => translate("{0} cannot be negative", translate(label)),
      })
      .refine((value) => compareDecimalStrings(value, max) <= 0, {
        error: () => translate("{0} cannot exceed {1}%", translate(label), max),
      }),
  );

/**
 * An organization's driver settlement settings, as the `settlementControl` query sends them
 * and the form edits them. Percentages stay decimal strings, as the server sends them, so a
 * rate is saved exactly as it was typed.
 */
export const settlementControlSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  version: z.number().int().min(0),

  payPeriodFrequency: payPeriodFrequencySchema,
  periodEndDayOfWeek: z
    .number({ error: () => translate("Choose the day each pay period ends") })
    .int()
    .min(0)
    .max(6),
  payDelayDays: z
    .number({ error: () => translate("Enter how many days after a period drivers are paid") })
    .int()
    .min(0, { error: () => translate("Pay delay cannot be negative") })
    .max(30, { error: () => translate("Pay delay cannot exceed 30 days") }),
  payTrigger: settlementPayTriggerSchema,
  autoGenerateBatches: z.boolean(),
  autoAttachAccruals: z.boolean(),

  varianceThresholdPct: percentSchema("Net pay change", "999.9999", () =>
    translate("Enter the change in net pay that flags a settlement"),
  ),
  varianceLookbackWeeks: z
    .number({ error: () => translate("Enter how many settlements make up the average") })
    .int()
    .min(1, { error: () => translate("Average over at least 1 settlement") })
    .max(52, { error: () => translate("Average over at most 52 settlements") }),
  autoApproveClean: z.boolean(),
  requireSeparateApprover: z.boolean(),
  autoPostOnApprove: z.boolean(),

  allowNegativeNet: z.boolean(),
  defaultEscrowInterestRate: percentSchema("Interest rate", "100", () =>
    translate("Enter the interest rate new escrow accounts start with"),
  ),
  escrowInterestFrequencyMonths: z
    .number({ error: () => translate("Choose how often escrow interest is paid") })
    .int()
    .min(1, { error: () => translate("Escrow interest must be paid at least every quarter") })
    .max(3, { error: () => translate("Escrow interest must be paid at least every quarter") }),
});

export type SettlementControl = z.infer<typeof settlementControlSchema>;

/** Whether the net pay check runs: the server skips it when the threshold is zero. */
export function checksNetPayVariance(threshold: string | null | undefined): boolean {
  if (threshold == null || threshold.trim() === "") return true;
  try {
    return compareDecimalStrings(threshold, "0") !== 0;
  } catch {
    return true;
  }
}
