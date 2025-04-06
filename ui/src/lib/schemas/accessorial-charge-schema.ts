import { AccessorialChargeMethod } from "@/types/billing";
import { Status } from "@/types/common";
import { InferType, mixed, number, object, string } from "yup";

export const accessorialChargeSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  status: mixed<Status>()
    .required("Status is required")
    .oneOf(Object.values(Status)),
  code: string()
    .min(3, "Code must be at least 3 characters")
    .max(10, "Code must be less than 10 characters")
    .required("Code is required"),
  description: string().required("Description is required"),
  unit: number().required("Unit is required"),
  method: mixed<AccessorialChargeMethod>()
    .required("Method is required")
    .oneOf(Object.values(AccessorialChargeMethod)),
  amount: number()
    .min(1, "Amount must be greater than 0")
    .required("Amount is required"),
});

export type AccessorialChargeSchema = InferType<typeof accessorialChargeSchema>;
