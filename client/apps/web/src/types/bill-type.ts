import { z } from "zod";

export const billTypeSchema = z.enum(["Invoice", "CreditMemo", "DebitMemo"]);

export type BillType = z.infer<typeof billTypeSchema>;

export const defaultBillTypeSchema = billTypeSchema.default("Invoice");
