import { z } from "zod";
import { accessorialChargeMethodSchema, type AccessorialCharge } from "./accessorial-charge";
import { defaultBillTypeSchema } from "./bill-type";
import type { Commodity } from "./commodity";
import { customerSchema } from "./customer";
import { formulaReceiptSchema, formulaTemplateSchema } from "./formula-template";
import {
  decimalNumberSchema,
  decimalStringSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  optionalStringSchema,
  tenantInfoSchema,
} from "./helpers";
import { locationSchema } from "./location";
import { userSchema } from "./user";

export const shipmentStatusSchema = z.enum([
  "New",
  "PartiallyAssigned",
  "Assigned",
  "InTransit",
  "Delayed",
  "PartiallyCompleted",
  "Completed",
  "ReadyToInvoice",
  "Invoiced",
  "Canceled",
]);
export type ShipmentStatus = z.infer<typeof shipmentStatusSchema>;

export const shipmentTenderStatusSchema = z.enum([
  "Tendered",
  "Accepted",
  "Rejected",
  "Expired",
  "Canceled",
]);
export type ShipmentTenderStatus = z.infer<typeof shipmentTenderStatusSchema>;

export const shipmentEntryMethodSchema = z.enum(["Manual", "EDI"]);
export type ShipmentEntryMethod = z.infer<typeof shipmentEntryMethodSchema>;

export const shipmentBillingReadinessPolicySchema = z.object({
  shipmentBillingRequirementEnforcement: z.string(),
  rateValidationEnforcement: z.string(),
  billingExceptionDisposition: z.string(),
  notifyOnBillingExceptions: z.boolean(),
  readyToBillAssignmentMode: z.string(),
  billingQueueTransferMode: z.string(),
});

export const shipmentBillingValidationSchema = z.object({
  field: z.string(),
  code: z.string(),
  message: z.string(),
});

export const shipmentBillingWarningSchema = z.object({
  code: z.string(),
  message: z.string(),
  context: z.record(z.string(), z.unknown()).optional(),
});

export const shipmentServiceFailureBillingContextSchema = z.object({
  hasUnresolved: z.boolean(),
  unresolvedCount: z.number(),
  serviceFailureIds: z.array(z.string()),
});

export const shipmentBillingRequirementSchema = z.object({
  documentTypeId: z.string(),
  documentTypeCode: z.string(),
  documentTypeName: z.string(),
  satisfied: z.boolean(),
  documentCount: z.number(),
  documentIds: z.array(z.string()),
});

export const shipmentBillingReadinessSchema = z.object({
  shipmentId: z.string(),
  shipmentStatus: shipmentStatusSchema,
  policy: shipmentBillingReadinessPolicySchema,
  requirements: z.array(shipmentBillingRequirementSchema),
  missingRequirements: z.array(shipmentBillingRequirementSchema),
  validationFailures: z.array(shipmentBillingValidationSchema),
  warnings: z.array(shipmentBillingWarningSchema).default([]),
  serviceFailureContext: shipmentServiceFailureBillingContextSchema.default({
    hasUnresolved: false,
    unresolvedCount: 0,
    serviceFailureIds: [],
  }),
  canMarkReadyToInvoice: z.boolean(),
  shouldAutoMarkReadyToInvoice: z.boolean(),
  shouldAutoTransferToBilling: z.boolean(),
});

export type ShipmentBillingReadiness = z.infer<typeof shipmentBillingReadinessSchema>;
export type ShipmentBillingRequirement = z.infer<typeof shipmentBillingRequirementSchema>;
export type ShipmentBillingValidation = z.infer<typeof shipmentBillingValidationSchema>;
export type ShipmentBillingWarning = z.infer<typeof shipmentBillingWarningSchema>;
export type ShipmentServiceFailureBillingContext = z.infer<
  typeof shipmentServiceFailureBillingContextSchema
>;

export const moveStatusSchema = z.enum(["New", "Assigned", "InTransit", "Completed", "Canceled"]);
export type MoveStatus = z.infer<typeof moveStatusSchema>;

export const assignmentStatusSchema = z.enum(["New", "InProgress", "Completed", "Canceled"]);
export type AssignmentStatus = z.infer<typeof assignmentStatusSchema>;

export const stopStatusSchema = z.enum(["New", "InTransit", "Completed", "Canceled"]);
export type StopStatus = z.infer<typeof stopStatusSchema>;

export const stopTypeSchema = z.enum(["Pickup", "Delivery", "SplitDelivery", "SplitPickup"]);
export type StopType = z.infer<typeof stopTypeSchema>;

