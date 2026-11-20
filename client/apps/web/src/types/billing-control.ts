import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "@trenova/shared/types/helpers";

const decimalNumberSchema = z.coerce.number().finite();

/** The most reminders one invoice can draw, as the server allows. */
export const MAX_PAYMENT_REMINDER_STEPS = 6;

/** The latest day past due a reminder can be scheduled for. */
export const MAX_PAYMENT_REMINDER_DAY = 365;

/** The latest day statements can go out on, so the day falls in every month. */
export const MAX_STATEMENT_DAY = 28;

export const transferScheduleSchema = z.enum(["Continuous", "Hourly", "Daily", "Weekly"]);
export type TransferSchedule = z.infer<typeof transferScheduleSchema>;

export const paymentTermSchema = z.enum([
  "Net10",
  "Net15",
  "Net30",
  "Net45",
  "Net60",
  "Net90",
  "DueOnReceipt",
]);
export type PaymentTerm = z.infer<typeof paymentTermSchema>;

export const queueAutoApproveModeSchema = z.enum(["Off", "WhenChecksClear", "WhenAgentAgrees"]);
export type QueueAutoApproveMode = z.infer<typeof queueAutoApproveModeSchema>;

export const enforcementLevelSchema = z.enum(["Ignore", "Warn", "RequireReview", "Block"]);
export type EnforcementLevel = z.infer<typeof enforcementLevelSchema>;

export const billingExceptionDispositionSchema = z.enum([
  "RouteToBillingReview",
  "ReturnToOperations",
]);
export type BillingExceptionDisposition = z.infer<typeof billingExceptionDispositionSchema>;

export const readyToBillAssignmentModeSchema = z.enum(["ManualOnly", "AutomaticWhenEligible"]);
export type ReadyToBillAssignmentMode = z.infer<typeof readyToBillAssignmentModeSchema>;

export const billingQueueTransferModeSchema = z.enum(["ManualOnly", "AutomaticWhenReady"]);
export type BillingQueueTransferMode = z.infer<typeof billingQueueTransferModeSchema>;

export const invoiceDraftCreationModeSchema = z.enum(["ManualOnly", "AutomaticWhenTransferred"]);
export type InvoiceDraftCreationMode = z.infer<typeof invoiceDraftCreationModeSchema>;

export const invoicePostingModeSchema = z.enum([
  "ManualReviewRequired",
  "AutomaticWhenNoBlockingExceptions",
]);
export type InvoicePostingMode = z.infer<typeof invoicePostingModeSchema>;

export const lateChargeAssessmentModeSchema = z.enum(["Disabled", "Preview", "Automatic"]);
export type LateChargeAssessmentMode = z.infer<typeof lateChargeAssessmentModeSchema>;

export const rateVarianceAutoResolutionModeSchema = z.enum([
  "Disabled",
  "BypassReviewWithinTolerance",
]);
export type RateVarianceAutoResolutionMode = z.infer<typeof rateVarianceAutoResolutionModeSchema>;

export const unratedShipmentDispositionSchema = z.enum([
  "FallbackFormulaTemplate",
  "ZeroAndFlag",
  "Block",
]);
export type UnratedShipmentDisposition = z.infer<typeof unratedShipmentDispositionSchema>;

