import { z } from "zod";

export const analyticsPageSchema = z.enum(["shipment-management", "api-key-management"]);

export type AnalyticsPage = z.infer<typeof analyticsPageSchema>;
