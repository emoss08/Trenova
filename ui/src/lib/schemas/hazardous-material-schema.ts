import { Status } from "@/types/common";
import {
  HazardousClassChoiceProps,
  PackingGroupChoiceProps,
} from "@/types/hazardous-material";
import { boolean, type InferType, mixed, object, string } from "yup";

export const hazardousMaterialSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  status: mixed<Status>()
    .required("Status is required")
    .oneOf(Object.values(Status)),
  code: string().required("Code is required"),
  name: string().required("Name is required"),
  description: string().required("Description is required"),
  class: mixed<HazardousClassChoiceProps>()
    .required("Class is required")
    .oneOf(Object.values(HazardousClassChoiceProps)),
  unNumber: string().optional(),
  ergNumber: string().optional(),
  packingGroup: mixed<PackingGroupChoiceProps>()
    .required("Packing Group is required")
    .oneOf(Object.values(PackingGroupChoiceProps)),
  properShippingName: string().optional(),
  handlingInstructions: string().optional(),
  emergencyContact: string().optional(),
  emergencyContactPhoneNumber: string().optional(),
  placardRequired: boolean().optional(),
  isReportableQuantity: boolean().optional(),
});

export type HazardousMaterialSchema = InferType<typeof hazardousMaterialSchema>;
