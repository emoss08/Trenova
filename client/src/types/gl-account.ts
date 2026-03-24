import { z } from "zod";
import {
  optionalStringSchema,
  statusSchema,
  tenantInfoSchema,
} from "./helpers";

export const glAccountSchema = z.object({
  ...tenantInfoSchema.shape,
  status: statusSchema,
  accountTypeId: z.string(),
  accountCode: z.string(),
  name: z.string(),
  description: optionalStringSchema,
});

export type GLAccount = z.infer<typeof glAccountSchema>;
