import { z } from "zod";
import { equipmentManufacturerSchema } from "./equipment-manufacturer";
import { equipmentTypeSchema } from "./equipment-type";
import { fleetCodeSchema } from "./fleet-code";
import {
  equipmentStatusSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  optionalStringSchema,
  tenantInfoSchema,
} from "./helpers";
import { usStateSchema } from "./us-state";
import { workerSchema } from "./worker";

export const tractorSchema = z.object({
  ...tenantInfoSchema.shape,
  status: equipmentStatusSchema,
  code: z.string().min(1, {
    message: "Code is required",
  }),
  equipmentTypeId: z.string().min(1, {
    message: "Equipment Type is required",
  }),
  equipmentManufacturerId: z.string().min(1, {
    message: "Equipment Manufacturer is required",
  }),
  primaryWorkerId: z.string().min(1, {
    message: "Primary Worker is required",
  }),
  secondaryWorkerId: nullableStringSchema,
  fleetCodeId: nullableStringSchema,
  stateId: nullableStringSchema,
  model: optionalStringSchema,
  make: optionalStringSchema,
  year: nullableIntegerSchema,
  licensePlateNumber: optionalStringSchema,
  vin: optionalStringSchema,
  registrationNumber: optionalStringSchema,
  registrationExpiry: nullableIntegerSchema,

  equipmentType: equipmentTypeSchema.nullish(),
  equipmentManufacturer: equipmentManufacturerSchema.nullish(),
  fleetCode: fleetCodeSchema.nullish(),
  state: usStateSchema.nullish(),
  primaryWorker: workerSchema.nullish(),
  secondaryWorker: workerSchema.nullish(),
  customFields: z.record(z.string(), z.any()).optional(),
});

export type Tractor = z.infer<typeof tractorSchema>;

export const bulkUpdateTractorStatusRequestSchema = z.object({
  tractorIds: z.array(z.string()),
  status: equipmentStatusSchema,
});

export type BulkUpdateTractorStatusRequest = z.infer<
  typeof bulkUpdateTractorStatusRequestSchema
>;

export const bulkUpdateTractorStatusResponseSchema = z.array(tractorSchema);

export type BulkUpdateTractorStatusResponse = z.infer<
  typeof bulkUpdateTractorStatusResponseSchema
>;
