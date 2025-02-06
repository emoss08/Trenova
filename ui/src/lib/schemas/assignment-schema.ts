import { AssignmentStatus } from "@/types/assignment";
import { type InferType, mixed, object, string } from "yup";

export const assignmentSchema = object({
  id: string().optional(),
  organizationId: string().nullable().optional(),
  businessUnitId: string().nullable().optional(),
  status: mixed<AssignmentStatus>()
    .required("Status is required")
    .oneOf(Object.values(AssignmentStatus)),
  shipmentMoveId: string().nullable().optional(),
  primaryWorkerId: string().required("Primary Worker is required"),
  secondaryWorkerId: string().nullable().optional(),
  trailerId: string().required("Trailer is required"),
  tractorId: string().required("Tractor is required"),
});

export type AssignmentSchema = InferType<typeof assignmentSchema>;
