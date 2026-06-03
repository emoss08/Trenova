import { z } from "zod";
import {
  nullableTextSchema,
  nullableIntegerSchema,
  optionalStringSchema,
  tenantInfoSchema,
} from "./helpers";
import { stopTypeSchema } from "./shipment";
import { serviceFailureReasonCodeSchema } from "./service-failure-reason-code";

const nullableTimestampSchema = z.number().int().nullable().optional();
const evaluationResultIdsSchema = z
  .array(z.string())
  .nullish()
  .transform((value) => value ?? []);
const serviceFailureEvaluationStopSummarySchema = z
  .object({
    shipmentId: optionalStringSchema,
    shipmentMoveId: optionalStringSchema,
    stopId: optionalStringSchema,
    stopSequence: nullableIntegerSchema,
    stopType: stopTypeSchema.optional(),
    locationId: optionalStringSchema,
    locationName: optionalStringSchema,
    locationCode: optionalStringSchema,
    city: optionalStringSchema,
    stateCode: optionalStringSchema,
    scheduledCutoff: nullableIntegerSchema,
    actualArrival: nullableIntegerSchema,
    gracePeriodMinutes: nullableIntegerSchema,
    lateMinutes: nullableIntegerSchema,
    serviceFailureId: optionalStringSchema,
    reason: optionalStringSchema,
  })
  .passthrough();
const evaluationStopsSchema = z
  .array(serviceFailureEvaluationStopSummarySchema)
  .nullish()
  .transform((value) => value ?? []);

export const serviceFailureTypeSchema = z.enum([
  "LatePickup",
  "LateDelivery",
  "MissedPickup",
  "MissedDelivery",
  "AppointmentMissed",
  "Other",
]);
export type ServiceFailureType = z.infer<typeof serviceFailureTypeSchema>;

export const serviceFailureSourceSchema = z.enum(["Detected", "Manual", "EDI", "Integration"]);
export type ServiceFailureSource = z.infer<typeof serviceFailureSourceSchema>;

export const serviceFailureStatusSchema = z.enum(["Open", "Reviewed", "Resolved", "Voided"]);
export type ServiceFailureStatus = z.infer<typeof serviceFailureStatusSchema>;

const serviceFailureShipmentSummarySchema = z
  .object({
    id: optionalStringSchema,
    proNumber: optionalStringSchema,
    bol: optionalStringSchema,
    status: optionalStringSchema,
  })
  .passthrough();

const serviceFailureStopSummarySchema = z
  .object({
    id: optionalStringSchema,
    type: stopTypeSchema.optional(),
    sequence: nullableIntegerSchema,
    locationId: optionalStringSchema,
    location: z
      .object({
        id: optionalStringSchema,
        name: optionalStringSchema,
        code: optionalStringSchema,
        city: optionalStringSchema,
        state: z
          .object({
            abbreviation: optionalStringSchema,
          })
          .passthrough()
          .nullish(),
      })
      .passthrough()
      .nullish(),
  })
  .passthrough();

export const serviceFailureSchema = z.object({
  ...tenantInfoSchema.shape,
  shipmentId: z.string(),
  shipmentMoveId: z.string(),
  stopId: z.string(),
  reasonCodeId: z.string().nullable().optional(),
  number: z.string(),
  type: serviceFailureTypeSchema,
  source: serviceFailureSourceSchema,
  status: serviceFailureStatusSchema,
  stopType: stopTypeSchema,
  scheduledCutoff: z.number().int(),
  actualArrival: z.number().int(),
  gracePeriodMinutes: z.number().int(),
  lateMinutes: z.number().int(),
  notes: nullableTextSchema,
  internalNotes: nullableTextSchema,
  x12StatusCodeOverride: optionalStringSchema.default(""),
  x12ReasonCodeOverride: optionalStringSchema.default(""),
  x12ExceptionCode: optionalStringSchema.default(""),
  detectedAt: z.number().int(),
  reviewedAt: nullableTimestampSchema,
  reviewedById: z.string().nullable().optional(),
  resolvedAt: nullableTimestampSchema,
  resolvedById: z.string().nullable().optional(),
  voidedAt: nullableTimestampSchema,
  voidedById: z.string().nullable().optional(),
  voidReason: nullableTextSchema,
  createdById: z.string().nullable().optional(),
  reasonCode: serviceFailureReasonCodeSchema.nullish(),
  shipment: serviceFailureShipmentSummarySchema.nullish(),
  stop: serviceFailureStopSummarySchema.nullish(),
});

export type ServiceFailure = z.infer<typeof serviceFailureSchema>;

export const serviceFailureUpdateSchema = z.object({
  id: z.string(),
  shipmentId: z.string(),
  reasonCodeId: z.string().optional(),
  clearReasonCode: z.boolean().default(false),
  notes: nullableTextSchema,
  internalNotes: nullableTextSchema,
  x12StatusCodeOverride: optionalStringSchema.default(""),
  x12ReasonCodeOverride: optionalStringSchema.default(""),
  x12ExceptionCode: optionalStringSchema.default(""),
  version: z.number().int().default(0),
});

export type ServiceFailureUpdate = z.infer<typeof serviceFailureUpdateSchema>;

export type ServiceFailureLifecycleRequest = {
  shipmentId: string;
  reasonCodeId?: string;
  notes?: string;
  version: number;
};

export const serviceFailureEvaluationResultSchema = z.object({
  createdIds: evaluationResultIdsSchema,
  updatedIds: evaluationResultIdsSchema,
  createdStops: evaluationStopsSchema,
  updatedStops: evaluationStopsSchema,
  skippedStops: evaluationStopsSchema,
  skipped: z
    .number()
    .int()
    .nullish()
    .transform((value) => value ?? 0),
});

export type ServiceFailureEvaluationResult = z.infer<typeof serviceFailureEvaluationResultSchema>;
export type ServiceFailureStopSummary = z.infer<typeof serviceFailureEvaluationStopSummarySchema>;

export const serviceFailureEdiPayloadResultSchema = z.object({
  payload: z.record(z.string(), z.unknown()),
  diagnostics: z.array(z.unknown()).default([]),
});

export type ServiceFailureEdiPayloadResult = z.infer<typeof serviceFailureEdiPayloadResultSchema>;

export const serviceFailureEdi214LifecycleActionSchema = z.enum([
  "skipped",
  "generated",
  "blocked",
  "duplicate",
]);

export const serviceFailureEdi214LifecycleResultSchema = z.object({
  trigger: z.enum(["Reviewed", "Resolved"]),
  action: serviceFailureEdi214LifecycleActionSchema,
  messageId: optionalStringSchema,
  skippedReason: optionalStringSchema,
  ediPartnerId: optionalStringSchema,
  partnerDocumentProfileId: optionalStringSchema,
  mandatory: z.boolean().default(false),
  diagnostics: z.array(z.unknown()).default([]),
});

export type ServiceFailureEdi214LifecycleResult = z.infer<
  typeof serviceFailureEdi214LifecycleResultSchema
>;

export const serviceFailureEdi214StatusSchema = z.object({
  serviceFailureId: optionalStringSchema,
  reviewedMessageId: optionalStringSchema,
  resolvedMessageId: optionalStringSchema,
  lastMessageId: optionalStringSchema,
  generatedStatus: optionalStringSchema,
  deliveryStatus: optionalStringSchema,
  ackStatus: optionalStringSchema,
  lastDiagnostic: optionalStringSchema,
  lastGeneratedAt: z.number().int().default(0),
});

export type ServiceFailureEdi214Status = z.infer<typeof serviceFailureEdi214StatusSchema>;
