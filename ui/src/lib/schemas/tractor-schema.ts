import { EquipmentStatus } from "@/types/tractor";
import * as z from "zod/v4";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

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
    secondaryWorkerId: z.string().nullable().optional(),
    equipmentManufacturerId: z
      .string()
      .min(1, { error: "Equipment Manufacturer is required" }),
    stateId: z.string().nullable().optional(),
    fleetCodeId: z.string().min(1, { error: "Fleet Code is required" }),
    status: z.enum(EquipmentStatus, {
      message: "Status is required",
    }),
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
    registrationNumber: z.string().optional(),
    year: z.preprocess(
      (val) => {
        if (val === "" || val === null || val === undefined) {
          return undefined;
        }
        const parsed = parseInt(String(val), 10);
        return isNaN(parsed) ? undefined : parsed;
      },
      z
        .number()
        .min(1900, {
          error: "Year must be between 1900 and 2099",
        })
        .max(2099, {
          error: "Year must be between 1900 and 2099",
        })
        .optional(),
    ),
    licensePlateNumber: z.string().optional(),
    vin: z.string().optional(),
    registrationExpiry: z.number().nullable().optional(),
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
