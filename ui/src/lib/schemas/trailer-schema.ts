/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import * as z from "zod/v4";
import { equipmentManufacturerSchema } from "./equipment-manufacturer-schema";
import { equipmentTypeSchema } from "./equipment-type-schema";
import { fleetCodeSchema } from "./fleet-code-schema";
import {
  nullableIntegerSchema,
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";
import { EquipmentStatus } from "./tractor-schema";
import { usStateSchema } from "./us-state-schema";

export const trailerSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,

  // * Core Fields
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
  status: z.enum(EquipmentStatus, {
    error: "Status is required",
  }),
  model: z.string().max(50, {
    error: "Model must be less than 50 characters",
  }),
  make: z.string().max(50, {
    error: "Make must be less than 50 characters",
  }),
  year: nullableIntegerSchema,
  licensePlateNumber: optionalStringSchema,
  vin: optionalStringSchema,
  registrationNumber: optionalStringSchema,
  maxLoadWeight: nullableIntegerSchema,
  lastInspectionDate: nullableIntegerSchema,
  registrationExpiry: nullableIntegerSchema,

  // * Relationships
  equipmentType: equipmentTypeSchema.nullish(),
  equipmentManufacturer: equipmentManufacturerSchema.nullish(),
  fleetCode: fleetCodeSchema.nullish(),
  registrationState: usStateSchema.nullish(),
});

export type TrailerSchema = z.infer<typeof trailerSchema>;
