import { z } from "zod";
import {
  nullableTextSchema,
  nullableIntegerSchema,
  optionalStringSchema,
} from "@trenova/shared/types/helpers";
import { stopTypeSchema } from "@trenova/shared/types/shipment";

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
  "queued",
]);

export const serviceFailureEdi214LifecycleResultSchema = z.object({
  trigger: z.enum(["Reviewed", "Resolved"]),
  action: serviceFailureEdi214LifecycleActionSchema,
  messageId: optionalStringSchema,
  skippedReason: optionalStringSchema,
  ediPartnerId: optionalStringSchema,
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
