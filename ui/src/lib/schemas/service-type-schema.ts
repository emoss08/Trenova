import { Status } from "@/types/common";
import * as z from "zod/v4";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const serviceTypeSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  status: z.enum(Status),
  code: z
    .string({
      error: "Code is required",
    })
    .min(1, "Code is required"),
  description: z.string().optional(),
  color: z.string().optional(),
});

export type ServiceTypeSchema = z.infer<typeof serviceTypeSchema>;
