import * as z from "zod/v4";
import {
  nullableStringSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const shipmentControlSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,

    // * Core Fields
    enableAutoAssignment: z.boolean(),
    autoAssignmentStrategy: nullableStringSchema,

    // Service Failure Related Fields
    recordServiceFailures: z.boolean().default(false),
    serviceFailureGracePeriod: z.preprocess((val) => {
      if (val === "" || val === null || val === undefined) {
        return 0;
      }
      const parsed = parseInt(String(val), 10);
      return isNaN(parsed) ? 0 : parsed;
    }, z.number().int("Service failure grace period must be an whole number").nonnegative("Service failure grace period must be non-negative")),

    // Delay Shipment Related Fields
    autoDelayShipments: z.boolean(),
    autoDelayShipmentsThreshold: z.preprocess((val) => {
      if (val === "" || val === null || val === undefined) {
        return 0;
      }
      const parsed = parseInt(String(val), 10);
      return isNaN(parsed) ? 0 : parsed;
    }, z.number().int("Auto delay shipments threshold must be an whole number").nonnegative("Auto delay shipments threshold must be non-negative")),

    // Compliance Controls
    enforceHosCompliance: z.boolean(),
    enforceDriverQualificationCompliance: z.boolean(),
    enforceMedicalCertCompliance: z.boolean(),
    enforceHazmatCompliance: z.boolean(),
    enforceDrugAndAlcoholCompliance: z.boolean(),
    complianceEnforcementLevel: z.string(),

    // Detention Tracking
    trackDetentionTime: z.boolean(),
    autoGenerateDetentionCharges: z.boolean().optional(),
    detentionThreshold: z.preprocess((val) => {
      if (val === "" || val === null || val === undefined) {
        return 0;
      }
      const parsed = parseInt(String(val), 10);
      return isNaN(parsed) ? 0 : parsed;
    }, z.number().int("Detention threshold must be an whole number").nonnegative("Detention threshold must be non-negative")),

    // Performance Metrics
    onTimeDeliveryTarget: z.preprocess((val) => {
      if (val === "" || val === null || val === undefined) {
        return 0;
      }
      const parsed = parseInt(String(val), 10);
      return isNaN(parsed) ? 0 : parsed;
    }, z.number().nonnegative("On-time delivery target must be non-negative")),
    serviceFailureTarget: z.preprocess((val) => {
      if (val === "" || val === null || val === undefined) {
        return 0;
      }
      const parsed = parseInt(String(val), 10);
      return isNaN(parsed) ? 0 : parsed;
    }, z.number().nonnegative("Service failure target must be non-negative")),
    trackCustomerRejections: z.boolean(),
    checkForDuplicateBols: z.boolean(),
    allowMoveRemovals: z.boolean(),
    checkHazmatSegregation: z.boolean(),
  })
  .refine(
    (data) => {
      // If enableAutoAssignment is true, autoAssignmentStrategy must be provided
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
      // If recordServiceFailures is true, serviceFailureGracePeriod must be greater than 0
      if (data.recordServiceFailures && data.serviceFailureGracePeriod <= 0) {
        return false;
      }
      return true;
    },
    {
      message:
        "Service failure grace period must be greater than 0 when record service failures is enabled",
      path: ["serviceFailureGracePeriod"],
    },
  )
  .refine(
    (data) => {
      // If autoDelayShipments is true, autoDelayShipmentsThreshold must be greater than 0
      if (data.autoDelayShipments && data.autoDelayShipmentsThreshold <= 0) {
        return false;
      }
      return true;
    },
    {
      message:
        "Auto delay shipments threshold must be greater than 0 when auto delay shipments is enabled",
      path: ["autoDelayShipmentsThreshold"],
    },
  )
  .refine(
    (data) => {
      // If trackDetentionTime is true, autoGenerateDetentionCharges must be provided
      if (
        data.trackDetentionTime &&
        data.autoGenerateDetentionCharges === undefined
      ) {
        return false;
      }
      return true;
    },
    {
      message:
        "Auto generate detention charges is required when track detention time is enabled",
      path: ["autoGenerateDetentionCharges"],
    },
  )
  .refine(
    (data) => {
      // If trackDetentionTime is true, detentionThreshold must be greater than 0
      if (data.trackDetentionTime && data.detentionThreshold <= 0) {
        return false;
      }
      return true;
    },
    {
      message:
        "Detention threshold must be greater than 0 when track detention time is enabled",
      path: ["detentionThreshold"],
    },
  );

export type ShipmentControlSchema = z.infer<typeof shipmentControlSchema>;
