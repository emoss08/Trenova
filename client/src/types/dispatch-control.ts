import { z } from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const serviceIncidentTypeSchema = z.enum([
  "Never",
  "Pickup",
  "Delivery",
  "PickupDelivery",
  "AllExceptShipper",
]);

export type ServiceIncidentType = z.infer<typeof serviceIncidentTypeSchema>;

export const autoAssignmentStrategySchema = z.enum([
  "Proximity",
  "Availability",
  "LoadBalancing",
]);

export type AutoAssignmentStrategy = z.infer<
  typeof autoAssignmentStrategySchema
>;

export const complianceEnforcementLevelSchema = z.enum([
  "Warning",
  "Block",
  "Audit",
]);

export type ComplianceEnforcementLevel = z.infer<
  typeof complianceEnforcementLevelSchema
>;

export const dispatchControlSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,

    enableAutoAssignment: z.boolean(),
    autoAssignmentStrategy: autoAssignmentStrategySchema,
    enforceWorkerAssign: z.boolean(),
    enforceTrailerContinuity: z.boolean(),
    enforceHosCompliance: z.boolean(),
    enforceWorkerPtaRestrictions: z.boolean(),
    enforceWorkerTractorFleetContinuity: z.boolean(),
    enforceDriverQualificationCompliance: z.boolean(),
    enforceMedicalCertCompliance: z.boolean(),
    enforceHazmatCompliance: z.boolean(),
    enforceDrugAndAlcoholCompliance: z.boolean(),
    complianceEnforcementLevel: complianceEnforcementLevelSchema,
    recordServiceFailures: serviceIncidentTypeSchema,
    serviceFailureTarget: z
      .number()
      .nonnegative("Service failure target must be non-negative")
      .nullish(),
    serviceFailureGracePeriod: z.number().int().nullish(),
  })
  .refine(
    (data) => {
      if (
        data.recordServiceFailures !== serviceIncidentTypeSchema.enum.Never &&
        (!data.serviceFailureGracePeriod || data.serviceFailureGracePeriod <= 0)
      ) {
        return false;
      }
      return true;
    },
    {
      path: ["serviceFailureGracePeriod"],
      message:
        "Service failure grace period must be greater than 0 when record service failures is enabled",
    },
  );

export type DispatchControl = z.infer<typeof dispatchControlSchema>;
