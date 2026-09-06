import { z } from "zod";

export {
  drugAlcoholStatusSchema,
  returnToDutyStatusSchema,
  type DrugAlcoholStatus,
  type ReturnToDutyStatus,
} from "./worker-drug-alcohol-status";

export const dotTestTypeSchema = z.enum([
  "PreEmployment",
  "Random",
  "PostAccident",
  "ReasonableSuspicion",
  "ReturnToDuty",
  "FollowUp",
  "Other",
]);
export type DOTTestType = z.infer<typeof dotTestTypeSchema>;

export const dotTestSubstanceSchema = z.enum(["Drug", "Alcohol"]);
export type DOTTestSubstance = z.infer<typeof dotTestSubstanceSchema>;

export const dotTestStatusSchema = z.enum([
  "Scheduled",
  "Collected",
  "AwaitingResult",
  "Completed",
  "Cancelled",
]);
export type DOTTestStatus = z.infer<typeof dotTestStatusSchema>;

export const dotTestResultSchema = z.enum([
  "Pending",
  "Negative",
  "NegativeDilute",
  "Positive",
  "Refusal",
  "Adulterated",
  "Substituted",
  "Invalid",
  "Cancelled",
]);
export type DOTTestResult = z.infer<typeof dotTestResultSchema>;

export const dotViolationTypeSchema = z.enum([
  "PositiveTest",
  "TestRefusal",
  "AlcoholUse",
  "DrugUse",
  "ActualKnowledge",
  "Other",
]);
export type DOTViolationType = z.infer<typeof dotViolationTypeSchema>;

export const clearinghouseQueryTypeSchema = z.enum([
  "PreEmploymentFull",
  "AnnualLimited",
  "Full",
  "Limited",
]);
export type ClearinghouseQueryType = z.infer<typeof clearinghouseQueryTypeSchema>;

export const clearinghouseResultSchema = z.enum([
  "Pending",
  "NoViolations",
  "ViolationsFound",
  "ConsentDenied",
]);
export type ClearinghouseResult = z.infer<typeof clearinghouseResultSchema>;

export const randomPeriodSchema = z.enum(["Monthly", "Quarterly", "SemiAnnual", "Annual"]);
export type RandomPeriod = z.infer<typeof randomPeriodSchema>;

export const randomEntryStatusSchema = z.enum([
  "Selected",
  "Notified",
  "Completed",
  "Excused",
  "Missed",
]);
export type RandomEntryStatus = z.infer<typeof randomEntryStatusSchema>;

/**
 * Scheduling a collection. Reasonable-suspicion and post-accident tests carry a
 * narrative because a regulator will ask what prompted them; the server
 * enforces the same rule.
 */
export const dotTestFormSchema = z
  .object({
    testType: dotTestTypeSchema,
    substance: dotTestSubstanceSchema,
    isDot: z.boolean(),
    reason: z.string().nullable(),
    scheduledAt: z.number().nullable(),
    collectedAt: z.number().nullable(),
    collectionSite: z.string().max(150).nullable(),
    collectorName: z.string().max(100).nullable(),
    specimenId: z.string().max(100).nullable(),
    notes: z.string().nullable(),
    drawEntryId: z.string().nullable(),
  })
  .refine(
    (values) =>
      !["ReasonableSuspicion", "PostAccident"].includes(values.testType) ||
      Boolean(values.reason?.trim()),
    {
      message: "Record what prompted this test",
      path: ["reason"],
    },
  );
export type DOTTestFormValues = z.infer<typeof dotTestFormSchema>;

/**
 * Recording what came back. An alcohol test needs its concentration: the number
 * decides the result, so the form cannot let it through without one.
 */
export const dotTestResultFormSchema = z
  .object({
    substance: dotTestSubstanceSchema,
    result: dotTestResultSchema,
    resultAt: z.number().nullable(),
    labName: z.string().max(100).nullable(),
    mroName: z.string().max(100).nullable(),
    mroVerifiedAt: z.number().nullable(),
    alcoholConcentration: z.string().nullable(),
    notes: z.string().nullable(),
  })
  .refine(
    (values) => values.substance !== "Alcohol" || Boolean(values.alcoholConcentration?.trim()),
    {
      message: "Record the concentration that was measured",
      path: ["alcoholConcentration"],
    },
  )
  .refine((values) => values.result !== "Pending", {
    message: "Choose the result that came back",
    path: ["result"],
  });
export type DOTTestResultFormValues = z.infer<typeof dotTestResultFormSchema>;

export const clearinghouseQueryFormSchema = z
  .object({
    queryType: clearinghouseQueryTypeSchema,
    requestedAt: z.number(),
    consentObtainedAt: z.number().nullable(),
    consentExpiresAt: z.number().nullable(),
    reference: z.string().max(100).nullable(),
    notes: z.string().nullable(),
  })
  .refine(
    (values) =>
      !["PreEmploymentFull", "Full"].includes(values.queryType) ||
      Boolean(values.consentObtainedAt),
    {
      message: "A full query requires the driver's electronic consent",
      path: ["consentObtainedAt"],
    },
  );
export type ClearinghouseQueryFormValues = z.infer<typeof clearinghouseQueryFormSchema>;

export const clearinghouseAnswerFormSchema = z
  .object({
    result: clearinghouseResultSchema,
    completedAt: z.number(),
    violationCount: z.number().min(0),
    reference: z.string().max(100).nullable(),
    notes: z.string().nullable(),
  })
  .refine((values) => values.result !== "Pending", {
    message: "Choose the answer that came back",
    path: ["result"],
  })
  .refine((values) => values.result !== "ViolationsFound" || values.violationCount > 0, {
    message: "Record how many violations the query returned",
    path: ["violationCount"],
  });
export type ClearinghouseAnswerFormValues = z.infer<typeof clearinghouseAnswerFormSchema>;

/**
 * The return-to-duty process. Every field is optional because the office fills
 * them in as each step happens; the stage is derived from what has been
 * recorded rather than chosen.
 */
export const violationProgressFormSchema = z
  .object({
    sapName: z.string().max(100).nullable(),
    sapReferredAt: z.number().nullable(),
    sapEvaluationCompletedAt: z.number().nullable(),
    followUpTestCount: z.number().nullable(),
    followUpEndsAt: z.number().nullable(),
    reportedToClearinghouseAt: z.number().nullable(),
    notes: z.string().nullable(),
  })
  .refine((values) => !values.sapEvaluationCompletedAt || Boolean(values.sapReferredAt), {
    message: "Record the referral before the evaluation that followed it",
    path: ["sapReferredAt"],
  })
  .refine(
    (values) =>
      values.followUpTestCount === null ||
      values.followUpTestCount === 0 ||
      values.followUpTestCount >= 6,
    {
      message: "A follow-up programme is at least six tests (49 CFR 382.311)",
      path: ["followUpTestCount"],
    },
  );
export type ViolationProgressFormValues = z.infer<typeof violationProgressFormSchema>;

export const randomPoolFormSchema = z.object({
  code: z.string().min(1, "Code is required").max(50),
  name: z.string().min(1, "Name is required").max(100),
  description: z.string().nullable(),
  status: z.enum(["Active", "Inactive"]),
  period: randomPeriodSchema,
  drugRatePercent: z.number().min(0).max(100),
  alcoholRatePercent: z.number().min(0).max(100),
  includedDriverTypes: z.array(z.string()).nullable(),
  isDefault: z.boolean(),
});
export type RandomPoolFormValues = z.infer<typeof randomPoolFormSchema>;
