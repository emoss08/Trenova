import { EquipmentStatus } from "@/types/tractor";
import { z } from "zod";

export const tractorSchema = z
  .object({
    id: z.string().optional(),
    version: z.number().optional(),
    createdAt: z.number().optional(),
    updatedAt: z.number().optional(),

    // * Core Fields
    equipmentTypeId: z.string().min(1, "Equipment Type is required"),
    primaryWorkerId: z.string().min(1, "Primary Worker is required"),
    secondaryWorkerId: z.string().nullable().optional(),
    equipmentManufacturerId: z
      .string()
      .min(1, "Equipment Manufacturer is required"),
    stateId: z.string().nullable().optional(),
    fleetCodeId: z.string().min(1, "Fleet Code is required"),
    status: z.nativeEnum(EquipmentStatus, {
      message: "Status is required",
    }),
    code: z
      .string()
      .min(1, "Code is required")
      .max(50, "Code must be less than 50 characters"),
    model: z.string().max(50, {
      message: "Model must be less than 50 characters",
    }),
    make: z.string().max(50, {
      message: "Make must be less than 50 characters",
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
          message: "Year must be between 1900 and 2099",
        })
        .max(2099, {
          message: "Year must be between 1900 and 2099",
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
