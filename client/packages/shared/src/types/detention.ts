import { z } from "zod";
import { decimalStringSchema } from "./helpers";
import { tenantInfoSchema } from "./helpers";

export const detentionPolicyStatusSchema = z.enum(["Active", "Inactive", "Draft"]);
export type DetentionPolicyStatus = z.infer<typeof detentionPolicyStatusSchema>;

export const clockStartBasisSchema = z.enum([
  "Arrival",
  "Appointment",
  "LaterOfArrivalOrAppointment",
  "EarlierOfArrivalOrAppointment",
]);
export type ClockStartBasis = z.infer<typeof clockStartBasisSchema>;

export const lateArrivalRuleSchema = z.enum([
  "NoEffect",
  "Forfeit",
  "ClockFromAppointment",
  "ReduceFreeTime",
]);
export type LateArrivalRule = z.infer<typeof lateArrivalRuleSchema>;

export const detentionRoundingModeSchema = z.enum(["Up", "Down", "Nearest", "Exact"]);
export type DetentionRoundingMode = z.infer<typeof detentionRoundingModeSchema>;

export const detentionRateSourceSchema = z.enum(["Accessorial", "Tiers"]);
export type DetentionRateSource = z.infer<typeof detentionRateSourceSchema>;

export const tierRateUnitSchema = z.enum(["Hour", "Day", "Flat"]);
export type TierRateUnit = z.infer<typeof tierRateUnitSchema>;

export const notificationRequirementSchema = z.enum(["None", "Advisory", "Required"]);
export type NotificationRequirement = z.infer<typeof notificationRequirementSchema>;

export const unnotifiedBehaviorSchema = z.enum(["Bill", "Flag", "Suppress"]);
export type UnnotifiedBehavior = z.infer<typeof unnotifiedBehaviorSchema>;

export const capScopeSchema = z.enum(["PerStop", "PerDay", "PerShipment"]);
export type CapScope = z.infer<typeof capScopeSchema>;

export const capKindSchema = z.enum([
  "None",
  "MaxBillableMinutes",
  "MaxChargePerStop",
  "MaxChargePerDay",
  "MaxChargePerShipment",
  "LayoverBoundary",
]);
export type CapKind = z.infer<typeof capKindSchema>;

export const occurrenceStatusSchema = z.enum([
  "Accruing",
  "Pending",
  "Approved",
  "Billed",
  "Waived",
  "Disputed",
  "NotBillable",
]);
export type OccurrenceStatus = z.infer<typeof occurrenceStatusSchema>;

export const detentionNotificationStatusSchema = z.enum([
  "NotRequired",
  "Pending",
  "Sent",
  "Late",
  "Missed",
  "Failed",
]);
export type DetentionNotificationStatus = z.infer<
  typeof detentionNotificationStatusSchema
>;

export const waiverReasonSchema = z.enum([
  "Weather",
  "FacilityClosure",
  "CarrierFault",
  "EquipmentIssue",
  "CustomerGoodwill",
  "DataCorrection",
  "ForceMajeure",
  "Other",
]);
export type WaiverReason = z.infer<typeof waiverReasonSchema>;

export const evidenceKindSchema = z.enum([
  "Arrival",
  "Departure",
  "Appointment",
  "Notice",
  "Document",
  "StatusChange",
  "Recalculated",
  "Waiver",
  "Dispute",
]);
export type EvidenceKind = z.infer<typeof evidenceKindSchema>;

export const evidenceSourceSchema = z.enum([
  "Telematics",
  "Geofence",
  "DriverApp",
  "EDI",
  "Document",
  "Manual",
  "System",
]);
export type EvidenceSource = z.infer<typeof evidenceSourceSchema>;

export const noticeKindSchema = z.enum(["Warning", "Started", "Update", "Final"]);
export type NoticeKind = z.infer<typeof noticeKindSchema>;

