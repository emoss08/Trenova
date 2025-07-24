/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { BillingCycleType, PaymentTerm } from "@/types/billing";
import { Status } from "@/types/common";
import * as z from "zod/v4";
import { documentTypeSchema } from "./document-type-schema";
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

  // * Core Fields
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

  // * Core Fields
  customerId: nullableStringSchema,
  billingCycleType: z.enum(BillingCycleType),

  // * Billing Profile Fields
  hasOverrides: z.boolean(),
  enforceCustomerBillingReq: z.boolean().default(false),
  validateCustomerRates: z.boolean().default(false),
  paymentTerm: z.enum(PaymentTerm).default(PaymentTerm.Net30),
  autoTransfer: z.boolean().default(false),
  autoMarkReadyToBill: z.boolean().default(false),
  autoBill: z.boolean().default(false),
  specialInstructions: optionalStringSchema,
  documentTypes: z.array(documentTypeSchema).optional(),
});

export const customerSchema = z.object({
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

  // * Relationships
  state: usStateSchema.optional(),
  billingProfile: billingProfileSchema.optional(),
  emailProfile: emailProfileSchema.optional(),
});

export type CustomerSchema = z.infer<typeof customerSchema>;
export type CustomerBillingProfileSchema = z.infer<typeof billingProfileSchema>;
