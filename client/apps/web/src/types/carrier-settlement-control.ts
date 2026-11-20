import { translate } from "@trenova/shared/i18n/runtime";
import {
  payPeriodFrequencySchema,
  settlementPayTriggerSchema,
} from "@trenova/shared/types/driver-pay";
import { z } from "zod";

/**
 * An organization's carrier settlement settings, as the `carrierSettlementControl` query sends
 * them and the form edits them. The variance tolerance stays in minor units (cents), as the
 * server keeps it, so an amount is saved exactly as it was typed.
 */
export const carrierSettlementControlSchema = z.object({
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
    .number({ error: () => translate("Enter how many days after a period carriers are paid") })
    .int()
    .min(0, { error: () => translate("Pay delay cannot be negative") })
    .max(30, { error: () => translate("Pay delay cannot exceed 30 days") }),
  autoGenerateBatches: z.boolean(),
  payTrigger: settlementPayTriggerSchema,

  varianceToleranceMinor: z
    .number({ error: () => translate("Enter the variance tolerance") })
    .int()
    .min(0, { error: () => translate("Variance tolerance cannot be negative") }),
  autoMatchInboundInvoices: z.boolean(),
  autoAcceptWithinTolerance: z.boolean(),
  holdUntilInvoiceMatched: z.boolean(),

  requireSeparateApprover: z.boolean(),
  autoPostOnApprove: z.boolean(),
  defaultApAccountId: z.string().nullable(),
  defaultPurchasedTransportationAccountId: z.string().nullable(),
});

export type CarrierSettlementControl = z.infer<typeof carrierSettlementControlSchema>;
