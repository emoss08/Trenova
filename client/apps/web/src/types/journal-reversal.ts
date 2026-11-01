import { z } from "zod";

export const journalReversalStatusSchema = z.enum([
  "Requested",
  "PendingApproval",
  "Approved",
  "Rejected",
  "Cancelled",
  "Posted",
]);
export type JournalReversalStatus = z.infer<typeof journalReversalStatusSchema>;