export const billingControlSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,

    defaultPaymentTerm: paymentTermSchema,
    defaultInvoiceTerms: z.string().optional().default(""),
    defaultInvoiceFooter: z.string().optional().default(""),
    showDueDateOnInvoice: z.boolean(),
    showBalanceDueOnInvoice: z.boolean(),

    readyToBillAssignmentMode: readyToBillAssignmentModeSchema,
    billingQueueTransferMode: billingQueueTransferModeSchema,
    // The server stores no schedule as an empty string, and a cleared select holds one too.
    billingQueueTransferSchedule: z.preprocess(
      (value) => (value === "" ? null : value),
      transferScheduleSchema.nullish(),
    ),
    billingQueueTransferBatchSize: z.coerce.number().int().nullish(),

    invoiceDraftCreationMode: invoiceDraftCreationModeSchema,
    invoicePostingMode: invoicePostingModeSchema,
    autoInvoiceBatchSize: z.coerce.number().int().nullish(),
    notifyOnAutoInvoiceCreation: z.boolean(),

    shipmentBillingRequirementEnforcement: enforcementLevelSchema,
    rateValidationEnforcement: enforcementLevelSchema,
    billingExceptionDisposition: billingExceptionDispositionSchema,
    notifyOnBillingExceptions: z.boolean(),

    rateVarianceTolerancePercent: decimalNumberSchema.min(0, {
      error: () => translate("The tolerance cannot be negative"),
    }),
    rateVarianceAutoResolutionMode: rateVarianceAutoResolutionModeSchema,

    unratedShipmentDisposition: unratedShipmentDispositionSchema,
    fallbackFormulaTemplateId: z.string().nullish(),
    requireRateOverrideReason: z.boolean(),
    enforceMarginFloor: z.boolean(),

    queueAutoApproveMode: queueAutoApproveModeSchema.default("Off"),
    queueAutoApproveMaxAmount: z
      .preprocess(
        (value) => (value === null || value === undefined || value === "" ? null : Number(value)),
        z.number().positive().nullable(),
      )
      .default(null),
    queueAgentReviewEnabled: z.boolean().default(true),

    lateChargeAssessmentMode: lateChargeAssessmentModeSchema.default("Disabled"),
    lateChargeMinimumAmount: decimalNumberSchema
      .min(0, { error: () => translate("The minimum late charge cannot be negative") })
      .default(0),

    paymentRemindersEnabled: z.boolean().default(false),
    paymentReminderDays: z
      .array(z.number().int())
      .nullish()
      .transform((value) => value ?? [1, 15, 30]),

    monthlyStatementsEnabled: z.boolean().default(false),
    monthlyStatementDay: z
      .number({ error: () => translate("Choose the day of the month to send statements on") })
      .int()
      .min(1, { error: () => translate("Choose a day from 1 to 28") })
      .max(MAX_STATEMENT_DAY, { error: () => translate("Choose a day from 1 to 28") })
      .default(1),
  })
  .superRefine((data, ctx) => {
    const refuse = (path: keyof typeof data, message: string) =>
      ctx.addIssue({ code: "custom", path: [path], message });

    if (data.billingQueueTransferMode === "AutomaticWhenReady") {
      if (!data.billingQueueTransferSchedule) {
        refuse(
          "billingQueueTransferSchedule",
          translate("Choose how often shipments move to the queue"),
        );
      }
      if ((data.billingQueueTransferBatchSize ?? 0) < 1) {
        refuse("billingQueueTransferBatchSize", translate("Move at least one shipment per batch"));
      }
    }

    if (data.invoiceDraftCreationMode === "AutomaticWhenTransferred") {
      if ((data.autoInvoiceBatchSize ?? 0) < 1) {
        refuse("autoInvoiceBatchSize", translate("Create at least one draft per batch"));
      }
    } else if (data.invoicePostingMode === "AutomaticWhenNoBlockingExceptions") {
      refuse(
        "invoicePostingMode",
        translate("Invoices can post on their own only when drafts are created on their own"),
      );
    }

    // Skipping review is offered only while rate validation requires review; under any other
    // level the mode is sent as Disabled (`prepareBillingControlForSubmit`).
    if (
      data.rateValidationEnforcement === "RequireReview" &&
      data.rateVarianceAutoResolutionMode === "BypassReviewWithinTolerance" &&
      !(data.rateVarianceTolerancePercent > 0)
    ) {
      refuse(
        "rateVarianceTolerancePercent",
        translate("Set a tolerance above zero to skip review within it"),
      );
    }

    if (data.paymentRemindersEnabled) {
      const days = data.paymentReminderDays;
      if (days.length === 0) {
        refuse(
          "paymentReminderDays",
          translate("Choose at least one day past due to send a reminder on"),
        );
      } else if (days.length > MAX_PAYMENT_REMINDER_STEPS) {
        refuse("paymentReminderDays", translate("Send at most six reminders per invoice"));
      } else if (days.some((day) => day < 1 || day > MAX_PAYMENT_REMINDER_DAY)) {
        refuse(
          "paymentReminderDays",
          translate("Reminder days must be between 1 and 365 days past due"),
        );
      } else if (days.some((day, index) => index > 0 && day <= days[index - 1])) {
        refuse(
          "paymentReminderDays",
          translate("List reminder days from earliest to latest, each once"),
        );
      }
    }
  });

export type BillingControl = z.infer<typeof billingControlSchema>;