export const noticeChannelSchema = z.enum(["Email", "EDI", "Portal", "Manual"]);
export type NoticeChannel = z.infer<typeof noticeChannelSchema>;

export const noticeDeliveryStatusSchema = z.enum([
  "Queued",
  "Sent",
  "Delivered",
  "Opened",
  "Bounced",
  "Failed",
]);
export type NoticeDeliveryStatus = z.infer<typeof noticeDeliveryStatusSchema>;

export const scoreBandSchema = z.enum(["Strong", "Adequate", "Weak", "AtRisk"]);
export type ScoreBand = z.infer<typeof scoreBandSchema>;

export const detentionPolicyTierSchema = z.object({
  id: z.string().optional(),
  fromMinute: z.number().int().min(0),
  toMinute: z.number().int().min(1).nullish(),
  rate: z.coerce.number().min(0),
  rateUnit: tierRateUnitSchema.default("Hour"),
  label: z.string().max(100).default(""),
  sortOrder: z.number().int().default(0),
});
export type DetentionPolicyTier = z.infer<typeof detentionPolicyTierSchema>;

export const detentionPolicySchema = z
  .object({
    ...tenantInfoSchema.shape,

    name: z.string().min(1, { error: "Name is required" }).max(100),
    code: z.string().min(1, { error: "Code is required" }).max(50),
    description: z.string().default(""),
    status: detentionPolicyStatusSchema.default("Draft"),

    isOrgDefault: z.boolean().default(false),
    priority: z.number().int().default(0),
    specificityScore: z.number().int().default(0),
    customerId: z.string().nullish(),
    locationId: z.string().nullish(),
    shipmentTypeIds: z.array(z.string()).nullish(),
    serviceTypeIds: z.array(z.string()).nullish(),
    commodityIds: z.array(z.string()).nullish(),
    stopTypes: z.array(z.string()).nullish(),
    appointmentStopsOnly: z.boolean().default(false),

    effectiveStartDate: z.number().int().nullish(),
    effectiveEndDate: z.number().int().nullish(),

    clockStartBasis: clockStartBasisSchema.default("LaterOfArrivalOrAppointment"),
    lateArrivalRule: lateArrivalRuleSchema.default("NoEffect"),
    lateArrivalGraceMinutes: z.number().int().min(0).default(0),

    billingFreeMinutes: z.number().int().min(0).max(10080).default(120),
    pickupFreeMinutes: z.number().int().min(0).max(10080).nullish(),
    deliveryFreeMinutes: z.number().int().min(0).max(10080).nullish(),
    payFreeMinutes: z.number().int().min(0).max(10080).nullish(),
    minimumBillableMinutes: z.number().int().min(0).default(0),
    billingIncrementMinutes: z.number().int().min(0).max(1440).default(15),
    roundingMode: detentionRoundingModeSchema.default("Up"),

    rateSource: detentionRateSourceSchema.default("Accessorial"),
    accessorialChargeId: z.string().min(1, { error: "Accessorial charge is required" }),
    tiers: z.array(detentionPolicyTierSchema).default([]),

    maxBillableMinutesPerStop: z.number().int().min(1).nullish(),
    maxChargePerStop: decimalStringSchema,
    maxChargePerDay: decimalStringSchema,
    maxChargePerShipment: decimalStringSchema,
    dayBoundaryMode: capScopeSchema.default("PerStop"),
    convertToLayoverAtMinutes: z.number().int().min(1).nullish(),
    layoverAccessorialChargeId: z.string().nullish(),

    notificationRequirement: notificationRequirementSchema.default("None"),
    notificationLeadMinutes: z.number().int().min(0).default(30),
    notificationDeadlineMinutes: z.number().int().min(0).default(0),
    unnotifiedBehavior: unnotifiedBehaviorSchema.default("Bill"),
    autoSendNotice: z.boolean().default(false),
    sendDepartureSummary: z.boolean().default(false),
    requireApprovalOverAmount: decimalStringSchema,
    autoApproveUnderAmount: decimalStringSchema,

    currency: z.string().length(3).default("USD"),
    comments: z.string().default(""),
  })
  .refine((data) => data.roundingMode === "Exact" || data.billingIncrementMinutes > 0, {
    error: "Billing increment must be greater than zero unless rounding is Exact",
    path: ["billingIncrementMinutes"],
  })
  .refine((data) => data.rateSource !== "Tiers" || data.tiers.length > 0, {
    error: "Add at least one rate tier when the rate source is Tiers",
    path: ["tiers"],
  })
  .refine(
    (data) =>
      data.notificationRequirement !== "Required" || data.unnotifiedBehavior !== "Bill",
    {
      error: "A required notice must Flag or Suppress billing when it is missed",
      path: ["unnotifiedBehavior"],
    },
  )
  .refine(
    (data) =>
      !data.effectiveStartDate ||
      !data.effectiveEndDate ||
      data.effectiveEndDate > data.effectiveStartDate,
    {
      error: "Expiration must be after the effective date",
      path: ["effectiveEndDate"],
    },
  );

