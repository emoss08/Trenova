import { z } from "zod";
import { optionalStringSchema } from "./helpers";

export const arOpenItemSchema = z.object({
  invoiceId: z.string(),
  customerId: z.string(),
  customerName: z.string(),
  invoiceNumber: z.string(),
  billType: z.string(),
  invoiceDate: z.number().int(),
  dueDate: z.number().int(),
  currency: z.string(),
  shipmentProNumber: optionalStringSchema,
  shipmentBolNumber: optionalStringSchema,
  totalAmountMinor: z.number().int(),
  appliedAmountMinor: z.number().int(),
  openAmountMinor: z.number().int(),
  daysPastDue: z.number().int(),
});
export type AROpenItem = z.infer<typeof arOpenItemSchema>;