export const stopScheduleTypeSchema = z.enum(["Open", "Appointment"]);
export type StopScheduleType = z.infer<typeof stopScheduleTypeSchema>;

const stopBaseSchema = z.object({
  locationId: z.string().min(1, { error: "Location is required" }),
  status: stopStatusSchema.default("New"),
  type: stopTypeSchema.default("Pickup"),
  scheduleType: stopScheduleTypeSchema.default("Open"),
  sequence: z.number().int().nonnegative().default(0),
  pieces: nullableIntegerSchema,
  weight: nullableIntegerSchema,
  scheduledWindowStart: z.number().int().nonnegative().default(0),
  scheduledWindowEnd: nullableIntegerSchema,
  actualArrival: nullableIntegerSchema,
  actualDeparture: nullableIntegerSchema,
  countLateOverride: z.boolean().nullable().optional(),
  countDetentionOverride: z.boolean().nullable().optional(),
  addressLine: optionalStringSchema,
  location: locationSchema.nullish(),
});

const stopReadMetadataSchema = z.object({
  ...tenantInfoSchema.shape,
  shipmentMoveId: optionalStringSchema,
});

export const stopSchema = stopReadMetadataSchema.extend(stopBaseSchema.shape);
export type Stop = z.infer<typeof stopSchema>;

export const stopCreateSchema = stopBaseSchema;
export type StopCreateInput = z.infer<typeof stopCreateSchema>;

export const stopUpdateSchema = stopReadMetadataSchema.extend({
  ...stopBaseSchema.shape,
  id: optionalStringSchema,
});
export type StopUpdateInput = z.infer<typeof stopUpdateSchema>;

export const assignmentPayloadSchema = z.object({
  primaryWorkerId: z.string().min(1, { error: "Primary Worker is required" }),
  secondaryWorkerId: nullableStringSchema,
  tractorId: z.string().min(1, { error: "Tractor is required" }),
  trailerId: nullableStringSchema,
});
export type AssignmentPayload = z.infer<typeof assignmentPayloadSchema>;

const assignmentTractorSummarySchema = z.object({
  id: optionalStringSchema,
  code: optionalStringSchema,
});

const assignmentTrailerSummarySchema = z.object({
  id: optionalStringSchema,
  code: optionalStringSchema,
  equipmentTypeId: nullableStringSchema,
});

const assignmentWorkerSummarySchema = z.object({
  id: optionalStringSchema,
  firstName: optionalStringSchema,
  lastName: optionalStringSchema,
  wholeName: optionalStringSchema,
  profilePicUrl: nullableStringSchema,
});

export const assignmentSchema = z.object({
  id: optionalStringSchema,
  shipmentMoveId: optionalStringSchema,
  status: assignmentStatusSchema.default("New"),
  primaryWorkerId: nullableStringSchema,
  tractorId: nullableStringSchema,
  secondaryWorkerId: nullableStringSchema,
  trailerId: nullableStringSchema,
  tractor: assignmentTractorSummarySchema.nullish(),
  trailer: assignmentTrailerSummarySchema.nullish(),
  primaryWorker: assignmentWorkerSummarySchema.nullish(),
  secondaryWorker: assignmentWorkerSummarySchema.nullish(),
  version: z.number().optional(),
});
export type Assignment = z.infer<typeof assignmentSchema>;

export const moveCoverageTypeSchema = z.enum(["unassigned", "driver", "carrier"]);
export type MoveCoverageType = z.infer<typeof moveCoverageTypeSchema>;

export const carrierAssignmentStatusSchema = z.enum(["Pending", "Confirmed", "Canceled"]);
export type CarrierAssignmentStatus = z.infer<typeof carrierAssignmentStatusSchema>;

export const carrierRateMethodSchema = z.enum(["Flat", "PerMile"]);
export type CarrierRateMethod = z.infer<typeof carrierRateMethodSchema>;

const carrierAssignmentCarrierSummarySchema = z.object({
  id: optionalStringSchema,
  code: optionalStringSchema,
  name: optionalStringSchema,
  scac: nullableStringSchema,
});

export const carrierAssignmentAccessorialSchema = z.object({
  id: optionalStringSchema,
  carrierAssignmentId: optionalStringSchema,
  accessorialChargeId: nullableStringSchema,
  description: z.string(),
  amount: decimalStringSchema,
  version: z.number().optional(),
});
export type CarrierAssignmentAccessorial = z.infer<typeof carrierAssignmentAccessorialSchema>;

