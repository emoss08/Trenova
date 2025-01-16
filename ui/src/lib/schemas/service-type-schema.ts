import { Status } from "@/types/common";
import { type InferType, mixed, object, string } from "yup";

export const serviceTypeSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  status: mixed<Status>()
    .required("Status is required")
    .oneOf(Object.values(Status)),
  code: string().required("Code is required"),
  description: string().optional(),
  color: string().optional(),
});

export type ServiceTypeSchema = InferType<typeof serviceTypeSchema>;
