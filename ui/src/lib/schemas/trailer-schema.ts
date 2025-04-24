import { EquipmentStatus } from "@/types/tractor";
import { z } from "zod";

export const trailerSchema = z.object({
  id: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

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
  status: z.nativeEnum(EquipmentStatus, {
    required_error: "Status is required",
  }),
  model: z.string().max(50, {
    message: "Model must be less than 50 characters",
  }),
  make: z.string().max(50, {
    message: "Make must be less than 50 characters",
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
        message: "Year must be between 1900 and 2099",
      })
      .max(2099, {
        message: "Year must be between 1900 and 2099",
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
        message: "Max Load Weight must be greater than 0",
      })
      .optional(),
  ),
  lastInspectionDate: z.number().nullable().optional(),
  registrationExpiry: z.number().nullable().optional(),
});

export type TrailerSchema = z.infer<typeof trailerSchema>;