export const carrierAssignmentSchema = z.object({
  id: optionalStringSchema,
  shipmentMoveId: optionalStringSchema,
  carrierId: optionalStringSchema,
  status: carrierAssignmentStatusSchema.default("Pending"),
  rateMethod: carrierRateMethodSchema.default("Flat"),
  baseRate: decimalStringSchema,
  baseAmount: decimalStringSchema,
  fuelSurcharge: decimalStringSchema,
  accessorialTotal: decimalStringSchema,
  totalCost: decimalStringSchema,
  currencyCode: optionalStringSchema,
  proNumber: nullableStringSchema,
  externalDriverName: nullableStringSchema,
  externalDriverPhone: nullableStringSchema,
  externalTractorNumber: nullableStringSchema,
  externalTrailerNumber: nullableStringSchema,
  confirmedAt: z.number().nullish(),
  canceledAt: z.number().nullish(),
  cancellationReason: nullableStringSchema,
  carrier: carrierAssignmentCarrierSummarySchema.nullish(),
  accessorials: z.array(carrierAssignmentAccessorialSchema).nullish(),
  version: z.number().optional(),
});
export type CarrierAssignment = z.infer<typeof carrierAssignmentSchema>;

export function isActiveCarrierAssignment(
  carrierAssignment: CarrierAssignment | null | undefined,
): carrierAssignment is CarrierAssignment {
  return !!carrierAssignment?.id && carrierAssignment.status !== "Canceled";
}

const carrierAssignmentAccessorialPayloadSchema = z.object({
  accessorialChargeId: nullableStringSchema,
  description: z
    .string()
    .min(1, { error: "Description is required" })
    .max(255, { error: "Description must be at most 255 characters" }),
  amount: decimalNumberSchema("Amount is required", "Amount cannot be negative"),
});
export type CarrierAssignmentAccessorialPayload = z.infer<
  typeof carrierAssignmentAccessorialPayloadSchema
>;

export const carrierAssignmentPayloadSchema = z.object({
  carrierId: z.string().min(1, { error: "Carrier is required" }),
  rateMethod: carrierRateMethodSchema,
  baseRate: decimalNumberSchema("Base rate is required", "Base rate cannot be negative"),
  fuelSurcharge: decimalNumberSchema(
    "Fuel surcharge must be a valid number",
    "Fuel surcharge cannot be negative",
  ).nullish(),
  accessorials: z.array(carrierAssignmentAccessorialPayloadSchema),
  proNumber: z.string().max(50, { error: "Pro number must be at most 50 characters" }),
  externalDriverName: z.string().max(255, { error: "Driver name must be at most 255 characters" }),
  externalDriverPhone: z.string().max(20, { error: "Driver phone must be at most 20 characters" }),
  externalTractorNumber: z
    .string()
    .max(50, { error: "Tractor number must be at most 50 characters" }),
  externalTrailerNumber: z
    .string()
    .max(50, { error: "Trailer number must be at most 50 characters" }),
  overrideInsuranceWarning: z.boolean(),

  /**
   * Asks the carrier's contract to price this assignment instead of taking the
   * rate typed alongside it. A typed rate wins: that is a negotiated number,
   * and overruling it is what makes people switch auto-rating off.
   */
  autoRate: z.boolean().default(false),

  /**
   * Lets somebody assign a carrier the contract's margin terms would otherwise
   * refuse, the same escape the insurance warning has.
   */
  overrideMarginFloor: z.boolean().default(false),
});
export type CarrierAssignmentPayload = z.infer<typeof carrierAssignmentPayloadSchema>;
export type CarrierAssignmentPayloadInput = z.input<typeof carrierAssignmentPayloadSchema>;

export const emptyCarrierAssignmentPayload: CarrierAssignmentPayload = {
  carrierId: "",
  rateMethod: "Flat",
  baseRate: 0,
  fuelSurcharge: null,
  accessorials: [],
  proNumber: "",
  externalDriverName: "",
  externalDriverPhone: "",
  externalTractorNumber: "",
  externalTrailerNumber: "",
  overrideInsuranceWarning: false,
  autoRate: false,
  overrideMarginFloor: false,
};

export const carrierEligibilitySchema = z.object({
  blockers: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
  warnings: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
});
export type CarrierEligibility = z.infer<typeof carrierEligibilitySchema>;

