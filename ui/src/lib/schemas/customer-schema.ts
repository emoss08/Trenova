import { BillingCycleType, PaymentTerm } from "@/types/billing";
import { Status } from "@/types/common";
import * as z from "zod";
import { documentTypeSchema } from "./document-type-schema";
import { geocodeBadeSchema } from "./geocode-schema";
import { glAccountSchema } from "./gl-account-schema";
import {
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { usStateSchema } from "./us-state-schema";

export const emailProfileSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  customerId: nullableStringSchema,
  subject: optionalStringSchema,
  comment: optionalStringSchema,
  fromEmail: optionalStringSchema,
  blindCopy: optionalStringSchema,
  readReceipt: z.boolean(),
  attachmentName: optionalStringSchema,
});

export const billingProfileSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  customerId: nullableStringSchema,
  billingCycleType: z.enum(BillingCycleType),
  hasOverrides: z.boolean(),
  revenueAccountId: nullableStringSchema,
  arAccountId: nullableStringSchema,
  allowInvoiceConsolidation: z.boolean().default(false),
  consolidationPeriodDays: z.number().default(7),
  enforceCustomerBillingReq: z.boolean().default(false),
  validateCustomerRates: z.boolean().default(false),
  paymentTerm: z.enum(PaymentTerm).default(PaymentTerm.Net30),
  autoTransfer: z.boolean().default(false),
  autoMarkReadyToBill: z.boolean().default(false),
  autoBill: z.boolean().default(false),
  specialInstructions: optionalStringSchema,
  documentTypes: z.array(documentTypeSchema),

  revenueAccount: glAccountSchema.nullish(),
  arAccount: glAccountSchema.nullish(),
});

export const customerSchema = z.object({
  ...geocodeBadeSchema.shape,
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  status: z.enum(Status),
  name: z.string().min(1, { error: "Name is required" }),
  code: z.string().min(1, { error: "Code is required" }),
  description: optionalStringSchema,
  addressLine1: z.string().min(1, { error: "Address line 1 is required" }),
  addressLine2: optionalStringSchema,
  city: z.string().min(1, { error: "City is required" }),
  postalCode: z.string().min(1, { error: "Postal code is required" }),
  stateId: z.string().min(1, { error: "State is required" }),
  consolidationPriority: z.number().default(1),
  allowConsolidation: z.boolean().default(true),
  exclusiveConsolidation: z.boolean().default(false),
  externalId: optionalStringSchema,
  state: usStateSchema.optional(),
  billingProfile: billingProfileSchema.nullish(),
  emailProfile: emailProfileSchema.nullish(),
});

export type CustomerSchema = z.infer<typeof customerSchema>;
export type CustomerBillingProfileSchema = z.infer<typeof billingProfileSchema>;
