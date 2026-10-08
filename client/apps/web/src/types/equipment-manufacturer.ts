import { z } from "zod";
import {
  optionalStringSchema,
  statusSchema,
  tenantInfoSchema,
} from "@trenova/shared/types/helpers";
import { translate } from "@trenova/shared/i18n/runtime";

export const equipmentManufacturerSchema = z.object({
  ...tenantInfoSchema.shape,
  status: statusSchema,
  name: z
    .string()
    .min(1, { error: () => translate("Name is required") })
    .max(100),
  description: optionalStringSchema,
});

export type EquipmentManufacturer = z.infer<typeof equipmentManufacturerSchema>;

export const bulkUpdateEquipmentManufacturerStatusRequestSchema = z.object({
  equipmentManufacturerIds: z.array(z.string()),
  status: statusSchema,
});

export type BulkUpdateEquipmentManufacturerStatusRequest = z.infer<
  typeof bulkUpdateEquipmentManufacturerStatusRequestSchema
>;

export const bulkUpdateEquipmentManufacturerStatusResponseSchema = z.array(
  equipmentManufacturerSchema,
);

export type BulkUpdateEquipmentManufacturerStatusResponse = z.infer<
  typeof bulkUpdateEquipmentManufacturerStatusResponseSchema
>;