const shipmentMoveBaseSchema = z.object({
  status: moveStatusSchema.default("New"),
  loaded: z.boolean().default(true),
  sequence: z.number().int().nonnegative().default(0),
  distance: z.number().nullable().optional(),
  distanceSource: z.string().nullish(),
  distanceProvider: z.string().nullish(),
  distanceCalculatedAt: z.number().nullable().optional(),
  distanceRouteSignature: z.string().nullish(),
  distanceDataVersion: z.string().nullish(),
  distanceRoutingType: z.string().nullish(),
  distanceUnits: z.string().nullish(),
  distanceMetadata: z.record(z.string(), z.unknown()).nullish(),
});

const shipmentMoveReadMetadataSchema = z.object({
  ...tenantInfoSchema.shape,
  shipmentId: optionalStringSchema,
});

export const shipmentMoveSchema = shipmentMoveReadMetadataSchema.extend({
  ...shipmentMoveBaseSchema.shape,
  id: optionalStringSchema,
  coverageType: moveCoverageTypeSchema.optional(),
  stops: z.array(stopSchema).default([]),
  assignment: assignmentSchema.nullish(),
  carrierAssignment: carrierAssignmentSchema.nullish(),
});
export type ShipmentMove = z.infer<typeof shipmentMoveSchema>;

export const shipmentMoveCreateSchema = shipmentMoveBaseSchema.extend({
  stops: z.array(stopCreateSchema).default([]),
});
export type ShipmentMoveCreateInput = z.infer<typeof shipmentMoveCreateSchema>;

export const shipmentMoveUpdateSchema = shipmentMoveReadMetadataSchema.extend({
  ...shipmentMoveBaseSchema.shape,
  id: optionalStringSchema,
  stops: z.array(stopUpdateSchema).default([]),
});
export type ShipmentMoveUpdateInput = z.infer<typeof shipmentMoveUpdateSchema>;

const additionalChargeBaseSchema = z.object({
  accessorialChargeId: z.string().min(1, { error: "Accessorial Charge is required" }),
  method: accessorialChargeMethodSchema.default("Flat"),
  amount: decimalStringSchema.default(0),
  unit: z.number().int().min(1, { error: "Unit must be at least 1" }).default(1),
});

export const fuelSurchargeDetailSchema = z.object({
  programId: z.string().optional(),
  programName: z.string().optional(),
  programCode: z.string().optional(),
  method: z.string().optional(),
  indexCode: z.string().optional(),
  indexSource: z.string().optional(),
  indexRegion: z.string().optional(),
  indexFuelType: z.string().optional(),
  eiaSeriesId: z.string().optional(),
  priceDate: z.string().optional(),
  price: z.number().optional(),
  currency: z.string().optional(),
  pegPrice: z.number().nullish(),
  increment: z.number().nullish(),
  incrementRate: z.number().nullish(),
  milesPerGallon: z.number().nullish(),
  bandMin: z.number().nullish(),
  bandMax: z.number().nullish(),
  bandValue: z.number().nullish(),
  miles: z.number().nullish(),
  ratePerMile: z.number().nullish(),
  percent: z.number().nullish(),
  percentBasis: z.string().nullish(),
  linehaulBase: z.number().nullish(),
  accessorialBase: z.number().nullish(),
  rawAmount: z.number().optional(),
  amount: z.number().optional(),
  capApplied: z.boolean().optional(),
  floorApplied: z.boolean().optional(),
  stepRounding: z.string().optional(),
  rateRounding: z.string().optional(),
  dateBasis: z.string().optional(),
  basisDate: z.string().optional(),
  usedFallback: z.boolean().optional(),
  stale: z.boolean().optional(),
  calculatedAt: z.number().optional(),
});
export type FuelSurchargeDetail = z.infer<typeof fuelSurchargeDetailSchema>;

export const additionalChargeSchema = z.object({
  ...tenantInfoSchema.shape,
  id: optionalStringSchema,
  isSystemGenerated: z.boolean().optional().default(false),
  ...additionalChargeBaseSchema.shape,
  accessorialCharge: z.custom<AccessorialCharge>().nullish(),
  fuelSurchargeProgramId: z.string().nullish(),
  fuelSurchargeDetail: fuelSurchargeDetailSchema.nullish(),
  detentionOccurrenceId: z.string().nullish(),
});
export type AdditionalCharge = z.infer<typeof additionalChargeSchema>;

export const additionalChargeCreateSchema = additionalChargeBaseSchema.extend({
  isSystemGenerated: z.boolean().optional().default(false),
});
export type AdditionalChargeCreateInput = z.infer<typeof additionalChargeCreateSchema>;

