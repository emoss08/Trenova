import { Status } from "@/types/common";
import { type InferType, mixed, object, string } from "yup";

export const equipmentManufacturerSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  status: mixed<Status>()
    .required("Status is required")
    .oneOf(Object.values(Status)),
  name: string().required("Name is required"),
  description: string().optional(),
});

export type EquipmentManufacturerSchema = InferType<
  typeof equipmentManufacturerSchema
>;
