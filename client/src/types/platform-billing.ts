import { z } from "zod";

const optionalStringSchema = z.string().optional().default("");

export const billingPlanSummarySchema = z.object({
  id: z.string(),
  key: z.string(),
  name: z.string(),
  status: z.string(),
});

export const billingSubscriptionSummarySchema = z.object({
  id: z.string(),
  planId: z.string(),
  status: z.string(),
  currentPeriodStart: z.number().int(),
  currentPeriodEnd: z.number().int(),
});

export const billingFeatureSummarySchema = z.object({
  featureKey: z.string(),
  allowed: z.boolean(),
});

export const billingUsageSummarySchema = z.object({
  meterKey: z.string(),
  unit: optionalStringSchema,
  limit: z.number().int(),
  used: z.number().int(),
  remaining: z.number().int(),
  windowStart: z.number().int(),
  windowEnd: z.number().int(),
});

export const billingSummarySchema = z.object({
  businessUnitId: optionalStringSchema,
  organizationId: optionalStringSchema,
  active: z.boolean(),
  reason: optionalStringSchema,
  plan: billingPlanSummarySchema.nullish(),
  subscription: billingSubscriptionSummarySchema.nullish(),
  features: z.array(billingFeatureSummarySchema).default([]),
  usage: z.array(billingUsageSummarySchema).default([]),
  checkedAt: z.number().int(),
});

export type BillingSummary = z.infer<typeof billingSummarySchema>;
export type BillingUsageSummary = z.infer<typeof billingUsageSummarySchema>;