const shipmentCommodityBaseSchema = z.object({
  commodityId: z.string().min(1, { error: "Commodity is required" }),
  pieces: z
    .number({ error: "Pieces is required" })
    .int()
    .min(1, { error: "Pieces must be at least 1" })
    .default(1),
  weight: z
    .number({ error: "Weight is required" })
    .int()
    .nonnegative({ error: "Weight cannot be negative" })
    .default(0),
  lengthFeet: decimalStringSchema,
  widthFeet: decimalStringSchema,
  heightFeet: decimalStringSchema,
});

export const shipmentCommoditySchema = z.object({
  ...tenantInfoSchema.shape,
  id: optionalStringSchema,
  ...shipmentCommodityBaseSchema.shape,
  commodity: z.custom<Commodity>().nullish(),
});
export type ShipmentCommodity = z.infer<typeof shipmentCommoditySchema>;

export const shipmentCommodityCreateSchema = shipmentCommodityBaseSchema;
export type ShipmentCommodityCreateInput = z.infer<typeof shipmentCommodityCreateSchema>;

export const ratingBreakdownItemSchema = z.object({
  name: z.string(),
  label: z.string().optional().default(""),
  amount: z.number(),
  error: z.string().optional(),
});
export type RatingBreakdownItem = z.output<typeof ratingBreakdownItemSchema>;

export const ratingGuardrailSchema = z.object({
  applied: z.boolean(),
  bound: z.string().optional(),
  rawResult: z.number(),
  minCharge: z.number().nullish(),
  maxCharge: z.number().nullish(),
});
export type RatingGuardrail = z.output<typeof ratingGuardrailSchema>;

export const ratingDetailSchema = z.object({
  formulaTemplateId: z.string(),
  formulaTemplateName: z.string(),
  expression: z.string(),
  resolvedVariables: z.record(z.string(), z.any()),
  result: z.number(),
  ratedAt: z.number(),
  versionNumber: z.number().nullish(),
  breakdown: z.array(ratingBreakdownItemSchema).nullish(),
  guardrail: ratingGuardrailSchema.nullish(),

  // The contract that priced the shipment, when one did. These point at the
  // quote rather than duplicating it: the quote holds the full explanation, and
  // copying any of it here would let the two drift.
  rateQuoteId: z.string().nullish(),
  agreementId: z.string().nullish(),
  agreementName: z.string().nullish(),
  ruleId: z.string().nullish(),
  ruleLabel: z.string().nullish(),
  source: z.string().nullish(),
  explanation: z.string().nullish(),
  receipt: formulaReceiptSchema.nullish(),
});
export type RatingDetail = z.infer<typeof ratingDetailSchema>;

const shipmentBaseSchema = z.object({
  sourceDocumentId: nullableStringSchema,
  orderId: nullableStringSchema,
  orderNumber: nullableStringSchema,
  orderStatus: nullableStringSchema,
  serviceTypeId: z.string().min(1, { error: "Service Type is required" }),
  shipmentTypeId: z.string().min(1, { error: "Shipment Type is required" }),
  customerId: z.string().min(1, { error: "Customer is required" }),
  tractorTypeId: nullableStringSchema,
  trailerTypeId: nullableStringSchema,
  ownerId: nullableStringSchema,
  enteredById: nullableStringSchema,
  canceledById: nullableStringSchema,
  formulaTemplateId: nullableStringSchema,
  consolidationGroupId: nullableStringSchema,

  // Set by the rating engine, never by a caller. A payload that omitted them
  // would otherwise clear an override or unlock an invoiced shipment as a side
  // effect of saving something else, which is why the server restores them.
  rateQuoteId: nullableStringSchema,
  rateAgreementId: nullableStringSchema,
  rateAgreementRuleId: nullableStringSchema,
  rateOverrideAmount: decimalStringSchema,
  rateOverrideReason: z.string().nullish(),
  rateOverrideById: nullableStringSchema,
  rateOverrideAt: z.number().nullish(),
  rateLocked: z.boolean().nullish(),
  // True while the shipment still carries the rate its contract applied. It
  // goes false the moment somebody edits the rating method, the base rate or
  // one of the contract's own accessorial charges.
  autoRated: z.boolean().nullish(),
  autoRatedAt: z.number().nullish(),

  status: shipmentStatusSchema.default("New"),
  tenderStatus: shipmentTenderStatusSchema.nullable().optional(),
  entryMethod: shipmentEntryMethodSchema.optional(),
  proNumber: optionalStringSchema,
  bol: nullableStringSchema,
  cancelReason: optionalStringSchema,
  otherChargeAmount: decimalStringSchema.default(0),
  freightChargeAmount: decimalStringSchema.default(0),
  baseRate: decimalStringSchema.default(0),
  totalChargeAmount: decimalStringSchema.default(0),
  pieces: nullableIntegerSchema,
  weight: nullableIntegerSchema,
  envelopeLengthFeet: decimalStringSchema,
  envelopeWidthFeet: decimalStringSchema,
  envelopeHeightFeet: decimalStringSchema,
  envelopeOverallHeightFeet: decimalStringSchema,
  temperatureMin: nullableIntegerSchema,
  temperatureMax: nullableIntegerSchema,
  actualDeliveryDate: nullableIntegerSchema,
  actualShipDate: nullableIntegerSchema,
  canceledAt: nullableIntegerSchema,
  billingTransferStatus: nullableStringSchema,
  transferredToBillingAt: nullableIntegerSchema,
  markedReadyToBillAt: nullableIntegerSchema,
  billedAt: nullableIntegerSchema,
  ratingUnit: z.number().int().positive().default(1),
  fuelSurchargeLocked: z.boolean().default(false),
  ratingDetail: ratingDetailSchema.nullable().optional(),
});

