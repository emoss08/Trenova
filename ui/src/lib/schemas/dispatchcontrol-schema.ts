import * as z from "zod/v4";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const ServiceIncidentType = z.enum([
  "Never",
  "Pickup",
  "Delivery",
  "PickupDelivery",
  "AllExceptShipper",
]);

export const AutoAssignmentStrategy = z.enum([
  "Proximity",
  "Availability",
  "LoadBalancing",
]);

export const dispatchControlSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,
    enableAutoAssignment: z.boolean(),
    autoAssignmentStrategy: AutoAssignmentStrategy,
    serviceFailureGracePeriod: z
      .number()
      .int("Service failure grace period must be an whole number")
      .nonnegative("Service failure grace period must be non-negative"),
    enforceHosCompliance: z.boolean(),
    enforceDriverQualificationCompliance: z.boolean(),
    enforceMedicalCertCompliance: z.boolean(),
    enforceHazmatCompliance: z.boolean(),
    enforceDrugAndAlcoholCompliance: z.boolean(),
    complianceEnforcementLevel: z.string(),
    recordServiceFailures: ServiceIncidentType,
    serviceFailureTarget: z
      .number()
      .nonnegative("Service failure target must be non-negative")
      .nullish(),
  })
  .refine(
    (data) => {
      if (data.enableAutoAssignment && !data.autoAssignmentStrategy) {
        return false;
      }
      return true;
    },
    {
      message:
        "Auto assignment strategy is required when auto assignment is enabled",
      path: ["autoAssignmentStrategy"],
    },
  )
  .refine(
    (data) => {
      if (
        data.recordServiceFailures !== ServiceIncidentType.enum.Never &&
        data.serviceFailureGracePeriod <= 0
      ) {
        return false;
      }
      return true;
    },
    {
      message:
        "Service failure grace period must be greater than 0 when record service failures is enabled",
      path: ["serviceFailureGracePeriod"],
    },
  );

export type DispatchControlSchema = z.infer<typeof dispatchControlSchema>;
