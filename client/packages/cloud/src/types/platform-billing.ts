import { z } from "zod";

const optionalStringSchema = z.string().optional().default("");

const optionalTimestampSchema = z
  .number()
  .int()
  .nullish()
  .transform((value) => value ?? null);

export const billingPlanSummarySchema = z.object({
  id: optionalStringSchema,
  key: z.string(),
  name: optionalStringSchema,
  status: optionalStringSchema,
});

// A cloud organization's subscription carries the free demo's lifecycle — trialing until
// trialEndsAt, read-only until readOnlyUntil, then purged — while the platform catalog's
// paid subscriptions carry a billing period. Either set of fields may be absent.
export const SubscriptionStatus = z.enum(["trialing", "read_only", "expired", "active"]);
export type SubscriptionStatus = z.infer<typeof SubscriptionStatus>;

export const billingSubscriptionSummarySchema = z.object({
  id: optionalStringSchema,
  planId: optionalStringSchema,
  planKey: optionalStringSchema,
  status: z.string(),
  trialEndsAt: optionalTimestampSchema,
  readOnlyUntil: optionalTimestampSchema,
  currentPeriodStart: optionalTimestampSchema,
  currentPeriodEnd: optionalTimestampSchema,
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
  windowStart: z
    .number()
    .int()
    .nullish()
    .transform((value) => value ?? 0),
  windowEnd: z
    .number()
    .int()
    .nullish()
    .transform((value) => value ?? 0),
});

export const billingSummarySchema = z.object({
  businessUnitId: optionalStringSchema,
  organizationId: optionalStringSchema,
  active: z.boolean().default(true),
  reason: optionalStringSchema,
  plan: billingPlanSummarySchema.nullish(),
  subscription: billingSubscriptionSummarySchema.nullish(),
  features: z.array(billingFeatureSummarySchema).default([]),
  usage: z.array(billingUsageSummarySchema).default([]),
  restrictions: z.array(z.string()).default([]),
  checkedAt: z.number().int().default(0),
});

export type BillingSummary = z.infer<typeof billingSummarySchema>;
export type BillingUsageSummary = z.infer<typeof billingUsageSummarySchema>;
export type BillingSubscriptionSummary = z.infer<typeof billingSubscriptionSummarySchema>;
export type BillingPlanSummary = z.infer<typeof billingPlanSummarySchema>;
