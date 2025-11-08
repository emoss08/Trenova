import { Status } from "@/types/common";
import * as z from "zod";
import { accountTypeSchema } from "./account-type-schema";
import {
  nullableStringSchema,
  optionalStringSchema,
  pulidSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const glAccountSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  status: z.enum(Status, {
    error: "Status is required",
  }),
  accountTypeId: pulidSchema,
  parentId: nullableStringSchema,
  accountCode: z.string().min(1, { error: "Account code is required" }),
  name: z.string().min(1, { error: "Name is required" }),
  description: optionalStringSchema,
  isActive: z.boolean().default(true),
  isSystem: z.boolean(),
  allowManualJE: z.boolean().default(true),
  requireProject: z.boolean().default(false),
  currentBalance: z.number().nonnegative().default(0),
  debitBalance: z.number().nonnegative().default(0),
  creditBalance: z.number().nonnegative().default(0),

  accountType: accountTypeSchema.nullish(),
  get parent(): z.ZodNullable<typeof glAccountSchema> {
    return z.nullable(glAccountSchema);
  },
  get children(): z.ZodNullable<z.ZodArray<typeof glAccountSchema>> {
    return z.nullable(z.array(glAccountSchema));
  },
});

export type GLAccountSchema = z.infer<typeof glAccountSchema>;