export type DetentionPolicy = z.infer<typeof detentionPolicySchema>;

export const tierSnapshotSchema = z.object({
  fromMinute: z.number().int(),
  toMinute: z.number().int().nullish(),
  rate: z.coerce.number(),
  rateUnit: tierRateUnitSchema,
  label: z.string().default(""),
});

export const policySnapshotSchema = z.object({
  policyId: z.string(),
  policyCode: z.string(),
  policyName: z.string(),
  specificityScore: z.number().int(),
  priority: z.number().int(),
  clockStartBasis: clockStartBasisSchema,
  lateArrivalRule: lateArrivalRuleSchema,
  freeMinutes: z.number().int(),
  payFreeMinutes: z.number().int(),
  minimumBillableMinutes: z.number().int(),
  billingIncrementMinutes: z.number().int(),
  roundingMode: detentionRoundingModeSchema,
  rateSource: detentionRateSourceSchema,
  accessorialChargeId: z.string(),
  accessorialCode: z.string(),
  flatRate: z.coerce.number(),
  flatRateUnit: tierRateUnitSchema,
  tiers: z.array(tierSnapshotSchema).nullish(),
  notificationRequirement: notificationRequirementSchema,
  unnotifiedBehavior: unnotifiedBehaviorSchema,
  currency: z.string(),
  capturedAt: z.number().int(),
});
export type PolicySnapshot = z.infer<typeof policySnapshotSchema>;

export const traceStepKindSchema = z.enum([
  "PolicyResolved",
  "ClockStart",
  "LateArrival",
  "ClockStop",
  "RawDwell",
  "FreeTime",
  "MinimumThreshold",
  "Rounding",
  "MinutesCap",
  "TierApplied",
  "FlatRateApplied",
  "AmountCap",
  "LayoverConversion",
  "NotificationGate",
  "DriverPay",
  "Final",
]);
export type TraceStepKind = z.infer<typeof traceStepKindSchema>;

export const traceStepSchema = z.object({
  kind: traceStepKindSchema,
  label: z.string(),
  detail: z.string().default(""),
  term: z.string().default(""),
  minutes: z.number().int().nullish(),
  amount: z.coerce.number().nullish(),
  timestamp: z.number().int().nullish(),
  isReduction: z.boolean().default(false),
});
export type TraceStep = z.infer<typeof traceStepSchema>;

export const calculationTraceSchema = z.object({
  steps: z.array(traceStepSchema).default([]),
  engineVersion: z.string(),
  computedAt: z.number().int(),
  deterministic: z.boolean().default(true),
});
export type CalculationTrace = z.infer<typeof calculationTraceSchema>;

