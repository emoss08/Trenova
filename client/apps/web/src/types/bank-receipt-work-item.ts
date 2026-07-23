import { z } from "zod";
import { nullableStringSchema, optionalStringSchema, tenantInfoSchema } from "./helpers";

export const workItemStatusSchema = z.enum([
  "Open",
  "Assigned",
  "InReview",
  "Resolved",
  "Dismissed",
]);
export type WorkItemStatus = z.infer<typeof workItemStatusSchema>;

export const resolutionTypeSchema = z.enum([
  "MatchedToPayment",
  "MarkedFalsePositive",
  "RequiresExternalFollowUp",
  "Superseded",
]);
export type ResolutionType = z.infer<typeof resolutionTypeSchema>;

export const bankReceiptWorkItemSchema = z.object({
  ...tenantInfoSchema.shape,
  bankReceiptId: z.string(),
  status: workItemStatusSchema,
  assignedToUserId: nullableStringSchema,
  assignedAt: z.number().int().nullish(),
  resolutionType: resolutionTypeSchema.nullish(),
  resolutionNote: optionalStringSchema,
  resolvedByUserId: nullableStringSchema,
  resolvedAt: z.number().int().nullish(),
  createdById: optionalStringSchema,
  updatedById: optionalStringSchema,
});
export type BankReceiptWorkItem = z.infer<typeof bankReceiptWorkItemSchema>;
