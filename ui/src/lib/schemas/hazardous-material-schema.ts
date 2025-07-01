import { Status } from "@/types/common";
import {
  HazardousClassChoiceProps,
  PackingGroupChoiceProps,
} from "@/types/hazardous-material";
import * as z from "zod/v4";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const hazardousMaterialSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
  status: z.enum(Status),
  code: z.string().min(1, { error: "Code is required" }),
  name: z.string().min(1, { error: "Name is required" }),
  description: z.string().min(1, { error: "Description is required" }),
  class: z.enum(HazardousClassChoiceProps),
  unNumber: z
    .string()
    .max(4, { error: "UN Number must be 4 characters or less" })
    .optional(),
  casNumber: z
    .string()
    .max(10, { error: "CAS Number must be 10 characters or less" })
    .optional(),
  packingGroup: z.enum(PackingGroupChoiceProps),
  properShippingName: optionalStringSchema,
  handlingInstructions: optionalStringSchema,
  emergencyContact: optionalStringSchema,
  emergencyContactPhoneNumber: optionalStringSchema,
  placardRequired: z.boolean(),
  isReportableQuantity: z.boolean(),
});

export type HazardousMaterialSchema = z.infer<typeof hazardousMaterialSchema>;
