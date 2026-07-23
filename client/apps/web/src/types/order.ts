import { z } from "zod";
import {
  decimalStringSchema,
  optionalStringSchema,
  tenantInfoSchema,
  versionSchema,
} from "./helpers";

export const orderStatusSchema = z.enum([
  "Draft",
  "Confirmed",
  "InProgress",
  "Completed",
  "Billed",
  "Closed",
  "Canceled",
]);

export type OrderStatus = z.infer<typeof orderStatusSchema>;

export const orderSchema = z.object({
  ...tenantInfoSchema.shape,
  customerId: z.string().min(1, { error: "Customer is required" }),
  ownerId: optionalStringSchema,
  status: orderStatusSchema,
  orderNumber: optionalStringSchema,
  poNumber: optionalStringSchema,
  bol: optionalStringSchema,
  currencyCode: z.string().min(1, { error: "Currency code is required" }),
  quotedAmount: decimalStringSchema,
  baseAmount: decimalStringSchema,
  totalAmount: decimalStringSchema,
  version: versionSchema,
  customer: z
    .object({
      id: z.string(),
      name: z.string(),
      code: z.string(),
    })
    .nullish(),
});

export type Order = z.infer<typeof orderSchema>;

// Raw form values (before zod transforms) — amounts may still be strings here.
export type OrderFormValues = z.input<typeof orderSchema>;

export const orderChargeFormSchema = z.object({
  description: z.string().trim().min(1, { error: "Description is required" }),
  amount: z
    .number({ error: "Amount is required" })
    .positive({ error: "Amount must be greater than zero" }),
});

export type OrderChargeFormValues = z.infer<typeof orderChargeFormSchema>;
