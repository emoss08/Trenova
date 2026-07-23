import { z } from "zod";
import { nullableStringSchema } from "./helpers";

export const customerLedgerEntrySchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  customerId: z.string(),
  sourceObjectType: z.string(),
  sourceObjectId: z.string(),
  sourceEventType: z.string(),
  relatedInvoiceId: nullableStringSchema,
  documentNumber: z.string(),
  transactionDate: z.number().int(),
  lineNumber: z.number().int(),
  amountMinor: z.number().int(),
  createdById: z.string(),
});
export type CustomerLedgerEntry = z.infer<typeof customerLedgerEntrySchema>;
