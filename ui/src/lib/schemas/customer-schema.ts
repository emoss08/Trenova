import { Status } from "@/types/common";
import { boolean, type InferType, mixed, object, string } from "yup";

export const customerSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  status: mixed<Status>()
    .required("Status is required")
    .oneOf(Object.values(Status)),
  name: string().required("Name is required"),
  code: string().required("Code is required"),
  description: string().optional(),
  addressLine1: string().required("Address line 1 is required"),
  addressLine2: string().optional(),
  city: string().required("City is required"),
  postalCode: string().required("Postal code is required"),
  stateId: string().required("State is required"),
  autoMarkReadyToBill: boolean().required(
    "Auto mark ready to bill is required",
  ),
});

export type CustomerSchema = InferType<typeof customerSchema>;