/** One charge a contract applies automatically to every shipment it prices. */
export const contractRateAccessorialSchema = z.object({
  accessorialChargeId: z.string(),
  description: z.string(),
  method: accessorialChargeMethodSchema.default("Flat"),
  amount: decimalStringSchema,
  unit: z.number(),
});
export type ContractRateAccessorial = z.infer<typeof contractRateAccessorialSchema>;

/**
 * What the rate agreements charge for a shipment.
 *
 * The same shape answers both questions the billing panel asks — what would
 * this rate at, and what did the re-rate just apply — because a preview and an
 * application differ only in whether the numbers were kept.
 */
export const contractRateSchema = z.object({
  applied: z.boolean(),
  outcome: z.string(),
  agreementId: nullableStringSchema,
  agreementName: z.string(),
  ruleId: nullableStringSchema,
  ruleLabel: z.string(),
  formulaTemplateId: nullableStringSchema,
  formulaTemplateName: z.string(),
  baseRate: decimalStringSchema,
  linehaulAmount: decimalStringSchema,
  otherChargeAmount: decimalStringSchema,
  totalChargeAmount: decimalStringSchema,
  previousLinehaulAmount: decimalStringSchema,
  accessorials: z.array(contractRateAccessorialSchema).default([]),
  explanation: z.string(),
});
export type ContractRate = z.infer<typeof contractRateSchema>;

export const shipmentProfitabilityEstimateSchema = z.object({
  shipmentId: z.string(),
  loadedMiles: z.number(),
  deadheadMiles: z.number(),
  totalMiles: z.number(),
  costPerMile: z.string(),
  estimatedCost: z.string(),
  profit: z.string(),
  marginPercent: z.string().nullish(),
  breakEvenRpm: z.string().nullish(),
  targetMarginPercent: z.string().nullish(),
  missingDistance: z.boolean(),
});
export type ShipmentProfitabilityEstimate = z.infer<typeof shipmentProfitabilityEstimateSchema>;

export const shipmentSchema = z.object({
  ...tenantInfoSchema.shape,
  ...shipmentBaseSchema.shape,
  profitabilityEstimate: shipmentProfitabilityEstimateSchema.nullish(),
  moves: z.array(shipmentMoveSchema).default([]),
  additionalCharges: z.array(additionalChargeSchema).default([]),
  commodities: z.array(shipmentCommoditySchema).default([]),
  customer: customerSchema.nullish(),
  owner: userSchema.nullish(),
  formulaTemplate: formulaTemplateSchema.nullish(),
});

export type Shipment = z.infer<typeof shipmentSchema>;

export const shipmentCreateSchema = shipmentBaseSchema.extend({
  moves: z.array(shipmentMoveCreateSchema).default([]),
  additionalCharges: z.array(additionalChargeCreateSchema).default([]),
  commodities: z.array(shipmentCommodityCreateSchema).default([]),
});
export type ShipmentCreateInput = z.infer<typeof shipmentCreateSchema>;