export const detentionEvidenceSchema = z.object({
  id: z.string(),
  detentionOccurrenceId: z.string(),
  sequence: z.number().int(),
  kind: evidenceKindSchema,
  source: evidenceSourceSchema,
  summary: z.string(),
  observedAt: z.number().int(),
  recordedAt: z.number().int(),
  recordedById: z.string().nullish(),
  documentId: z.string().nullish(),
  prevHash: z.string(),
  hash: z.string(),
  createdAt: z.number().int(),
});
export type DetentionEvidence = z.infer<typeof detentionEvidenceSchema>;

export const detentionNoticeSchema = z.object({
  id: z.string(),
  detentionOccurrenceId: z.string(),
  threadKey: z.string(),
  kind: noticeKindSchema,
  channel: noticeChannelSchema,
  deliveryStatus: noticeDeliveryStatusSchema,
  recipients: z.array(z.string()).nullish(),
  subject: z.string(),
  body: z.string(),
  scheduledFor: z.number().int(),
  sentAt: z.number().int().nullish(),
  deliveredAt: z.number().int().nullish(),
  openedAt: z.number().int().nullish(),
  failedAt: z.number().int().nullish(),
  failureReason: z.string().default(""),
  wasAutomatic: z.boolean().default(false),
  satisfiesRequirement: z.boolean().default(false),
  quotedFreeMinutes: z.number().int().default(0),
  createdAt: z.number().int(),
});
export type DetentionNotice = z.infer<typeof detentionNoticeSchema>;

export const detentionOccurrenceSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  shipmentId: z.string(),
  shipmentMoveId: z.string(),
  stopId: z.string(),
  customerId: z.string(),
  locationId: z.string(),
  detentionPolicyId: z.string().nullish(),
  policySnapshot: policySnapshotSchema.nullish(),
  calculationTrace: calculationTraceSchema.nullish(),
  stopType: z.string(),
  scheduleType: z.string(),
  appointmentStart: z.number().int().nullish(),
  appointmentEnd: z.number().int().nullish(),
  arrivedAt: z.number().int().nullish(),
  departedAt: z.number().int().nullish(),
  clockStartAt: z.number().int(),
  clockStopAt: z.number().int().nullish(),
  freeTimeExpiresAt: z.number().int(),
  noticeDueAt: z.number().int().nullish(),
  noticeDeadlineAt: z.number().int().nullish(),
  isOpen: z.boolean(),
  arrivedLate: z.boolean(),
  lateByMinutes: z.number().int(),
  freeMinutesGranted: z.number().int(),
  rawDwellMinutes: z.number().int(),
  billableMinutes: z.number().int(),
  roundedMinutes: z.number().int(),
  billableUnits: z.coerce.number(),
  grossAmount: z.coerce.number(),
  billableAmount: z.coerce.number(),
  driverPayMinutes: z.number().int(),
  driverPayAmount: z.coerce.number(),
  netMargin: z.coerce.number(),
  capApplied: capKindSchema,
  convertedToLayover: z.boolean(),
  currency: z.string(),
  status: occurrenceStatusSchema,
  notificationStatus: detentionNotificationStatusSchema,
  noticeSentAt: z.number().int().nullish(),
  suppressedByGate: z.boolean(),
  requiresApproval: z.boolean(),
  waiverReason: waiverReasonSchema.nullish(),
  waiverNote: z.string().default(""),
  waivedAt: z.number().int().nullish(),
  waivedAmount: z.coerce.number(),
  disputeNote: z.string().default(""),
  disputedAt: z.number().int().nullish(),
  collectabilityScore: z.number().int(),
  evidenceHead: z.string().default(""),
  additionalChargeId: z.string().nullish(),
  version: z.number().int(),
  createdAt: z.number().int(),
  updatedAt: z.number().int(),
});
export type DetentionOccurrence = z.infer<typeof detentionOccurrenceSchema>;

