import { Status } from "@/types/common";
import { EquipmentClass } from "@/types/equipment-type";
import { type InferType, mixed, object, string } from "yup";

export const equipmentTypeSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  status: mixed<Status>()
    .required("Status is required")
    .oneOf(Object.values(Status)),
  code: string().required("Code is required"),
  description: string().optional(),
  class: mixed<EquipmentClass>()
    .required("Class is required")
    .oneOf(Object.values(EquipmentClass)),
  color: string().optional(),
});

export type EquipmentTypeSchema = InferType<typeof equipmentTypeSchema>;