export const shipmentUpdateSchema = z.object({
  ...tenantInfoSchema.shape,
  ...shipmentBaseSchema.shape,
  moves: z.array(shipmentMoveUpdateSchema).default([]),
  additionalCharges: z.array(additionalChargeSchema).default([]),
  commodities: z.array(shipmentCommoditySchema).default([]),
});
export type ShipmentUpdateInput = z.infer<typeof shipmentUpdateSchema>;

export const duplicateShipmentRequestSchema = z.object({
  shipmentId: z.string().min(1),
  count: z
    .number()
    .int()
    .min(1, { error: "Count must be at least 1" })
    .max(20, { error: "Count must be between 1 and 20" })
    .default(1),
  overrideDates: z.boolean().default(false),
});
export type DuplicateShipmentRequest = z.infer<typeof duplicateShipmentRequestSchema>;

export const duplicateShipmentResponseSchema = z.object({
  workflowId: z.string(),
  runId: z.string(),
  taskQueue: z.string(),
  status: z.string(),
  submittedAt: z.number(),
});
export type DuplicateShipmentResponse = z.infer<typeof duplicateShipmentResponseSchema>;

export type SplitStopTimes = {
  scheduledWindowStart: number;
  scheduledWindowEnd?: number | null;
};

export type SplitMovePayload = {
  newDeliveryLocationId: string;
  splitPickupTimes: SplitStopTimes;
  newDeliveryTimes: SplitStopTimes;
  pieces?: number;
  weight?: number;
};

export type SplitMoveResponse = {
  originalMove: ShipmentMove;
  newMove: ShipmentMove;
};

export type StopActualAction = "Arrive" | "Depart";

export type RecordStopActualPayload = {
  action: StopActualAction;
  occurredAt?: number;
};

export const transferOwnershipSchema = z.object({
  ownerId: z.string().min(1, { error: "Owner is required" }),
});
export type TransferOwnershipPayload = z.infer<typeof transferOwnershipSchema>;

export const shipmentTotalsFuelSurchargeSchema = z.object({
  accessorialChargeId: z.string(),
  isSystemGenerated: z.boolean().default(true),
  method: accessorialChargeMethodSchema.default("Flat"),
  amount: decimalStringSchema,
  unit: z.number().int().default(1),
  fuelSurchargeProgramId: z.string().nullish(),
  fuelSurchargeDetail: fuelSurchargeDetailSchema.nullish(),
});
export type ShipmentTotalsFuelSurcharge = z.infer<typeof shipmentTotalsFuelSurchargeSchema>;

export const shipmentTotalsResponseSchema = z.object({
  freightChargeAmount: decimalStringSchema,
  otherChargeAmount: decimalStringSchema,
  totalChargeAmount: decimalStringSchema,
  fuelSurcharge: shipmentTotalsFuelSurchargeSchema.nullish(),
});
export type ShipmentTotalsResponse = z.infer<typeof shipmentTotalsResponseSchema>;

export const shipmentDistanceMoveResultSchema = z.object({
  moveId: optionalStringSchema,
  moveIndex: z.number(),
  distance: z.number(),
  source: z.string(),
  provider: z.string().optional(),
  routingType: z.string().optional(),
  dataVersion: z.string().optional(),
  warnings: z.array(z.string()).optional(),
  calculatedAt: z.number(),
});

export const shipmentDistanceResponseSchema = z.object({
  shipmentId: optionalStringSchema,
  totalDistance: z.number(),
  moves: z.array(shipmentDistanceMoveResultSchema),
});

export type ShipmentDistanceResponse = z.infer<typeof shipmentDistanceResponseSchema>;

export const enforcementLevelSchema = z.enum(["Ignore", "Warn", "RequireReview", "Block"]);
export type EnforcementLevel = z.infer<typeof enforcementLevelSchema>;

export const capabilityProvenanceSchema = z.object({
  profileId: z.string(),
  profileCode: z.string(),
  profileName: z.string(),
  isOrgDefault: z.boolean(),
  priority: z.number().int(),
  specificityScore: z.number().int(),
  matchedOn: z.array(z.string()).nullish(),
  ruleKey: z.string(),
  capability: z.string(),
  capabilityLabel: z.string(),
  enforcement: enforcementLevelSchema,
  defaultEnforcement: enforcementLevelSchema,
  overridden: z.boolean(),
  overrideReason: z.string().nullish(),
  rationale: z.string(),
});
export type CapabilityProvenance = z.infer<typeof capabilityProvenanceSchema>;

