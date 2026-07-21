import { z } from "zod";
import {
  nullableIntegerSchema,
  nullableStringSchema,
  optionalStringSchema,
  tenantInfoSchema,
} from "./helpers";

export const recurringShipmentStatusSchema = z.enum(["Active", "Paused", "Expired"]);

export type RecurringShipmentStatus = z.infer<typeof recurringShipmentStatusSchema>;

export const recurringShipmentExceptionPolicySchema = z.enum([
  "Skip",
  "PreviousBusinessDay",
  "NextBusinessDay",
]);

export type RecurringShipmentExceptionPolicy = z.infer<
  typeof recurringShipmentExceptionPolicySchema
>;

const laneEntitySchema = z
  .object({
    id: z.string(),
    name: z.string().optional(),
    code: z.string().optional(),
  })
  .loose();

export const recurringShipmentSchema = z.object({
  ...tenantInfoSchema.shape,
  sourceShipmentId: z.string().min(1, { error: "Source shipment is required" }),
  customerId: optionalStringSchema,
  originLocationId: optionalStringSchema,
  destinationLocationId: optionalStringSchema,
  enteredById: optionalStringSchema,
  lastGeneratedShipmentId: optionalStringSchema,
  name: z.string().min(1, { error: "Name is required" }).max(100),
  description: optionalStringSchema,
  status: recurringShipmentStatusSchema,
  cronExpression: z.string().min(1, { error: "A schedule is required" }),
  timezone: z.string().min(1, { error: "Timezone is required" }),
  startDate: nullableIntegerSchema,
  endDate: nullableIntegerSchema,
  maxOccurrences: nullableIntegerSchema,
  leadTimeDays: z.number().int().min(0).max(60),
  skipWeekends: z.boolean(),
  exceptionPolicy: recurringShipmentExceptionPolicySchema,
  blackoutDates: z.array(z.string()).nullish(),
  autoGenerate: z.boolean(),
  nextOccurrenceAt: nullableIntegerSchema,
  nextOccurrenceSourceAt: nullableIntegerSchema,
  lastOccurrenceAt: nullableIntegerSchema,
  lastRunAt: nullableIntegerSchema,
  generationCount: z.number().optional(),
  consecutiveFailures: z.number().optional(),
  customer: laneEntitySchema.nullish(),
  originLocation: laneEntitySchema.nullish(),
  destinationLocation: laneEntitySchema.nullish(),
  sourceShipment: z
    .object({
      id: z.string(),
      proNumber: z.string().optional(),
      bol: nullableStringSchema,
    })
    .loose()
    .nullish(),
});

export type RecurringShipment = z.infer<typeof recurringShipmentSchema>;

export const recurringShipmentRunStatusSchema = z.enum(["Generated", "Skipped", "Failed"]);

export type RecurringShipmentRunStatus = z.infer<typeof recurringShipmentRunStatusSchema>;

export const recurringShipmentRunSchema = z.object({
  id: z.string(),
  businessUnitId: z.string(),
  organizationId: z.string(),
  recurringShipmentId: z.string(),
  generatedShipmentId: optionalStringSchema,
  triggeredById: optionalStringSchema,
  status: recurringShipmentRunStatusSchema,
  trigger: z.enum(["Auto", "Manual"]),
  occurrenceAt: z.number(),
  originalOccurrenceAt: nullableIntegerSchema,
  detail: optionalStringSchema,
  createdAt: z.number(),
  generatedShipment: z
    .object({
      id: z.string(),
      proNumber: z.string().optional(),
      status: z.string().optional(),
    })
    .loose()
    .nullish(),
  triggeredBy: z
    .object({
      id: z.string(),
      name: z.string().optional(),
    })
    .loose()
    .nullish(),
});

export type RecurringShipmentRun = z.infer<typeof recurringShipmentRunSchema>;

export const lanePatternSummarySchema = z.object({
  shipmentCount: z.number(),
  firstShipmentAt: z.number(),
  lastShipmentAt: z.number(),
});

export type LanePatternSummary = z.infer<typeof lanePatternSummarySchema>;

export const matchRecurringShipmentsResponseSchema = z.object({
  matches: z.array(recurringShipmentSchema),
  pattern: lanePatternSummarySchema.nullish(),
});

export type MatchRecurringShipmentsResponse = z.infer<typeof matchRecurringShipmentsResponseSchema>;

export const generateRecurringShipmentResultSchema = z
  .object({
    series: recurringShipmentSchema.nullish(),
    run: recurringShipmentRunSchema.nullish(),
    shipment: z
      .object({
        id: z.string(),
        proNumber: z.string().optional(),
      })
      .loose()
      .nullish(),
  })
  .loose();

export type GenerateRecurringShipmentResult = z.infer<typeof generateRecurringShipmentResultSchema>;
