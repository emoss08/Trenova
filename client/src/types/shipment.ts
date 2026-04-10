import { z } from "zod";
import { accessorialChargeMethodSchema, type AccessorialCharge } from "./accessorial-charge";
import type { Commodity } from "./commodity";
import { customerSchema } from "./customer";
import { formulaTemplateSchema } from "./formula-template";
import {
  decimalStringSchema,
  nullableIntegerSchema,
  nullableStringSchema,
  optionalStringSchema,
  tenantInfoSchema,
} from "./helpers";
import { locationSchema } from "./location";
import { tractorSchema } from "./tractor";
import { trailerSchema } from "./trailer";
import { userSchema } from "./user";
import { workerSchema } from "./worker";

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
  canMarkReadyToInvoice: z.boolean(),
  shouldAutoMarkReadyToInvoice: z.boolean(),
  shouldAutoTransferToBilling: z.boolean(),
});

export type ShipmentBillingReadiness = z.infer<typeof shipmentBillingReadinessSchema>;
export type ShipmentBillingRequirement = z.infer<typeof shipmentBillingRequirementSchema>;
export type ShipmentBillingValidation = z.infer<typeof shipmentBillingValidationSchema>;

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

export const assignmentSchema = z.object({
  id: optionalStringSchema,
  shipmentMoveId: optionalStringSchema,
  status: assignmentStatusSchema.default("New"),
  primaryWorkerId: nullableStringSchema,
  tractorId: nullableStringSchema,
  secondaryWorkerId: nullableStringSchema,
  trailerId: nullableStringSchema,
  tractor: tractorSchema.nullish(),
  trailer: trailerSchema.nullish(),
  primaryWorker: workerSchema.nullish(),
  secondaryWorker: workerSchema.nullish(),
  version: z.number().optional(),
});
export type Assignment = z.infer<typeof assignmentSchema>;

const shipmentMoveBaseSchema = z.object({
  status: moveStatusSchema.default("New"),
  loaded: z.boolean().default(true),
  sequence: z.number().int().nonnegative().default(0),
  distance: z.number().nullable().optional(),
});

const shipmentMoveReadMetadataSchema = z.object({
  ...tenantInfoSchema.shape,
  shipmentId: optionalStringSchema,
});

export const shipmentMoveSchema = shipmentMoveReadMetadataSchema.extend({
  ...shipmentMoveBaseSchema.shape,
  id: optionalStringSchema,
  stops: z.array(stopSchema).default([]),
  assignment: assignmentSchema.nullish(),
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

export const additionalChargeSchema = z.object({
  ...tenantInfoSchema.shape,
  id: optionalStringSchema,
  isSystemGenerated: z.boolean().optional().default(false),
  ...additionalChargeBaseSchema.shape,
  accessorialCharge: z.custom<AccessorialCharge>().nullish(),
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

export const ratingDetailSchema = z.object({
  formulaTemplateId: z.string(),
  formulaTemplateName: z.string(),
  expression: z.string(),
  resolvedVariables: z.record(z.string(), z.any()),
  result: z.number(),
  ratedAt: z.number(),
});
export type RatingDetail = z.infer<typeof ratingDetailSchema>;

const shipmentBaseSchema = z.object({
  sourceDocumentId: z.string().optional(),
  serviceTypeId: z.string().min(1, { error: "Service Type is required" }),
  shipmentTypeId: z.string().min(1, { error: "Shipment Type is required" }),
  customerId: z.string().min(1, { error: "Customer is required" }),
  tractorTypeId: nullableStringSchema,
  trailerTypeId: nullableStringSchema,
  ownerId: nullableStringSchema,
  enteredById: nullableStringSchema,
  canceledById: nullableStringSchema,
  formulaTemplateId: z.string().min(1, { error: "Formula Template is required" }),
  consolidationGroupId: nullableStringSchema,
  status: shipmentStatusSchema.default("New"),
  proNumber: optionalStringSchema,
  bol: nullableStringSchema,
  cancelReason: optionalStringSchema,
  otherChargeAmount: decimalStringSchema.default(0),
  freightChargeAmount: decimalStringSchema.default(0),
  baseRate: decimalStringSchema.default(0),
  totalChargeAmount: decimalStringSchema.default(0),
  pieces: nullableIntegerSchema,
  weight: nullableIntegerSchema,
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
  ratingDetail: ratingDetailSchema.nullable().optional(),
});

export const shipmentSchema = z.object({
  ...tenantInfoSchema.shape,
  ...shipmentBaseSchema.shape,
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

export const transferOwnershipSchema = z.object({
  ownerId: z.string().min(1, { error: "Owner is required" }),
});
export type TransferOwnershipPayload = z.infer<typeof transferOwnershipSchema>;

export const shipmentTotalsResponseSchema = z.object({
  freightChargeAmount: decimalStringSchema,
  otherChargeAmount: decimalStringSchema,
  totalChargeAmount: decimalStringSchema,
});
export type ShipmentTotalsResponse = z.infer<typeof shipmentTotalsResponseSchema>;

export const shipmentUIPolicySchema = z.object({
  allowMoveRemovals: z.boolean(),
  checkForDuplicateBols: z.boolean(),
  checkHazmatSegregation: z.boolean(),
  maxShipmentWeightLimit: z.number().int().nonnegative(),
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

export type GetPreviousRatesRequest = {
  originLocationId: string;
  destinationLocationId: string;
  shipmentTypeId: string;
  serviceTypeId: string;
  customerId?: string;
  excludeShipmentId?: string;
};

export type BulkTransferToBillingResult = {
  shipmentId: string;
  success: boolean;
  error?: string;
};

export type BulkTransferToBillingResponse = {
  results: BulkTransferToBillingResult[];
  totalCount: number;
  successCount: number;
  errorCount: number;
};
