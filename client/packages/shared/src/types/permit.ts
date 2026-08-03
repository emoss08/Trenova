import { z } from "zod";
import { decimalStringSchema, nullableStringSchema, optionalStringSchema } from "./helpers";

export const permitStatusSchema = z.enum(["Pending", "Active", "Expired", "Void"]);
export type PermitStatus = z.infer<typeof permitStatusSchema>;

export const requirementStatusSchema = z.enum(["Open", "Satisfied", "Superseded", "Waived"]);
export type RequirementStatus = z.infer<typeof requirementStatusSchema>;

export const verificationStateSchema = z.enum(["Unverified", "Verified", "Disputed"]);
export type VerificationState = z.infer<typeof verificationStateSchema>;

export const triggerKindSchema = z.enum(["Width", "Height", "Length", "Weight", "Superload"]);
export type TriggerKind = z.infer<typeof triggerKindSchema>;

export const escortRoleSchema = z.enum(["Front", "Rear", "Police", "RouteSurvey"]);
export type EscortRole = z.infer<typeof escortRoleSchema>;

export const restrictionKindSchema = z.enum([
  "DaylightOnly",
  "RushHour",
  "Weekend",
  "Holiday",
  "NightOnly",
]);
export type RestrictionKind = z.infer<typeof restrictionKindSchema>;

export const exceedanceSchema = z.object({
  trigger: triggerKindSchema,
  limit: z.number(),
  actual: z.number(),
  overBy: z.number(),
  superload: z.boolean().default(false),
});
export type Exceedance = z.infer<typeof exceedanceSchema>;

export const escortRequirementSchema = z.object({
  role: escortRoleSchema,
  trigger: triggerKindSchema,
  threshold: z.number(),
  actual: z.number(),
  note: optionalStringSchema,
});
export type EscortRequirement = z.infer<typeof escortRequirementSchema>;

export const ruleProvenanceSchema = z.object({
  ruleId: z.string(),
  stateCode: z.string(),
  stateName: z.string(),
  verificationState: verificationStateSchema,
  verifiedAt: z.number().nullish(),
  sourceNote: optionalStringSchema,
  sourceUrl: optionalStringSchema,
  maxWidthFeet: z.number(),
  maxHeightFeet: z.number(),
  maxLengthFeet: z.number(),
  maxWeightPounds: z.number(),
  overriddenFields: z.array(z.string()).nullish(),
  overrideReason: optionalStringSchema,
});
export type RuleProvenance = z.infer<typeof ruleProvenanceSchema>;

export const permitRequirementSchema = z.object({
  id: z.string(),
  shipmentId: z.string(),
  stateId: z.string(),
  status: requirementStatusSchema,
  routeSequence: z.number().int(),
  exceedances: z.array(exceedanceSchema).nullish(),
  escorts: z.array(escortRequirementSchema).nullish(),
  restrictions: z.array(restrictionKindSchema).nullish(),
  leadTimeDays: z.number().int(),
  validityDays: z.number().int(),
  estimatedFee: decimalStringSchema,
  isSuperload: z.boolean(),
  satisfiedByPermitId: nullableStringSchema,
  waivedById: nullableStringSchema,
  waiverReason: optionalStringSchema,
  provenance: ruleProvenanceSchema.nullish(),
  derivedAt: z.number(),
});
export type PermitRequirement = z.infer<typeof permitRequirementSchema>;

export const assessedMeasurementsSchema = z.object({
  widthFeet: z.number(),
  heightFeet: z.number(),
  lengthFeet: z.number(),
  weightPounds: z.number(),
});
export type AssessedMeasurements = z.infer<typeof assessedMeasurementsSchema>;

export const assessedJurisdictionSchema = z.object({
  stateId: z.string(),
  stateCode: z.string(),
  stateName: z.string(),
  sequence: z.number().int(),
  verificationState: verificationStateSchema,
  sourceNote: optionalStringSchema,
  sourceUrl: optionalStringSchema,
  maxWidthFeet: z.number(),
  maxHeightFeet: z.number(),
  maxLengthFeet: z.number(),
  maxWeightPounds: z.number(),
  widthHeadroomFeet: z.number(),
  heightHeadroomFeet: z.number(),
  lengthHeadroomFeet: z.number(),
  weightHeadroomPounds: z.number(),
  requiresPermit: z.boolean(),
  restrictions: z.array(restrictionKindSchema).nullish(),
});
export type AssessedJurisdiction = z.infer<typeof assessedJurisdictionSchema>;

export const permitAssessmentSchema = z.object({
  measurements: assessedMeasurementsSchema,
  jurisdictions: z.array(assessedJurisdictionSchema).nullish(),
  requirements: z.array(permitRequirementSchema).nullish(),
  open: z.array(permitRequirementSchema).nullish(),
  expiringBeforeDelivery: z.array(permitRequirementSchema).nullish(),
  totalEscorts: z.number().int(),
  totalEstimatedFee: decimalStringSchema,
  maxLeadTimeDays: z.number().int(),
  earliestPickup: z.number(),
  routeResolved: z.boolean(),
  feeIsBaseOnly: z.boolean(),
});
export type PermitAssessment = z.infer<typeof permitAssessmentSchema>;
