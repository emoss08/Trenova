import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";

export const safetyEventKindSchema = z.enum([
  "Accident",
  "Incident",
  "NearMiss",
  "Citation",
  "Inspection",
]);
export type SafetyEventKind = z.infer<typeof safetyEventKindSchema>;

export const safetySeveritySchema = z.enum(["Minor", "Moderate", "Major", "Critical"]);
export type SafetySeverity = z.infer<typeof safetySeveritySchema>;

export const safetyEventStatusSchema = z.enum(["Open", "UnderReview", "Closed"]);
export type SafetyEventStatus = z.infer<typeof safetyEventStatusSchema>;

export const inspectionResultSchema = z.enum(["Pass", "Fail", "OutOfService"]);
export type InspectionResult = z.infer<typeof inspectionResultSchema>;

export const safetyRatingSchema = z.enum(["Excellent", "Good", "Watch", "AtRisk"]);
export type SafetyRating = z.infer<typeof safetyRatingSchema>;

export const disciplinaryLevelSchema = z.enum([
  "Coaching",
  "VerbalWarning",
  "WrittenWarning",
  "FinalWarning",
  "Suspension",
  "Termination",
]);
export type DisciplinaryLevel = z.infer<typeof disciplinaryLevelSchema>;

export const disciplinaryStatusSchema = z.enum(["Active", "Expired", "Rescinded"]);
export type DisciplinaryStatus = z.infer<typeof disciplinaryStatusSchema>;

export const recognitionKindSchema = z.enum([
  "SafetyMilestone",
  "CustomerPraise",
  "Performance",
  "Tenure",
  "TeamPlayer",
  "Other",
]);
export type RecognitionKind = z.infer<typeof recognitionKindSchema>;

export const SAFETY_EVENT_KIND_LABELS: Record<SafetyEventKind, string> = {
  Accident: "Accident",
  Incident: "Incident",
  NearMiss: "Near miss",
  Citation: "Citation",
  Inspection: "Inspection",
};

export const SAFETY_SEVERITY_LABELS: Record<SafetySeverity, string> = {
  Minor: "Minor",
  Moderate: "Moderate",
  Major: "Major",
  Critical: "Critical",
};

export const SAFETY_EVENT_STATUS_LABELS: Record<SafetyEventStatus, string> = {
  Open: "Open",
  UnderReview: "Under review",
  Closed: "Closed",
};

export const INSPECTION_RESULT_LABELS: Record<InspectionResult, string> = {
  Pass: "Passed",
  Fail: "Failed",
  OutOfService: "Out of service",
};

export const DISCIPLINARY_LEVEL_LABELS: Record<DisciplinaryLevel, string> = {
  Coaching: "Coaching",
  VerbalWarning: "Verbal warning",
  WrittenWarning: "Written warning",
  FinalWarning: "Final warning",
  Suspension: "Suspension",
  Termination: "Termination",
};

export const DISCIPLINARY_STATUS_LABELS: Record<DisciplinaryStatus, string> = {
  Active: "Active",
  Expired: "Expired",
  Rescinded: "Rescinded",
};

export const RECOGNITION_KIND_LABELS: Record<RecognitionKind, string> = {
  SafetyMilestone: "Safety milestone",
  CustomerPraise: "Customer praise",
  Performance: "Performance",
  Tenure: "Tenure",
  TeamPlayer: "Team player",
  Other: "Other",
};

const optionalTrimmed = (max: number, message: string) =>
  z
    .string()
    .nullable()
    .optional()
    .transform((value) => {
      const trimmed = value?.trim() ?? "";
      return trimmed === "" ? null : trimmed;
    })
    .pipe(z.string().max(max, { message }).nullable());

const MONEY_PATTERN = /^\d+(\.\d{1,2})?$/;

const optionalMoney = z
  .string()
  .nullable()
  .optional()
  .transform((value) => {
    const trimmed = value?.trim() ?? "";
    return trimmed === "" ? null : trimmed;
  })
  .pipe(
    z
      .string()
      .regex(MONEY_PATTERN, { message: "Enter an amount with up to two decimals" })
      .nullable(),
  );

