import { BillingCycleType, PaymentTerm } from "@/types/billing";
import { Status } from "@/types/common";
import * as z from "zod/v4";
import { documentTypeSchema } from "./document-type-schema";
import { usStateSchema } from "./us-state-schema";

export const emailProfileSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),

  // * Core Fields
  customerId: z.string().nullable().optional(),
  subject: z.string().optional(),
  comment: z.string().optional(),
  fromEmail: z.string().optional(),
  blindCopy: z.string().optional(),
  readReceipt: z.boolean(),
  attachmentName: z.string().optional(),
});

export const billingProfileSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  customerId: z.string().nullable().optional(),
  billingCycleType: z.enum(BillingCycleType),

  // * Billing Profile Fields
  hasOverrides: z.boolean(),
  enforceCustomerBillingReq: z.boolean(),
  validateCustomerRates: z.boolean(),
  paymentTerm: z.enum(PaymentTerm),
  autoTransfer: z.boolean(),
  autoMarkReadyToBill: z.boolean(),
  autoBill: z.boolean(),
  specialInstructions: z.string().optional(),
  documentTypes: z.array(documentTypeSchema).optional(),
});

export const customerSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  status: z.enum(Status),
  name: z.string().min(1, "Name is required"),
  code: z.string().min(1, "Code is required"),
  description: z.string().optional(),
  addressLine1: z.string().min(1, "Address line 1 is required"),
  addressLine2: z.string().optional(),
  city: z.string().min(1, "City is required"),
  postalCode: z.string().min(1, "Postal code is required"),
  stateId: z.string().min(1, "State is required"),

  // * Relationships
  state: usStateSchema.optional(),
  billingProfile: billingProfileSchema.optional(),
  emailProfile: emailProfileSchema.optional(),
});

export type CustomerSchema = z.infer<typeof customerSchema>;
export type CustomerBillingProfileSchema = z.infer<typeof billingProfileSchema>;
