import { EquipmentStatus } from "@/types/tractor";
import { type InferType, mixed, number, object, string } from "yup";

export const tractorSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  equipmentTypeId: string().required("Equipment Type is required"),
  primaryWorkerId: string().required("Primary Worker is required"),
  secondaryWorkerId: string().nullable().optional(),
  equipmentManufacturerId: string().nullable().optional(),
  stateId: string().nullable().optional(),
  fleetCodeId: string().nullable().optional(),
  status: mixed<EquipmentStatus>()
    .required("Status is required")
    .oneOf(Object.values(EquipmentStatus)),
  code: string().required("Code is required"),
  model: string().optional(),
  make: string().optional(),
  year: number().nullable().optional(),
  licensePlateNumber: string().optional(),
  vin: string().optional(),
});

export type TractorSchema = InferType<typeof tractorSchema>;