export const safetyEventFormSchema = z
  .object({
    kind: safetyEventKindSchema,
    severity: safetySeveritySchema,
    occurredAt: z.number().int().positive({ message: "When did it happen?" }),
    location: optionalTrimmed(255, "Location cannot exceed 255 characters"),
    description: z
      .string()
      .trim()
      .min(1, { message: "Describe what happened" })
      .max(4000, { message: "Description cannot exceed 4000 characters" }),
    preventable: z.boolean(),
    points: z.number().int().min(0, { message: "Points cannot be negative" }),
    referenceNumber: optionalTrimmed(100, "Reference cannot exceed 100 characters"),
    shipmentId: z.string().nullable().optional(),
    inspectionLevel: z.number().int().min(1).max(6).nullable(),
    inspectionResult: inspectionResultSchema.nullable(),
    fineAmount: optionalMoney,
    costAmount: optionalMoney,
  })
  .superRefine((values, ctx) => {
    if (values.kind === "Inspection" && values.inspectionResult === null) {
      ctx.addIssue({
        code: "custom",
        path: ["inspectionResult"],
        message: translate("Record how the inspection went"),
      });
    }
    if (values.kind !== "Inspection" && values.inspectionResult !== null) {
      ctx.addIssue({
        code: "custom",
        path: ["inspectionResult"],
        message: translate("Only inspections carry a result"),
      });
    }
  });
export type SafetyEventFormValues = z.infer<typeof safetyEventFormSchema>;

export const closeSafetyEventFormSchema = z.object({
  resolution: z
    .string()
    .trim()
    .min(3, { message: "Say how the event was resolved" })
    .max(4000, { message: "Resolution cannot exceed 4000 characters" }),
});
export type CloseSafetyEventFormValues = z.infer<typeof closeSafetyEventFormSchema>;

export const issueActionFormSchema = z
  .object({
    level: disciplinaryLevelSchema,
    reason: z
      .string()
      .trim()
      .min(3, { message: "Say why the action is being taken" })
      .max(4000, { message: "Reason cannot exceed 4000 characters" }),
    details: optionalTrimmed(4000, "Details cannot exceed 4000 characters"),
    occurredAt: z.number().int().positive().nullable(),
    expiresAt: z.number().int().positive().nullable(),
    suspensionDays: z.number().int().min(1, { message: "At least one day" }).nullable(),
    safetyEventId: z.string().nullable().optional(),
    recordEmploymentEvent: z.boolean(),
  })
  .superRefine((values, ctx) => {
    if (values.level === "Suspension" && values.suspensionDays === null) {
      ctx.addIssue({
        code: "custom",
        path: ["suspensionDays"],
        message: translate("How many days is the suspension?"),
      });
    }
    if (values.level !== "Suspension" && values.suspensionDays !== null) {
      ctx.addIssue({
        code: "custom",
        path: ["suspensionDays"],
        message: translate("Only suspensions carry days"),
      });
    }
  });
export type IssueActionFormValues = z.infer<typeof issueActionFormSchema>;

export const rescindActionFormSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(3, { message: "Say why the action is rescinded" })
    .max(255, { message: "Reason cannot exceed 255 characters" }),
});
export type RescindActionFormValues = z.infer<typeof rescindActionFormSchema>;

export const recognitionFormSchema = z.object({
  kind: recognitionKindSchema,
  title: z
    .string()
    .trim()
    .min(1, { message: "Give the recognition a title" })
    .max(120, { message: "Title cannot exceed 120 characters" }),
  message: optionalTrimmed(2000, "Message cannot exceed 2000 characters"),
  occurredAt: z.number().int().positive({ message: "Choose the date" }),
  visibleToWorker: z.boolean(),
});
export type RecognitionFormValues = z.infer<typeof recognitionFormSchema>;
