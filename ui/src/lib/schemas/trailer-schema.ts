import { EquipmentStatus } from "@/types/tractor";
import { type InferType, mixed, number, object, string } from "yup";

export const trailerSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  equipmentTypeId: string().required("Equipment Type is required"),
  equipmentManufacturerId: string().nullable().optional(),
  fleetCodeId: string().nullable().optional(),
  registrationStateId: string().nullable().optional(),
  status: mixed<EquipmentStatus>()
    .required("Status is required")
    .oneOf(Object.values(EquipmentStatus)),
  code: string().required("Code is required"),
  model: string().optional(),
  make: string().optional(),
  year: number().nullable().optional(),
  licensePlateNumber: string().optional(),
  vin: string().optional(),
  registrationNumber: string().optional(),
  maxLoadWeight: number().nullable().optional(),
  lastInspectionDate: number().nullable().optional(),
  registrationExpiry: number().nullable().optional(),
});

export type TrailerSchema = InferType<typeof trailerSchema>;
