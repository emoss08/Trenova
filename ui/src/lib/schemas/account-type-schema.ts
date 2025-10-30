import { Status } from "@/types/common";
import * as z from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const AccountTypeCategorySchema = z.enum([
  "Asset",
  "Liability",
  "Equity",
  "Revenue",
  "CostOfRevenue",
  "Expense",
]);

export const accountTypeSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  status: z.enum(Status),
  code: z
    .string({
      error: "Code must be a string",
    })
    .min(1, { error: "Code is required" })
    .max(10, { error: "Code must be less than 10 characters" })
    .regex(/^[A-Za-z0-9]+$/, {
      error: "Code must contain only letters and numbers",
    }),
  name: z
    .string({
      error: "Name must be a string",
    })
    .min(1, { error: "Name is required" })
    .max(100, { error: "Name must be less than 100 characters" }),
  description: z.string().optional(),
  category: AccountTypeCategorySchema,
  color: z.string().optional(),
  isSystem: z.boolean().default(false),
});

export type AccountTypeSchema = z.infer<typeof accountTypeSchema>;
