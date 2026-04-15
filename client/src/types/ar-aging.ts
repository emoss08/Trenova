import { z } from "zod";

export const agingBucketTotalsSchema = z.object({
  currentMinor: z.number().int(),
  days1To30Minor: z.number().int(),
  days31To60Minor: z.number().int(),
  days61To90Minor: z.number().int(),
  daysOver90Minor: z.number().int(),
  totalOpenMinor: z.number().int(),
});
export type AgingBucketTotals = z.infer<typeof agingBucketTotalsSchema>;

export const customerAgingRowSchema = z.object({
  customerId: z.string(),
  customerName: z.string(),
  buckets: agingBucketTotalsSchema,
});
export type CustomerAgingRow = z.infer<typeof customerAgingRowSchema>;

export const agingSummarySchema = z.object({
  asOfDate: z.number().int(),
  totals: agingBucketTotalsSchema,
  rows: z.array(customerAgingRowSchema),
});
export type AgingSummary = z.infer<typeof agingSummarySchema>;
