import { z } from "zod";

export const analyticsPageSchema = z.enum(["shipment-management", "api-key-management"]);

export type AnalyticsPage = z.infer<typeof analyticsPageSchema>;

export type AnalyticsData = Record<string, any>;

export const shipmentSavedViewCountsSchema = z.object({
  all: z.number(),
  transit: z.number(),
  "at-risk": z.number(),
  unassigned: z.number(),
  "delivering-today": z.number(),
});

export const shipmentSavedViewCountsAnalyticsSchema = z.object({
  page: z.literal("shipment-management"),
  savedViewCounts: shipmentSavedViewCountsSchema,
});

export type ShipmentSavedViewCountsAnalytics = z.infer<
  typeof shipmentSavedViewCountsAnalyticsSchema
>;

export const analyticsParamsSchema = z.object({
  page: analyticsPageSchema,
  startDate: z.number().optional(),
  endDate: z.number().optional(),
  limit: z.number().optional(),
  offset: z.number().optional(),
  timezone: z.string().optional(),
  windowDays: z.number().optional(),
  include: z.string().optional(),
});

export type AnalyticsParams = z.infer<typeof analyticsParamsSchema>;
