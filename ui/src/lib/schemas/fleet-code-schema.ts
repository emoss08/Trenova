import { Status } from "@/types/common";
import { type InferType, mixed, number, object, string } from "yup";

export const fleetCodeSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  status: mixed<Status>()
    .required("Status is required")
    .oneOf(Object.values(Status)),
  name: string().required("Name is required"),
  description: string().optional(),
  revenueGoal: number()
    .transform((value) => (Number.isNaN(value) ? undefined : value))
    .nullable()
    .optional(),
  deadheadGoal: number()
    .transform((value) => (Number.isNaN(value) ? undefined : value))
    .nullable()
    .optional(),
  color: string().optional(),
  managerId: string().nullable(),
});

export type FleetCodeSchema = InferType<typeof fleetCodeSchema>;
