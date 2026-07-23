import { z } from "zod";
import {
  decimalStringSchema,
  optionalStringSchema,
  statusSchema,
  tenantInfoSchema,
} from "./helpers";

export const equipmentClassSchema = z.enum([
  "Tractor",
  "Trailer",
  "Container",
  "Other",
]);

export type EquipmentClass = z.infer<typeof equipmentClassSchema>;

export const equipmentTypeSchema = z.object({
  ...tenantInfoSchema.shape,
  status: statusSchema,
  code: z.string().min(1, { error: "Code is required" }),
  description: optionalStringSchema,
  class: equipmentClassSchema,
  color: optionalStringSchema,
  interiorLength: decimalStringSchema,
});

export type EquipmentType = z.infer<typeof equipmentTypeSchema>;

export const bulkUpdateEquipmentTypeStatusRequestSchema = z.object({
  equipmentTypeIds: z.array(z.string()),
  status: statusSchema,
});

export type BulkUpdateEquipmentTypeStatusRequest = z.infer<
  typeof bulkUpdateEquipmentTypeStatusRequestSchema
>;

export const bulkUpdateEquipmentTypeStatusResponseSchema =
  z.array(equipmentTypeSchema);

export type BulkUpdateEquipmentTypeStatusResponse = z.infer<
  typeof bulkUpdateEquipmentTypeStatusResponseSchema
>;
