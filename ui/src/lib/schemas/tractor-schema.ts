/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

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
import { workerSchema } from "./worker-schema";

export enum EquipmentStatus {
  Available = "Available",
  OOS = "OutOfService",
  AtMaintenance = "AtMaintenance",
  Sold = "Sold",
}

export const EquipmentStatusSchema = z.enum([
  "Available",
  "OutOfService",
  "AtMaintenance",
  "Sold",
]);

export const tractorSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,

    // * Core Fields
    equipmentTypeId: z.string().min(1, { error: "Equipment Type is required" }),
    primaryWorkerId: z.string().min(1, { error: "Primary Worker is required" }),
    secondaryWorkerId: nullableStringSchema,
    equipmentManufacturerId: z
      .string()
      .min(1, { error: "Equipment Manufacturer is required" }),
    stateId: nullableStringSchema,
    fleetCodeId: z.string().min(1, { error: "Fleet Code is required" }),
    status: z.enum(EquipmentStatus),
    code: z
      .string()
      .min(1, { error: "Code is required" })
      .max(50, { error: "Code must be less than 50 characters" }),
    model: z.string().max(50, {
      error: "Model must be less than 50 characters",
    }),
    make: z.string().max(50, {
      error: "Make must be less than 50 characters",
    }),
    registrationNumber: optionalStringSchema,
    year: nullableIntegerSchema,
    licensePlateNumber: optionalStringSchema,
    vin: optionalStringSchema,
    registrationExpiry: nullableIntegerSchema,
    primaryWorker: workerSchema.nullish(),
    secondaryWorker: workerSchema.nullish(),
    equipmentType: equipmentTypeSchema.nullish(),
    equipmentManufacturer: equipmentManufacturerSchema.nullish(),
    fleetCode: fleetCodeSchema.nullish(),
  })
  .refine(
    (data) => {
      // * if there is a primary worker and the primary worker is the same as the secondary worker
      if (
        data.primaryWorkerId &&
        data.primaryWorkerId === data.secondaryWorkerId
      ) {
        return false;
      }
      return true;
    },
    {
      message: "Secondary worker cannot be the same as the primary worker",
      path: ["secondaryWorkerId"],
    },
  );

export type TractorSchema = z.infer<typeof tractorSchema>;
