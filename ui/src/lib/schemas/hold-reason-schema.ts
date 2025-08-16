import * as z from "zod/v4";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { HoldSeverity, HoldType } from "./shipment-hold-schema";

export const holdReasonSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  active: z.boolean().default(true),
  type: HoldType,
  code: z
    .string()
    .min(1, {
      error: "Code is required",
    })
    .max(64, {
      error: "Code must be less than 64 characters",
    }),
  label: z
    .string()
    .min(1, {
      error: "Label is required",
    })
    .max(100, {
      error: "Label must be less than 100 characters",
    }),
  description: optionalStringSchema,
  defaultSeverity: HoldSeverity,
  defaultBlocksDispatch: z.boolean().default(false),
  defaultBlocksDelivery: z.boolean().default(false),
  defaultBlocksBilling: z.boolean().default(false),
  defaultVisibleToCustomer: z.boolean().default(false),
  sortOrder: z.number().int().min(0).default(100),
  externalMap: z.record(z.string(), z.any()).nullish(),
});

export type HoldReasonSchema = z.infer<typeof holdReasonSchema>;
