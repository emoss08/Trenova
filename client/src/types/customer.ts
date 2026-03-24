import { z } from "zod";
import {
  decimalStringSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  statusSchema,
  tenantInfoSchema,
} from "./helpers";

export const billingCycleTypeSchema = z.enum([
  "Immediate",
  "Daily",
  "Weekly",
  "BiWeekly",
  "Monthly",
  "Quarterly",
  "PerShipment",
]);
export type BillingCycleType = z.infer<typeof billingCycleTypeSchema>;

export const customerPaymentTermSchema = z.enum([
  "Net10",
  "Net15",
  "Net30",
  "Net45",
  "Net60",
  "Net90",
  "DueOnReceipt",
]);
export type CustomerPaymentTerm = z.infer<typeof customerPaymentTermSchema>;

export const creditStatusSchema = z.enum(["Active", "Warning", "Hold", "Suspended", "Review"]);
export type CreditStatus = z.infer<typeof creditStatusSchema>;

export const invoiceMethodSchema = z.enum(["Individual", "Summary", "SummaryWithDetail"]);
export type InvoiceMethod = z.infer<typeof invoiceMethodSchema>;

export const consolidationGroupBySchema = z.enum([
  "None",
  "Location",
  "PONumber",
  "BOL",
  "Division",
]);
export type ConsolidationGroupBy = z.infer<typeof consolidationGroupBySchema>;

export const invoiceNumberFormatSchema = z.enum(["Default", "CustomPrefix", "POBased"]);
export type InvoiceNumberFormat = z.infer<typeof invoiceNumberFormatSchema>;

export const customerBillingProfileSchema = z.object({
  id: z.string().optional(),
  version: z.number().int().min(0).optional(),
  createdAt: z.number().int().positive().optional(),
  updatedAt: z.number().int().positive().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  customerId: z.string().optional(),
  billingCycleType: billingCycleTypeSchema.default("Immediate"),
  billingCycleDayOfWeek: nullableIntegerSchema,
  paymentTerm: customerPaymentTermSchema.default("Net30"),
  hasBillingControlOverrides: z.boolean().default(false),
  creditLimit: decimalStringSchema,
  creditBalance: decimalStringSchema,
  creditStatus: creditStatusSchema.default("Active"),
  enforceCreditLimit: z.boolean().default(false),
  autoCreditHold: z.boolean().default(false),
  creditHoldReason: z.string().default(""),
  invoiceMethod: invoiceMethodSchema.default("Individual"),
  summaryTransmitOnGeneration: z.boolean().default(true),
  allowInvoiceConsolidation: z.boolean().default(false),
  consolidationPeriodDays: z.number().int().default(7),
  consolidationGroupBy: consolidationGroupBySchema.default("None"),
  invoiceNumberFormat: invoiceNumberFormatSchema.default("Default"),
  customerInvoicePrefix: z.string().default(""),
  invoiceCopies: z.number().int().default(1),
  revenueAccountId: nullableStringSchema,
  arAccountId: nullableStringSchema,
  applyLateCharges: z.boolean().default(false),
  lateChargeRate: decimalStringSchema,
  gracePeriodDays: z.number().int().default(0),
  taxExempt: z.boolean().default(false),
  taxExemptNumber: z.string().default(""),
  enforceCustomerBillingReq: z.boolean().default(true),
  validateCustomerRates: z.boolean().default(true),
  autoTransfer: z.boolean().default(true),
  autoMarkReadyToBill: z.boolean().default(true),
  autoBill: z.boolean().default(true),
  detentionBillingEnabled: z.boolean().default(false),
  detentionFreeMinutes: z.number().int().default(120),
  detentionRatePerHour: decimalStringSchema,
  countLateOnlyOnAppointmentStops: z.boolean().default(false),
  countDetentionOnlyOnAppointmentStops: z.boolean().default(false),
  autoApplyAccessorials: z.boolean().default(true),
  billingCurrency: z.string().max(3).default("USD"),
  requirePONumber: z.boolean().default(false),
  requireBOLNumber: z.boolean().default(false),
  requireDeliveryNumber: z.boolean().default(false),
  billingNotes: z.string().default(""),
  documentTypes: z.array(z.any()).nullish(),
});

export type CustomerBillingProfile = z.infer<typeof customerBillingProfileSchema>;

export const customerEmailProfileSchema = z.object({
  id: z.string().optional(),
  version: z.number().int().min(0).optional(),
  createdAt: z.number().int().positive().optional(),
  updatedAt: z.number().int().positive().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  customerId: z.string().optional(),
  subject: z.string().default(""),
  comment: z.string().default(""),
  fromEmail: z.string().default(""),
  toRecipients: z.string().default(""),
  ccRecipients: z.string().default(""),
  bccRecipients: z.string().default(""),
  attachmentName: z.string().default(""),
  readReceipt: z.boolean().default(false),
  sendInvoiceOnGeneration: z.boolean().default(true),
  includeShipmentDetail: z.boolean().default(false),
});

export type CustomerEmailProfile = z.infer<typeof customerEmailProfileSchema>;

export const customerSchema = z
  .object({
    ...tenantInfoSchema.shape,
    status: statusSchema,
    code: z
      .string()
      .min(1, { error: "Code is required" })
      .max(10, { error: "Code must be 10 characters or less" }),
    name: z
      .string()
      .min(1, { error: "Name is required" })
      .max(255, { error: "Name must be 255 characters or less" }),
    addressLine1: z
      .string()
      .min(1, { error: "Address line 1 is required" })
      .max(150, { error: "Address line 1 must be 150 characters or less" }),
    addressLine2: nullableStringSchema,
    city: z
      .string()
      .min(1, { error: "City is required" })
      .max(100, { error: "City must be 100 characters or less" }),
    stateId: z.string().min(1, { error: "State is required" }),
    postalCode: z.string().min(1, { error: "Postal code is required" }),
    isGeocoded: z.boolean().default(false),
    longitude: z.number().nullable().optional(),
    latitude: z.number().nullable().optional(),
    placeId: nullableStringSchema,
    externalId: nullableStringSchema,
    allowConsolidation: z.boolean().default(true),
    exclusiveConsolidation: z.boolean().default(false),
    consolidationPriority: z.number().int().min(1).default(1),
    billingProfile: customerBillingProfileSchema.optional(),
    emailProfile: customerEmailProfileSchema.optional(),
  })
  .refine(
    (data) => {
      if (data.exclusiveConsolidation && !data.allowConsolidation) {
        return false;
      }
      return true;
    },
    {
      path: ["allowConsolidation"],
      message: "Allow consolidation is required when exclusive consolidation is true",
    },
  );

export type Customer = z.infer<typeof customerSchema>;

export const bulkUpdateCustomerStatusRequestSchema = z.object({
  customerIds: z.array(z.string()),
  status: statusSchema,
});

export type BulkUpdateCustomerStatusRequest = z.infer<typeof bulkUpdateCustomerStatusRequestSchema>;

export const bulkUpdateCustomerStatusResponseSchema = z.array(customerSchema);

export type BulkUpdateCustomerStatusResponse = z.infer<
  typeof bulkUpdateCustomerStatusResponseSchema
>;
