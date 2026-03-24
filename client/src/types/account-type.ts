import { z } from "zod";
import {
  optionalStringSchema,
  statusSchema,
  tenantInfoSchema,
} from "./helpers";

const accountCategorySchema = z.enum([
  "Asset",
  "Liability",
  "Equity",
  "Revenue",
  "CostOfRevenue",
  "Expense",
]);
export type AccountCategory = z.infer<typeof accountCategorySchema>;

export const accountTypeSchema = z.object({
  ...tenantInfoSchema.shape,
  status: statusSchema,
  code: z.string().min(1, { error: "Code is required" }),
  name: z.string().min(1, { error: "Name is required" }),
  description: optionalStringSchema,
  category: accountCategorySchema,
  color: optionalStringSchema,
  isSystem: z.boolean().optional(),
});

export type AccountType = z.infer<typeof accountTypeSchema>;

export const bulkUpdateAccountTypeStatusRequestSchema = z.object({
  accountTypeIds: z.array(z.string()),
  status: statusSchema,
});

export type BulkUpdateAccountTypeStatusRequest = z.infer<
  typeof bulkUpdateAccountTypeStatusRequestSchema
>;

export const bulkUpdateAccountTypeStatusResponseSchema =
  z.array(accountTypeSchema);

export type BulkUpdateAccountTypeStatusResponse = z.infer<
  typeof bulkUpdateAccountTypeStatusResponseSchema
>;
