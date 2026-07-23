import { z } from "zod";
import { optionalStringSchema, tenantInfoSchema } from "./helpers";

export const holdTypeSchema = z.enum([
  "OperationalHold",
  "ComplianceHold",
  "CustomerHold",
  "FinanceHold",
]);

export type HoldType = z.infer<typeof holdTypeSchema>;

export const holdSeveritySchema = z.enum([
  "Informational",
  "Advisory",
  "Blocking",
]);

export type HoldSeverity = z.infer<typeof holdSeveritySchema>;

export const holdReasonSchema = z.object({
  ...tenantInfoSchema.shape,
  active: z.boolean().default(true),
  type: holdTypeSchema,
  code: z
    .string()
    .min(1, { message: "Code is required" })
    .max(64, { message: "Code must be less than 64 characters" }),
  label: z
    .string()
    .min(1, { message: "Label is required" })
    .max(100, { message: "Label must be less than 100 characters" }),
  description: optionalStringSchema,
  defaultSeverity: holdSeveritySchema,
  defaultBlocksDispatch: z.boolean().default(false),
  defaultBlocksDelivery: z.boolean().default(false),
  defaultBlocksBilling: z.boolean().default(false),
  defaultVisibleToCustomer: z.boolean().default(false),
  sortOrder: z.number().int().min(0).default(100),
  externalMap: z.record(z.string(), z.any()).nullish(),
});

export type HoldReason = z.infer<typeof holdReasonSchema>;
