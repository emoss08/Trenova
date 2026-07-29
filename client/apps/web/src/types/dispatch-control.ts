import { z } from "zod";
import { optionalStringSchema, timestampSchema, versionSchema } from "@trenova/shared/types/helpers";

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
  "Performance",
]);

export type AutoAssignmentStrategy = z.infer<typeof autoAssignmentStrategySchema>;

export const scoringFactorSchema = z.enum([
  "deadhead",
  "hosMargin",
  "onTime",
  "trailerContinuity",
  "fleetMatch",
  "driverTypeFit",
  "homeTime",
  "loadBalance",
  "laneExperience",
  "onTimeHistory",
  "safetyHistory",
  "acceptance",
  "ptoProximity",
]);

export type ScoringFactor = z.infer<typeof scoringFactorSchema>;

export const scoringWeightsSchema = z.partialRecord(
  scoringFactorSchema,
  z.number().min(0, "Weights must be at least 0").max(10, "Weights cannot exceed 10").optional(),
);

export type ScoringWeights = z.infer<typeof scoringWeightsSchema>;

/**
 * Mirrors the server's strategy presets (dispatchcontrol.PresetWeights) so the weight
 * editor can show what an unset factor inherits.
 */
export const PRESET_SCORING_WEIGHTS: Record<
  AutoAssignmentStrategy,
  Record<ScoringFactor, number>
> = {
  Proximity: {
    deadhead: 3,
    hosMargin: 1.5,
    onTime: 2.5,
    trailerContinuity: 1,
    fleetMatch: 0.5,
    driverTypeFit: 1,
    homeTime: 0.5,
    loadBalance: 0.5,
    laneExperience: 0.5,
    onTimeHistory: 1,
    safetyHistory: 0.5,
    acceptance: 0.5,
    ptoProximity: 0.5,
  },
  Availability: {
    deadhead: 1,
    hosMargin: 3,
    onTime: 2.5,
    trailerContinuity: 1,
    fleetMatch: 0.5,
    driverTypeFit: 1,
    homeTime: 1,
    loadBalance: 0.5,
    laneExperience: 0.5,
    onTimeHistory: 1,
    safetyHistory: 1,
    acceptance: 1,
    ptoProximity: 1.5,
  },
  LoadBalancing: {
    deadhead: 1,
    hosMargin: 1,
    onTime: 2,
    trailerContinuity: 0.5,
    fleetMatch: 0.5,
    driverTypeFit: 1,
    homeTime: 2,
    loadBalance: 3,
    laneExperience: 0.5,
    onTimeHistory: 0.5,
    safetyHistory: 0.5,
    acceptance: 1,
    ptoProximity: 1.5,
  },
  Performance: {
    deadhead: 1,
    hosMargin: 1,
    onTime: 2,
    trailerContinuity: 0.5,
    fleetMatch: 0.5,
    driverTypeFit: 1,
    homeTime: 0.5,
    loadBalance: 0.5,
    laneExperience: 1.5,
    onTimeHistory: 3,
    safetyHistory: 2,
    acceptance: 2,
    ptoProximity: 1,
  },
};

export const SCORING_FACTOR_META: ReadonlyArray<{
  key: ScoringFactor;
  label: string;
  description: string;
}> = [
  {
    key: "deadhead",
    label: "Empty Miles",
    description: "Fewer empty miles from the driver's projected location to the pickup.",
  },
  {
    key: "hosMargin",
    label: "Hours Remaining",
    description: "Projected hours-of-service margin left after completing the trip.",
  },
  {
    key: "onTime",
    label: "On-Time Margin",
    description: "Slack between the projected arrival and the pickup appointment.",
  },
  {
    key: "trailerContinuity",
    label: "Trailer Continuity",
    description: "Driver keeps the trailer from their previous leg instead of swapping.",
  },
  {
    key: "fleetMatch",
    label: "Fleet Match",
    description: "Driver and power unit belong to the same fleet.",
  },
  {
    key: "driverTypeFit",
    label: "Haul Fit",
    description: "Trip length suits the driver's type (local, regional, or over-the-road).",
  },
  {
    key: "homeTime",
    label: "Home Time",
    description: "Drivers recently home score higher than those out for weeks.",
  },
  {
    key: "loadBalance",
    label: "Load Balance",
    description: "Spreads work by weekly miles, move count, and open assignments.",
  },
  {
    key: "laneExperience",
    label: "Customer Experience",
    description: "How often the driver has run freight for this customer in the last year.",
  },
  {
    key: "onTimeHistory",
    label: "On-Time History",
    description: "The driver's delivered-on-time percentage versus the organization's goal.",
  },
  {
    key: "safetyHistory",
    label: "Safety History",
    description: "Hours-of-service violations recorded in the last 90 days.",
  },
  {
    key: "acceptance",
    label: "Acceptance",
    description: "How reliably and quickly the driver accepts offered assignments.",
  },
  {
    key: "ptoProximity",
    label: "Time-Off Buffer",
    description: "Whether the run finishes comfortably before the driver's approved time off.",
  },
];

export const complianceEnforcementLevelSchema = z.enum(["Warning", "Block", "Audit"]);

export type ComplianceEnforcementLevel = z.infer<typeof complianceEnforcementLevelSchema>;

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
    scoringWeights: scoringWeightsSchema.optional(),
    autoAssignConfidenceThreshold: z.coerce
      .number()
      .min(0, "Confidence threshold must be at least 0")
      .max(1, "Confidence threshold cannot exceed 1")
      .optional(),
    autoAssignMaxDeadheadMiles: z
      .number()
      .int()
      .positive("Max deadhead miles must be greater than 0")
      .nullish(),
    autoAssignPlanningHorizonHours: z
      .number()
      .int()
      .min(1, "Planning horizon must be at least 1 hour")
      .max(336, "Planning horizon cannot exceed 336 hours")
      .optional(),
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
      if (data.serviceFailureGracePeriod != null && data.serviceFailureGracePeriod <= 0) {
        return false;
      }
      return true;
    },
    {
      path: ["serviceFailureGracePeriod"],
      message: "Service failure grace period must be greater than 0",
    },
  );

export type DispatchControl = z.infer<typeof dispatchControlSchema>;
