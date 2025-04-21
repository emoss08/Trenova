import { EquipmentStatus } from "@/types/tractor";
import { z } from "zod";

export const trailerSchema = z.object({
  id: z.string().optional(),
  organizationId: z.string().optional(),
  businessUnitId: z.string().optional(),
  version: z.number().optional(),
  createdAt: z.number().optional(),
  updatedAt: z.number().optional(),

  // * Core Fields
  equipmentTypeId: z.string().min(1, "Equipment Type is required"),
  equipmentManufacturerId: z.string().nullable().optional(),
  fleetCodeId: z.string().nullable().optional(),
  registrationStateId: z.string().nullable().optional(),
  status: z.nativeEnum(EquipmentStatus, {
    message: "Status is required",
  }),
  code: z.string().min(1, "Code is required"),
  model: z.string().optional(),
  make: z.string().optional(),
  year: z.number().nullable().optional(),
  licensePlateNumber: z.string().optional(),
  vin: z.string().optional(),
  registrationNumber: z.string().optional(),
  maxLoadWeight: z.number().nullable().optional(),
  lastInspectionDate: z.number().nullable().optional(),
  registrationExpiry: z.number().nullable().optional(),
});

export type TrailerSchema = z.infer<typeof trailerSchema>;
