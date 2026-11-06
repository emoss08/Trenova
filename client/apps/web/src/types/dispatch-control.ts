import { z } from "zod";
import {
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "@trenova/shared/types/helpers";
import { translate } from "@trenova/shared/i18n/runtime";
import { compareDecimalStrings, decimalString } from "@trenova/shared/types/decimal";

export const serviceIncidentTypeSchema = z.enum([
  "Never",
  "Pickup",
  "Delivery",
  "PickupDelivery",
  "AllExceptShipper",
]);

export type ServiceIncidentType = z.infer<typeof serviceIncidentTypeSchema>;

const centsSchema = (label: string, max = 1_000_000) =>
  z
    .number()
    .int()
    .min(0, { error: () => translate("{0} cannot be negative", label) })
    .max(max, { error: () => translate("{0} is too large", label) });

const coverageCentsSchema = (label: string) =>
  z
    .number()
    .int()
    .min(0, { error: () => translate("{0} cannot be negative", label) })
    .max(100_000_000, { error: () => translate("{0} is too large", label) });

export const dispatchPreferenceSchema = z.enum([
  "customerExperience",
  "onTimeHistory",
  "safety",
  "acceptance",
  "trailerContinuity",
  "haulFit",
  "timeOffBuffer",
]);

export type DispatchPreference = z.infer<typeof dispatchPreferenceSchema>;

/** Mirrors dispatcheconomics.Default on the server: what an organization starts with. */
export const DEFAULT_DISPATCH_ECONOMICS = {
  emptyMileCents: 200,
  lateMinuteCents: 100,
  driverHourCents: 3_000,
  earlyGraceMinutes: 120,
  homeTimeTargetDays: 10,
  homeTimeDayCents: 7_500,
  fairnessMileCents: 3,
  preferences: {
    customerExperience: 5_000,
    onTimeHistory: 7_500,
    safety: 7_500,
    acceptance: 3_000,
    trailerContinuity: 10_000,
    haulFit: 6_000,
    timeOffBuffer: 10_000,
  },
  coverageMarginPercent: 30,
  coverageFloorCents: 40_000,
  latePremiumCents: 150_000,
  dueNowPremiumCents: 75_000,
  candidatesPerMove: 40,
  replanChangeCents: 2_500,
  anticipationMaxCents: 0,
} as const satisfies DispatchEconomics;

export const dispatchEconomicsSchema = z.object({
  emptyMileCents: centsSchema("Empty mile cost"),
  lateMinuteCents: centsSchema("Late minute cost"),
  driverHourCents: centsSchema("Driver hour cost"),
  earlyGraceMinutes: z
    .number()
    .int()
    .min(0, { error: () => translate("Early grace cannot be negative") })
    .max(1_440, { error: () => translate("Early grace cannot exceed 1440 minutes") }),
  homeTimeTargetDays: z
    .number()
    .int()
    .min(0, { error: () => translate("Home time target cannot be negative") })
    .max(60, { error: () => translate("Home time target cannot exceed 60 days") }),
  homeTimeDayCents: centsSchema("Home time cost"),
  fairnessMileCents: centsSchema("Fairness cost"),
  preferences: z.record(dispatchPreferenceSchema, centsSchema("Preference")),
  coverageMarginPercent: z
    .number()
    .int()
    .min(0, { error: () => translate("Coverage margin cannot be negative") })
    .max(100, { error: () => translate("Coverage margin cannot exceed 100%") }),
  coverageFloorCents: coverageCentsSchema("Coverage floor"),
  latePremiumCents: coverageCentsSchema("Late premium"),
  dueNowPremiumCents: coverageCentsSchema("Due now premium"),
  candidatesPerMove: z
    .number()
    .int()
    .min(5, { error: () => translate("Candidates per move must be at least 5") })
    .max(200, { error: () => translate("Candidates per move cannot exceed 200") }),
  replanChangeCents: centsSchema("Replan change cost"),
  anticipationMaxCents: centsSchema("Likely load value", 200_000).default(0),
});

export type DispatchEconomics = z.infer<typeof dispatchEconomicsSchema>;

export const DISPATCH_PREFERENCE_META: ReadonlyArray<{
  key: DispatchPreference;
  label: string;
  description: string;
}> = [
  {
    key: "customerExperience",
    label: "Customer experience",
    description:
      "What a driver who has never run this customer costs against one who runs it often.",
  },
  {
    key: "onTimeHistory",
    label: "On-time history",
    description: "What a driver far below your on-time goal costs against one who meets it.",
  },
  {
    key: "safety",
    label: "Safety history",
    description: "What five or more hours-of-service violations in 90 days cost against none.",
  },
  {
    key: "acceptance",
    label: "Acceptance",
    description: "What a driver who rarely accepts assignments costs against one who always does.",
  },
  {
    key: "trailerContinuity",
    label: "Trailer continuity",
    description: "What a trailer swap costs against keeping the trailer from the previous leg.",
  },
  {
    key: "haulFit",
    label: "Haul fit",
    description:
      "What a run that does not suit the driver's type costs, such as a local driver on a long haul.",
  },
  {
    key: "timeOffBuffer",
    label: "Time-off buffer",
    description:
      "What finishing right before approved time off costs against finishing two days ahead.",
  },
];

export const tenderExhaustedActionSchema = z.enum(["Notify", "Rebroadcast", "Rewaterfall"]);

export type TenderExhaustedAction = z.infer<typeof tenderExhaustedActionSchema>;

const PERCENT_SCALE = 4;

/**
 * A percentage the server keeps as NUMERIC(7,4). It travels as a decimal
 * string so the digits a dispatcher typed are the digits saved, and the
 * server's JSON number form is read back as the same string.
 */
function percentSchema(params: { min: string; max: string; fallback: string; label: string }) {
  return z
    .union([z.string(), z.number()])
    .transform((value) => (typeof value === "number" ? value.toString() : value))
    .pipe(
      decimalString(
        PERCENT_SCALE,
        translate("{0} must be a number with at most 4 decimals", translate(params.label)),
      )
        .refine((value) => compareDecimalStrings(value, params.min) >= 0, {
          error: () => translate("{0} cannot be below {1}%", translate(params.label), params.min),
        })
        .refine((value) => compareDecimalStrings(value, params.max) <= 0, {
          error: () => translate("{0} cannot exceed {1}%", translate(params.label), params.max),
        }),
    )
    .default(params.fallback);
}

export const complianceEnforcementLevelSchema = z.enum(["Warning", "Block", "Audit"]);

export type ComplianceEnforcementLevel = z.infer<typeof complianceEnforcementLevelSchema>;

function exceptionDueMinutes(fallback: number) {
  return z
    .number({
      error: () => translate("Enter how many minutes an exception has before it is overdue"),
    })
    .int()
    .min(5, { error: () => translate("Due time must be at least 5 minutes") })
    .max(10080, { error: () => translate("Due time cannot exceed 10080 minutes") })
    .default(fallback);
}

export const dispatchControlSchema = z
  .object({
    id: optionalStringSchema,
    version: versionSchema,
    createdAt: timestampSchema,
    updatedAt: timestampSchema,
    organizationId: optionalStringSchema,
    businessUnitId: optionalStringSchema,

    enableAutoAssignment: z.boolean(),
    autoProposeCoverage: z.boolean(),
    economics: dispatchEconomicsSchema.default(DEFAULT_DISPATCH_ECONOMICS),
    learnDispatchStyle: z.boolean().default(false),
    planForEndMarket: z.boolean().default(false),
    anticipateFreight: z.boolean().default(false),
    autoAssignConfidenceThreshold: z.coerce
      .number()
      .min(0, { error: () => translate("Confidence threshold must be at least 0") })
      .max(1, { error: () => translate("Confidence threshold cannot exceed 1") })
      .optional(),
    autoAssignMaxDeadheadMiles: z
      .number()
      .int()
      .positive({ error: () => translate("Max deadhead miles must be greater than 0") })
      .nullish(),
    autoAssignPlanningHorizonHours: z
      .number()
      .int()
      .min(1, { error: () => translate("Planning horizon must be at least 1 hour") })
      .max(336, { error: () => translate("Planning horizon cannot exceed 336 hours") })
      .optional(),
    coverageRiskWindowHours: z
      .number()
      .int()
      .min(1, { error: () => translate("Coverage risk window must be at least 1 hour") })
      .max(168, { error: () => translate("Coverage risk window cannot exceed 168 hours") })
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
      .nonnegative({ error: () => translate("Service failure target must be non-negative") })
      .nullish(),
    serviceFailureGracePeriod: z.number().int().nullish(),
    enableCheckCallMonitoring: z.boolean().default(false),
    checkCallIntervalHours: z
      .number({ error: () => translate("Enter how many hours a load may go without a check call") })
      .int()
      .min(1, { error: () => translate("Check call interval must be at least 1 hour") })
      .max(72, { error: () => translate("Check call interval cannot exceed 72 hours") })
      .default(4),
    enableEtaAlerts: z.boolean().default(true),
    etaRiskThresholdMinutes: z
      .number({ error: () => translate("Enter how many minutes of slack count as at risk") })
      .int()
      .min(0, { error: () => translate("ETA risk threshold cannot be negative") })
      .max(1440, { error: () => translate("ETA risk threshold cannot exceed 1440 minutes") })
      .default(60),
    etaDwellMinutes: z
      .number({ error: () => translate("Enter how long a stop takes once the truck is there") })
      .int()
      .min(0, { error: () => translate("Stop dwell time cannot be negative") })
      .max(720, { error: () => translate("Stop dwell time cannot exceed 720 minutes") })
      .default(60),
    exceptionDueInfoMinutes: exceptionDueMinutes(1440),
    exceptionDueWarningMinutes: exceptionDueMinutes(240),
    exceptionDueCriticalMinutes: exceptionDueMinutes(60),
    exceptionEscalationMinutes: z
      .number({ error: () => translate("Enter how often an overdue exception escalates") })
      .int()
      .min(5, { error: () => translate("Escalation interval must be at least 5 minutes") })
      .max(1440, { error: () => translate("Escalation interval cannot exceed 1440 minutes") })
      .default(60),
    exceptionMaxEscalations: z
      .number({ error: () => translate("Enter how many times an exception may escalate") })
      .int()
      .min(0, { error: () => translate("Escalations cannot be negative") })
      .max(10, { error: () => translate("Escalations cannot exceed 10") })
      .default(3),
    exceptionDefaultAssigneeId: z.string().nullish(),
    exceptionEscalationContactId: z.string().nullish(),
    exceptionMissedStopGraceMinutes: z
      .number({ error: () => translate("Enter how long after a stop window closes it is missed") })
      .int()
      .min(0, { error: () => translate("Missed stop grace cannot be negative") })
      .max(1440, { error: () => translate("Missed stop grace cannot exceed 1440 minutes") })
      .default(30),
    exceptionPodGraceHours: z
      .number({ error: () => translate("Enter how long after delivery a POD is due") })
      .int()
      .min(1, { error: () => translate("POD grace must be at least 1 hour") })
      .max(720, { error: () => translate("POD grace cannot exceed 720 hours") })
      .default(24),
    tenderCounterOffersEnabled: z.boolean().default(true),
    /** Null to a reader without the margins field group; the server keeps the stored floor. */
    tenderCounterMarginFloorPercent: percentSchema({
      min: "-100",
      max: "100",
      fallback: "10",
      label: "Margin floor",
    }).nullable(),
    tenderAutoAcceptCounters: z.boolean().default(false),
    tenderCounterDecisionMinutes: z
      .number({ error: () => translate("Enter how long dispatch has to answer a counter") })
      .int()
      .min(5, { error: () => translate("Counter decision time must be at least 5 minutes") })
      .max(1440, { error: () => translate("Counter decision time cannot exceed 1440 minutes") })
      .default(60),
    tenderExhaustedAction: tenderExhaustedActionSchema.default("Notify"),
    tenderEscalationRateIncreasePercent: percentSchema({
      min: "0",
      max: "100",
      fallback: "0",
      label: "Rate increase",
    }),
    tenderEscalationOfferTtlMinutes: z
      .number({ error: () => translate("Enter how long an escalated offer stays open") })
      .int()
      .min(5, { error: () => translate("Escalated offers must stay open at least 5 minutes") })
      .max(10080, {
        error: () => translate("Escalated offers cannot stay open beyond 10080 minutes"),
      })
      .default(60),
    tenderMaxEscalations: z
      .number({ error: () => translate("Enter how many times an exhausted tender may escalate") })
      .int()
      .min(0, { error: () => translate("Escalations cannot be negative") })
      .max(5, { error: () => translate("Escalations cannot exceed 5") })
      .default(1),
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
      error: () => translate("Service failure grace period must be greater than 0"),
    },
  );

export type DispatchControl = z.infer<typeof dispatchControlSchema>;
