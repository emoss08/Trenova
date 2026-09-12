import { z } from "zod";

export const billingQueueStatusSchema = z.enum([
  "ReadyForReview",
  "InReview",
  "Approved",
  "Posted",
  "OnHold",
  "SentBackToOps",
  "Exception",
  "Canceled",
]);
export type BillingQueueStatus = z.infer<typeof billingQueueStatusSchema>;