export const resolvedCapabilityRuleSchema = z.object({
  key: z.string(),
  capability: z.string(),
  label: z.string(),
  enforcement: enforcementLevelSchema,
  enabled: z.boolean(),
  fields: z.array(z.string()).nullish(),
  requiredFields: z.array(z.string()).nullish(),
  parameters: z.record(z.string(), z.unknown()).nullish(),
  provenance: capabilityProvenanceSchema,
});
export type ResolvedCapabilityRule = z.infer<typeof resolvedCapabilityRuleSchema>;

export const profileCandidateSchema = z.object({
  profileId: z.string(),
  profileCode: z.string(),
  profileName: z.string(),
  priority: z.number().int(),
  specificityScore: z.number().int(),
  matchedOn: z.array(z.string()).nullish(),
  selected: z.boolean(),
  rejectionReason: z.string().nullish(),
});
export type ProfileCandidate = z.infer<typeof profileCandidateSchema>;

export const resolvedModeProfileSchema = z.object({
  profileId: z.string(),
  profileCode: z.string(),
  profileName: z.string(),
  serviceModel: z.string(),
  equipmentClass: z.string(),
  executionParty: z.string(),
  capabilities: z.array(z.string()).nullish(),
  rules: z.record(z.string(), resolvedCapabilityRuleSchema),
  candidates: z.array(profileCandidateSchema).nullish(),
  resolvedAt: z.number().int(),
});
export type ResolvedModeProfile = z.infer<typeof resolvedModeProfileSchema>;

export const shipmentUIPolicySchema = z.object({
  allowMoveRemovals: z.boolean(),
  checkForDuplicateBols: z.boolean(),
  checkHazmatSegregation: z.boolean(),
  maxShipmentWeightLimit: z.number().int().nonnegative(),
  profile: resolvedModeProfileSchema.nullish(),
});
export type ShipmentUIPolicy = z.infer<typeof shipmentUIPolicySchema>;

export const previousRateSummarySchema = z.object({
  shipmentId: z.string(),
  proNumber: z.string(),
  customerId: z.string(),
  serviceTypeId: z.string(),
  shipmentTypeId: z.string(),
  formulaTemplateId: z.string(),
  freightChargeAmount: decimalStringSchema,
  otherChargeAmount: decimalStringSchema,
  totalChargeAmount: decimalStringSchema,
  ratingUnit: z.number(),
  pieces: z.number().nullable(),
  weight: z.number().nullable(),
  createdAt: z.number(),
});
export type PreviousRateSummary = z.infer<typeof previousRateSummarySchema>;

export const previousRatesResponseSchema = z.object({
  items: z.array(previousRateSummarySchema),
  total: z.number(),
});
export type PreviousRatesResponse = z.infer<typeof previousRatesResponseSchema>;

export const getPreviousRatesRequestSchema = z.object({
  originLocationId: z.string().min(1, { error: "Origin Location is required" }),
  destinationLocationId: z.string().min(1, { error: "Destination Location is required" }),
  shipmentTypeId: z.string().min(1, { error: "Shipment Type is required" }),
  serviceTypeId: z.string().min(1, { error: "Service Type is required" }),
  customerId: optionalStringSchema,
  excludeShipmentId: optionalStringSchema,
});

export type GetPreviousRatesRequest = z.infer<typeof getPreviousRatesRequestSchema>;

const bulkTransferToBillingResultSchema = z.object({
  shipmentId: z.string(),
  success: z.boolean(),
  error: optionalStringSchema,
});

export type BulkTransferToBillingResult = z.infer<typeof bulkTransferToBillingResultSchema>;

export const bulkTransferToBillingResponseSchema = z.object({
  results: z.array(bulkTransferToBillingResultSchema),
  totalCount: z.number(),
  successCount: z.number(),
  errorCount: z.number(),
});

export type BulkTransferToBillingResponse = z.infer<typeof bulkTransferToBillingResponseSchema>;

export const bulkTransferToBillingRequestSchema = z.object({
  shipmentIds: z.array(z.string()).min(1, { error: "At least one Shipment ID is required" }),
  billType: defaultBillTypeSchema,
});

export type BulkTransferToBillingRequest = z.infer<typeof bulkTransferToBillingRequestSchema>;

export const transferToBillingRequestSchema = z.object({
  shipmentId: z.string().min(1, { error: "Shipment ID is required" }),
  billType: defaultBillTypeSchema,
});

export type TransferToBillingRequest = z.infer<typeof transferToBillingRequestSchema>;

export const listShipmentCommentRequestSchema = z.object({
  shipmentId: shipmentSchema.shape.id,
});
