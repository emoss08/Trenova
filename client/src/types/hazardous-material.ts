import { z } from "zod";
import {
  optionalStringSchema,
  statusSchema,
  tenantInfoSchema,
} from "./helpers";

export const hazardousClassSchema = z.enum([
  "HazardClass1",
  "HazardClass1And1",
  "HazardClass1And2",
  "HazardClass1And3",
  "HazardClass1And4",
  "HazardClass1And5",
  "HazardClass1And6",
  "HazardClass2And1",
  "HazardClass2And2",
  "HazardClass2And3",
  "HazardClass3",
  "HazardClass4And1",
  "HazardClass4And2",
  "HazardClass4And3",
  "HazardClass5And1",
  "HazardClass5And2",
  "HazardClass6And1",
  "HazardClass6And2",
  "HazardClass7",
  "HazardClass8",
  "HazardClass9",
]);

export type HazardousClass = z.infer<typeof hazardousClassSchema>;

export const packingGroupSchema = z.enum(["I", "II", "III"]);

export type PackingGroup = z.infer<typeof packingGroupSchema>;

export const hazardousMaterialSchema = z.object({
  ...tenantInfoSchema.shape,
  status: statusSchema,
  code: optionalStringSchema,
  name: z.string().min(1, { error: "Name is required" }),
  description: optionalStringSchema,
  class: hazardousClassSchema,
  packingGroup: packingGroupSchema.optional(),
  unNumber: optionalStringSchema,
  subsidiaryHazardClass: optionalStringSchema,
  ergGuideNumber: optionalStringSchema,
  labelCodes: optionalStringSchema,
  specialProvisions: optionalStringSchema,
  properShippingName: optionalStringSchema,
  handlingInstructions: optionalStringSchema,
  emergencyContact: optionalStringSchema,
  emergencyContactPhoneNumber: optionalStringSchema,
  quantityThreshold: optionalStringSchema,
  placardRequired: z.boolean().default(false),
  isReportableQuantity: z.boolean().default(false),
  marinePollutant: z.boolean().default(false),
  inhalationHazard: z.boolean().default(false),
});

export type HazardousMaterial = z.infer<typeof hazardousMaterialSchema>;

export const bulkUpdateHazardousMaterialStatusRequestSchema = z.object({
  hazardousMaterialIds: z.array(z.string()),
  status: statusSchema,
});

export type BulkUpdateHazardousMaterialStatusRequest = z.infer<
  typeof bulkUpdateHazardousMaterialStatusRequestSchema
>;

export const bulkUpdateHazardousMaterialStatusResponseSchema = z.array(
  hazardousMaterialSchema,
);

export type BulkUpdateHazardousMaterialStatusResponse = z.infer<
  typeof bulkUpdateHazardousMaterialStatusResponseSchema
>;
