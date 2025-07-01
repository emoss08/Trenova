import { EquipmentStatus } from "@/types/tractor";
import * as z from "zod/v4";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

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
  fleetCodeId: z.string().nullable().optional(),
  registrationStateId: z.string().nullable().optional(),
  status: z.enum(EquipmentStatus, {
    error: "Status is required",
  }),
  model: z.string().max(50, {
    error: "Model must be less than 50 characters",
  }),
  make: z.string().max(50, {
    error: "Make must be less than 50 characters",
  }),
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
  registrationNumber: z.string().optional(),
  maxLoadWeight: z.preprocess(
    (val) => {
      if (val === "" || val === null || val === undefined) {
        return undefined;
      }
      const parsed = parseInt(String(val), 10);
      return isNaN(parsed) ? undefined : parsed;
    },
    z
      .number()
      .min(0, {
        error: "Max Load Weight must be greater than 0",
      })
      .optional(),
  ),
  lastInspectionDate: z.number().nullable().optional(),
  registrationExpiry: z.number().nullable().optional(),
});

export type TrailerSchema = z.infer<typeof trailerSchema>;
