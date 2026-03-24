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

export const trailerSchema = z.object({
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
  fleetCodeId: nullableStringSchema,
  registrationStateId: nullableStringSchema,
  model: z.string().min(1, { error: "Model is required" }).max(50, {
    error: "Model must be less than 50 characters",
  }),
  make: z.string().min(1, { error: "Make is required" }).max(50, {
    error: "Make must be less than 50 characters",
  }),
  year: z
    .number({
      error: "Year must be a valid integer",
    })
    .positive()
    .int()
    .min(1, {
      error: "Year must be between 1900 and 2099",
    })
    .max(2099, {
      error: "Year must be between 1900 and 2099",
    }),
  licensePlateNumber: optionalStringSchema,
  vin: optionalStringSchema,
  registrationNumber: optionalStringSchema,
  maxLoadWeight: nullableIntegerSchema,
  lastInspectionDate: nullableIntegerSchema,
  registrationExpiry: nullableIntegerSchema,
  lastKnownLocationId: nullableStringSchema,
  lastKnownLocationName: optionalStringSchema,

  equipmentType: equipmentTypeSchema.nullish(),
  equipmentManufacturer: equipmentManufacturerSchema.nullish(),
  fleetCode: fleetCodeSchema.nullish(),
  registrationState: usStateSchema.nullish(),
  customFields: z.record(z.string(), z.any()).optional(),
});

export type Trailer = z.infer<typeof trailerSchema>;

export const bulkUpdateTrailerStatusRequestSchema = z.object({
  trailerIds: z.array(z.string()),
  status: equipmentStatusSchema,
});

export type BulkUpdateTrailerStatusRequest = z.infer<
  typeof bulkUpdateTrailerStatusRequestSchema
>;

export const bulkUpdateTrailerStatusResponseSchema = z.array(trailerSchema);

export type BulkUpdateTrailerStatusResponse = z.infer<
  typeof bulkUpdateTrailerStatusResponseSchema
>;

export const locateTrailerPayloadSchema = z.object({
  newLocationId: z.string().min(1, { message: "Location is required" }),
});

export type LocateTrailerPayload = z.infer<typeof locateTrailerPayloadSchema>;
