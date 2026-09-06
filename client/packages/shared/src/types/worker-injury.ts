import { z } from "zod";

export const oshaCaseClassificationSchema = z.enum([
  "NotRecordable",
  "FirstAidOnly",
  "OtherRecordable",
  "JobTransferOrRestriction",
  "DaysAway",
  "Death",
]);
export type OSHACaseClassification = z.infer<typeof oshaCaseClassificationSchema>;

export const oshaIllnessTypeSchema = z.enum([
  "Injury",
  "SkinDisorder",
  "RespiratoryCondition",
  "Poisoning",
  "HearingLoss",
  "OtherIllness",
]);
export type OSHAIllnessType = z.infer<typeof oshaIllnessTypeSchema>;

export const injuryTreatmentSchema = z.enum([
  "None",
  "FirstAid",
  "MedicalTreatment",
  "EmergencyRoom",
  "Hospitalized",
]);
export type InjuryTreatment = z.infer<typeof injuryTreatmentSchema>;

export const injuryCaseStatusSchema = z.enum(["Open", "Closed"]);
export type InjuryCaseStatus = z.infer<typeof injuryCaseStatusSchema>;

export const workersCompClaimStatusSchema = z.enum([
  "NotFiled",
  "Filed",
  "Accepted",
  "Denied",
  "Closed",
]);
export type WorkersCompClaimStatus = z.infer<typeof workersCompClaimStatusSchema>;

/**
 * One injury or illness case. The refinements mirror the server's: the log
 * records only the most serious outcome, so a case with days away cannot be
 * filed in a lesser column and disappear from the count that matters.
 */
export const injuryFormSchema = z
  .object({
    occurredAt: z.number(),
    reportedAt: z.number().nullable(),
    returnedToWorkAt: z.number().nullable(),
    description: z.string().min(1, "Describe what happened"),
    location: z.string().max(255).nullable(),
    bodyPart: z.string().max(100).nullable(),
    harmfulAgent: z.string().max(255).nullable(),
    classification: oshaCaseClassificationSchema,
    illnessType: oshaIllnessTypeSchema,
    treatment: injuryTreatmentSchema,
    status: injuryCaseStatusSchema,
    daysAway: z.number().min(0).max(180),
    daysRestricted: z.number().min(0).max(180),
    privacyCase: z.boolean(),
    claimStatus: workersCompClaimStatusSchema,
    claimNumber: z.string().max(100).nullable(),
    claimCarrier: z.string().max(150).nullable(),
    claimFiledAt: z.number().nullable(),
    claimClosedAt: z.number().nullable(),
    safetyEventId: z.string().nullable(),
    notes: z.string().nullable(),
  })
  .refine(
    (values) =>
      values.daysAway === 0 ||
      values.classification === "DaysAway" ||
      values.classification === "Death",
    {
      message: "A case with days away from work is a days-away case",
      path: ["classification"],
    },
  )
  .refine(
    (values) =>
      values.daysRestricted === 0 ||
      values.classification === "OtherRecordable" ||
      values.classification === "JobTransferOrRestriction" ||
      values.classification === "DaysAway" ||
      values.classification === "Death",
    {
      message: "A case with restricted days is recordable",
      path: ["classification"],
    },
  )
  .refine((values) => values.claimStatus === "NotFiled" || Boolean(values.claimFiledAt), {
    message: "Record when the claim was filed",
    path: ["claimFiledAt"],
  })
  .refine((values) => values.claimStatus !== "Closed" || Boolean(values.claimClosedAt), {
    message: "Record when the claim was closed",
    path: ["claimClosedAt"],
  })
  .refine(
    (values) => values.returnedToWorkAt === null || values.returnedToWorkAt >= values.occurredAt,
    {
      message: "The return to work cannot pre-date the injury",
      path: ["returnedToWorkAt"],
    },
  );
export type InjuryFormValues = z.infer<typeof injuryFormSchema>;

/**
 * The establishment figures the 300A needs and the log cannot supply.
 */
export const oshaSummaryFormSchema = z.object({
  naicsCode: z.string().max(10).nullable(),
  averageEmployees: z.number().min(0),
  totalHoursWorked: z.number().min(0),
  executiveName: z.string().max(100).nullable(),
  executiveTitle: z.string().max(100).nullable(),
  executivePhone: z.string().max(30).nullable(),
  submittedAt: z.number().nullable(),
  submissionReference: z.string().max(100).nullable(),
  notes: z.string().nullable(),
});
export type OSHASummaryFormValues = z.infer<typeof oshaSummaryFormSchema>;
