import { z } from "zod";

export const insightCategorySchema = z.enum([
  "ServiceQuality",
  "CashFlow",
  "CostLeakage",
  "Compliance",
]);

export const insightSeveritySchema = z.enum(["Info", "Warning", "Critical"]);

export const insightStatusSchema = z.enum(["Active", "Dismissed", "Resolved", "Superseded"]);

export const insightUnitSchema = z.enum(["Count", "Currency", "Percent", "Days", "Hours", "Miles"]);

export const insightDirectionSchema = z.enum(["Neutral", "HigherIsWorse", "LowerIsWorse"]);

/**
 * One number a detector computed. The value is exact and arrives as a decimal
 * string rather than a JS number, because a currency amount that has been
 * through a float is no longer the amount.
 */
export const insightMetricSchema = z.object({
  key: z.string(),
  label: z.string(),
  value: z.string(),
  unit: insightUnitSchema,
  /** Which way is bad, so a change can be coloured without knowing the metric. */
  direction: insightDirectionSchema,
  /** The comparison this should be read against, when the detector computed one. */
  baseline: z.string().nullish(),
  /** What the baseline is — "prior 30 days", "fleet average". */
  baselineLabel: z.string().optional().default(""),
});

/**
 * A path into the application, built by the detector. The server validates that
 * it is rooted and relative before storing it, so nothing generated can end up
 * as a link a person is invited to click.
 */
export const insightLinkSchema = z.object({
  label: z.string(),
  path: z.string(),
  count: z.number().default(0),
});

export const insightSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  detectorKey: z.string(),
  category: insightCategorySchema,
  severity: insightSeveritySchema,
  status: insightStatusSchema,
  dedupeKey: z.string(),
  subject: z.string().optional().default(""),
  headline: z.string(),
  narrative: z.string().optional().default(""),
  recommendation: z.string().optional().default(""),
  /**
   * True when a model wrote the prose. False means the wording is the
   * detector's own, which is plainer but never wrong — and the card says so,
   * because a reader deciding how much to trust a sentence deserves to know
   * where it came from.
   */
  narrated: z.boolean().default(false),
  metrics: z.array(insightMetricSchema).nullish(),
  links: z.array(insightLinkSchema).nullish(),
  windowStart: z.number().default(0),
  windowEnd: z.number().default(0),
  detectedAt: z.number(),
  /** When these numbers stop being worth trusting. */
  staleAt: z.number().default(0),
  dismissedAt: z.number().nullish(),
  dismissedById: z.string().optional().default(""),
  dismissReason: z.string().optional().default(""),
  modelIdentifier: z.string().optional().default(""),
  providerId: z.string().optional().default(""),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export const insightListSchema = z.object({
  results: z.array(insightSchema),
});

/**
 * What a detector looks for, in the reader's language.
 *
 * `excludes` is the part worth reading: a reader who knows a rule ignores
 * low-volume customers can tell "no finding" apart from "no problem".
 */
export const insightExplanationSchema = z.object({
  measures: z.string().optional().default(""),
  threshold: z.string().optional().default(""),
  excludes: z.string().optional().default(""),
});

export const insightDetailSchema = z.object({
  insight: insightSchema,
  /** Earlier runs of the same finding, newest first. */
  history: z.array(insightSchema).nullish(),
  explanation: insightExplanationSchema,
});

export const insightPageSchema = z.object({
  results: z.array(insightSchema),
  total: z.number().default(0),
});

export type InsightCategory = z.infer<typeof insightCategorySchema>;
export type InsightSeverity = z.infer<typeof insightSeveritySchema>;
export type InsightUnit = z.infer<typeof insightUnitSchema>;
export type InsightDirection = z.infer<typeof insightDirectionSchema>;
export type InsightMetric = z.infer<typeof insightMetricSchema>;
export type InsightLink = z.infer<typeof insightLinkSchema>;
export type Insight = z.infer<typeof insightSchema>;
export type InsightExplanation = z.infer<typeof insightExplanationSchema>;
export type InsightDetail = z.infer<typeof insightDetailSchema>;
