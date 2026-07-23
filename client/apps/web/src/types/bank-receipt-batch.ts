import { z } from "zod";
import { nullableStringSchema, optionalStringSchema, tenantInfoSchema } from "./helpers";
import { bankReceiptSchema } from "./bank-receipt";

export const bankReceiptBatchStatusSchema = z.enum(["Processing", "Completed"]);
export type BankReceiptBatchStatus = z.infer<typeof bankReceiptBatchStatusSchema>;

export const bankReceiptBatchSchema = z.object({
  ...tenantInfoSchema.shape,
  source: z.string(),
  reference: z.string(),
  status: bankReceiptBatchStatusSchema,
  importedCount: z.number().int(),
  matchedCount: z.number().int(),
  exceptionCount: z.number().int(),
  importedAmountMinor: z.number().int(),
  matchedAmountMinor: z.number().int(),
  exceptionAmountMinor: z.number().int(),
  createdById: optionalStringSchema,
  updatedById: nullableStringSchema,
});
export type BankReceiptBatch = z.infer<typeof bankReceiptBatchSchema>;

export const batchSourceOptionSchema = z.object({
  value: z.string(),
  label: z.string(),
});
export type BatchSourceOption = z.infer<typeof batchSourceOptionSchema>;

export const importBatchLineSchema = z.object({
  receiptDate: z.number().int(),
  amountMinor: z.number().int(),
  referenceNumber: z.string(),
  memo: z.string().optional(),
});
export type ImportBatchLine = z.infer<typeof importBatchLineSchema>;

export const createBatchRequestSchema = z.object({
  source: z.string().min(1),
  reference: z.string().min(1),
  receipts: z.array(importBatchLineSchema).min(1),
});
export type CreateBatchRequest = z.infer<typeof createBatchRequestSchema>;

export const batchDetailResponseSchema = z.object({
  batch: bankReceiptBatchSchema,
  receipts: z.array(bankReceiptSchema),
});
export type BatchDetailResponse = z.infer<typeof batchDetailResponseSchema>;
