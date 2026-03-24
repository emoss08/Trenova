import { z } from "zod";

export const analyticsPageSchema = z.enum(["shipment-management", "api-key-management"]);

export type AnalyticsPage = z.infer<typeof analyticsPageSchema>;

export type AnalyticsData = Record<string, any>;

export const analyticsParamsSchema = z.object({
  page: analyticsPageSchema,
  startDate: z.number().optional(),
  endDate: z.number().optional(),
  limit: z.number().optional(),
  timezone: z.string().optional(),
});

export type AnalyticsParams = z.infer<typeof analyticsParamsSchema>;
