import { z } from "zod";
import { equipmentManufacturerSchema } from "./equipment-manufacturer";
import { equipmentTypeSchema } from "./equipment-type";
import { fleetCodeRelationSchema } from "@trenova/shared/types/fleet-code";
import {
  customFieldsSchema,
  equipmentStatusSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  optionalStringSchema,
  relationSchema,
  tenantInfoSchema,
} from "@trenova/shared/types/helpers";
import { usStateRelationSchema } from "@trenova/shared/types/us-state";
import { iftaFuelTypeSchema } from "@trenova/shared/types/fuel-ifta-enums";
import { workerSchema } from "@trenova/shared/types/worker";

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
  externalId: optionalStringSchema.default(""),
  fuelType: iftaFuelTypeSchema.default("Diesel"),
  iftaQualified: z.boolean().default(true),

  equipmentType: relationSchema(equipmentTypeSchema),
  equipmentManufacturer: relationSchema(equipmentManufacturerSchema),
  fleetCode: fleetCodeRelationSchema,
  state: usStateRelationSchema,
  primaryWorker: relationSchema(workerSchema),
  secondaryWorker: relationSchema(workerSchema),
  customFields: customFieldsSchema,
});

export type Tractor = z.infer<typeof tractorSchema>;

export const bulkUpdateTractorStatusRequestSchema = z.object({
  tractorIds: z.array(z.string()),
  status: equipmentStatusSchema,
});

export type BulkUpdateTractorStatusRequest = z.infer<typeof bulkUpdateTractorStatusRequestSchema>;

export const bulkUpdateTractorStatusResponseSchema = z.array(tractorSchema);

export type BulkUpdateTractorStatusResponse = z.infer<typeof bulkUpdateTractorStatusResponseSchema>;