export const scoreFactorSchema = z.object({
  key: z.string(),
  label: z.string(),
  earned: z.number().int(),
  possible: z.number().int(),
  detail: z.string().default(""),
  remedy: z.string().default(""),
});
export type ScoreFactor = z.infer<typeof scoreFactorSchema>;

export const collectabilityAssessmentSchema = z.object({
  score: z.number().int(),
  band: scoreBandSchema,
  factors: z.array(scoreFactorSchema).default([]),
  chainValid: z.boolean(),
  summary: z.string().default(""),
});
export type CollectabilityAssessment = z.infer<typeof collectabilityAssessmentSchema>;

export const occurrenceDetailSchema = z.object({
  occurrence: detentionOccurrenceSchema,
  evidence: z.array(detentionEvidenceSchema).nullish(),
  notices: z.array(detentionNoticeSchema).nullish(),
  collectability: collectabilityAssessmentSchema,
  receipt: z.string().default(""),
});
export type OccurrenceDetail = z.infer<typeof occurrenceDetailSchema>;

export const deskUrgencySchema = z.enum([
  "Lost",
  "NoticeOverdue",
  "NoticeDueSoon",
  "Accruing",
  "FreeTimeEnding",
  "Normal",
]);
export type DeskUrgency = z.infer<typeof deskUrgencySchema>;

export const deskEntrySchema = z.object({
  occurrence: detentionOccurrenceSchema,
  minutesUntilFreeEnds: z.number().int(),
  minutesUntilNoticeDue: z.number().int().nullish(),
  noticeWindowOpen: z.boolean(),
  amountAtRisk: z.coerce.number(),
  urgency: deskUrgencySchema,
});
export type DeskEntry = z.infer<typeof deskEntrySchema>;

export const disputePacketSchema = z.object({
  occurrence: detentionOccurrenceSchema,
  policySnapshot: policySnapshotSchema.nullish(),
  receipt: z.string().default(""),
  evidence: z.array(detentionEvidenceSchema).nullish(),
  notices: z.array(detentionNoticeSchema).nullish(),
  collectability: collectabilityAssessmentSchema,
  chainVerified: z.boolean(),
  generatedAt: z.number().int(),
});
export type DisputePacket = z.infer<typeof disputePacketSchema>;

/** Ordered by how urgently a dispatcher must act, most urgent first. */
export const DESK_URGENCY_RANK: Record<DeskUrgency, number> = {
  NoticeOverdue: 0,
  NoticeDueSoon: 1,
  Accruing: 2,
  FreeTimeEnding: 3,
  Lost: 4,
  Normal: 5,
};

export const previewScenarioSchema = z.object({
  arrivedAt: z.number().int(),
  departedAt: z.number().int().nullish(),
  appointmentStart: z.number().int().nullish(),
  appointmentEnd: z.number().int().nullish(),
  stopType: z.string().default("Delivery"),
  scheduleType: z.string().default("Appointment"),
  noticeSentAt: z.number().int().nullish(),
  driverPayRate: z.string().nullish(),
});
export type PreviewScenario = z.infer<typeof previewScenarioSchema>;

export const previewResultSchema = z.object({
  policySnapshot: policySnapshotSchema,
  rawDwellMinutes: z.number().int(),
  freeMinutesGranted: z.number().int(),
  billableMinutes: z.number().int(),
  roundedMinutes: z.number().int(),
  billableAmount: z.coerce.number(),
  grossAmount: z.coerce.number(),
  driverPayAmount: z.coerce.number(),
  netMargin: z.coerce.number(),
  arrivedLate: z.boolean(),
  capApplied: capKindSchema,
  status: occurrenceStatusSchema,
  notificationStatus: detentionNotificationStatusSchema,
  suppressedByGate: z.boolean(),
  calculationTrace: calculationTraceSchema.nullish(),
  receipt: z.string().default(""),
});
export type PreviewResult = z.infer<typeof previewResultSchema>;
