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

export const serviceFailureTypeSchema = z.enum(["LatePickup", "LateDelivery"]);
export type ServiceFailureType = z.infer<typeof serviceFailureTypeSchema>;

export const serviceFailureSourceSchema = z.enum(["Detected", "Manual"]);
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

export const serviceFailureManualCreateSchema = z.object({
  shipmentId: z.string().min(1, { message: "Shipment is required" }),
  shipmentMoveId: z.string().min(1, { message: "Move is required" }),
  stopId: z.string().min(1, { message: "Stop is required" }),
  reasonCodeId: z.string().min(1, { message: "Reason code is required" }),
  type: serviceFailureTypeSchema,
  notes: nullableTextSchema,
  internalNotes: nullableTextSchema,
  x12StatusCodeOverride: optionalStringSchema.default(""),
  x12ReasonCodeOverride: optionalStringSchema.default(""),
  x12ExceptionCode: optionalStringSchema.default(""),
  scheduledCutoff: z.number().int().positive().nullable().optional(),
  actualArrival: z.number().int().positive().nullable().optional(),
  gracePeriodMinutes: z.number().int().positive().nullable().optional(),
  lateMinutes: z.number().int().positive().nullable().optional(),
});

export type ServiceFailureManualCreate = z.infer<typeof serviceFailureManualCreateSchema>;

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
  notes?: string;
  version: number;
};

export const serviceFailureEvaluationResultSchema = z.object({
  createdIds: z.array(z.string()).default([]),
  updatedIds: z.array(z.string()).default([]),
  skipped: z.number().int().default(0),
});

export type ServiceFailureEvaluationResult = z.infer<typeof serviceFailureEvaluationResultSchema>;

export const serviceFailureEdiPayloadResultSchema = z.object({
  payload: z.record(z.string(), z.unknown()),
  diagnostics: z.array(z.unknown()).default([]),
});

export type ServiceFailureEdiPayloadResult = z.infer<typeof serviceFailureEdiPayloadResultSchema>;
