import { z } from "zod";
import { agingBucketTotalsSchema } from "./ar-aging";

export const statementTransactionSchema = z.object({
  date: z.number().int(),
  documentNumber: z.string(),
  description: z.string(),
  chargeMinor: z.number().int(),
  paymentMinor: z.number().int(),
  runningBalanceMinor: z.number().int(),
});
export type StatementTransaction = z.infer<typeof statementTransactionSchema>;

export const statementOpenItemSchema = z.object({
  invoiceNumber: z.string(),
  invoiceDate: z.number().int(),
  dueDate: z.number().int(),
  totalAmountMinor: z.number().int(),
  openAmountMinor: z.number().int(),
  daysPastDue: z.number().int(),
});
export type StatementOpenItem = z.infer<typeof statementOpenItemSchema>;

export const customerStatementSchema = z.object({
  customerId: z.string(),
  customerName: z.string(),
  statementDate: z.number().int(),
  startDate: z.number().int(),
  openingBalanceMinor: z.number().int(),
  totalChargesMinor: z.number().int(),
  totalPaymentsMinor: z.number().int(),
  endingBalanceMinor: z.number().int(),
  aging: agingBucketTotalsSchema,
  transactions: z.array(statementTransactionSchema),
  openItems: z.array(statementOpenItemSchema),
});
export type CustomerStatement = z.infer<typeof customerStatementSchema>;
