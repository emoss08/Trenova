import { BillingCycleType, PaymentTerm } from "@/types/billing";
import { Status } from "@/types/common";
import { z } from "zod";

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

  // * Core Fields
  customerId: z.string().nullable().optional(),
  billingCycleType: z.nativeEnum(BillingCycleType),
  documentTypeIds: z.array(z.string()).min(1, "Document Type IDs are required"),
  hasOverrides: z.boolean(),
  enforceCustomerBillingReq: z.boolean(),
  validateCustomerRates: z.boolean(),
  paymentTerm: z.nativeEnum(PaymentTerm),
  autoTransfer: z.boolean(),
  autoMarkReadyToBill: z.boolean(),
  autoBill: z.boolean(),
  specialInstructions: z.string().optional(),
});

export const customerSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),

  // * Core Fields
  status: z.nativeEnum(Status),
  name: z.string().min(1, "Name is required"),
  code: z.string().min(1, "Code is required"),
  description: z.string().optional(),
  addressLine1: z.string().min(1, "Address line 1 is required"),
  addressLine2: z.string().optional(),
  city: z.string().min(1, "City is required"),
  postalCode: z.string().min(1, "Postal code is required"),
  stateId: z.string().min(1, "State is required"),
  billingProfile: billingProfileSchema.optional(),
  emailProfile: emailProfileSchema.optional(),
});

export type CustomerSchema = z.infer<typeof customerSchema>;
export type CustomerBillingProfileSchema = z.infer<typeof billingProfileSchema>;
