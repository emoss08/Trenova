import { z } from "zod";
import { notificationPrioritySchema, type NotificationPriority } from "./notification";
import { createLimitOffsetResponse } from "./server";

export { notificationPrioritySchema, type NotificationPriority };

export const subscriptionStatusSchema = z.enum(["Active", "Paused"]);
export type SubscriptionStatus = z.infer<typeof subscriptionStatusSchema>;

export const eventTypeSchema = z.enum(["INSERT", "UPDATE", "DELETE"]);
export type EventType = z.infer<typeof eventTypeSchema>;

export const conditionOperatorSchema = z.enum([
  "eq",
  "neq",
  "gt",
  "gte",
  "lt",
  "lte",
  "is_null",
  "is_not_null",
  "contains",
  "not_contains",
  "changed_to",
  "changed_from",
  "changed",
]);
export type ConditionOperator = z.infer<typeof conditionOperatorSchema>;

export const conditionSchema = z.object({
  field: z.string().min(1, "Field is required"),
  operator: conditionOperatorSchema,
  value: z.union([z.string(), z.number(), z.null()]).optional(),
});
export type Condition = z.infer<typeof conditionSchema>;

export const conditionMatchSchema = z.enum(["all", "any"]);
export type ConditionMatch = z.infer<typeof conditionMatchSchema>;

export const tcaAllowlistedTableSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  tableName: z.string(),
  displayName: z.string(),
  enabled: z.boolean(),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export type TCAAllowlistedTable = z.infer<typeof tcaAllowlistedTableSchema>;

export const tcaSubscriptionSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string(),
  userId: z.string(),
  name: z.string(),
  tableName: z.string(),
  recordId: z.string().nullable(),
  eventTypes: z.array(z.string()),
  conditions: z.array(conditionSchema).default([]),
  conditionMatch: conditionMatchSchema.default("all"),
  watchedColumns: z.array(z.string()).default([]),
  customTitle: z.string().max(500).default(""),
  customMessage: z.string().max(5000).default(""),
  topic: z.string().max(100).default(""),
  priority: notificationPrioritySchema.default("medium"),
  status: subscriptionStatusSchema,
  version: z.number(),
  createdAt: z.number(),
  updatedAt: z.number(),
});

export type TCASubscription = z.infer<typeof tcaSubscriptionSchema>;

export const tcaSubscriptionFormSchema = z.object({
  name: z.string().min(1, "Name is required").max(255),
  tableName: z.string().min(1, "Table is required"),
  recordId: z.string().optional(),
  eventTypes: z.array(z.string()).min(1, "At least one event type is required"),
  conditions: z.array(conditionSchema).default([]),
  conditionMatch: conditionMatchSchema.default("all"),
  watchedColumns: z.array(z.string()).default([]),
  customTitle: z.string().max(500).default(""),
  customMessage: z.string().max(5000).default(""),
  topic: z.string().max(100).default(""),
  priority: notificationPrioritySchema.default("medium"),
  status: subscriptionStatusSchema.default("Active"),
});

export type TCASubscriptionFormValues = z.input<
  typeof tcaSubscriptionFormSchema
>;

export const tcaSubscriptionResponseSchema = createLimitOffsetResponse(
  tcaSubscriptionSchema,
);

export type TCASubscriptionResponse = z.infer<
  typeof tcaSubscriptionResponseSchema
>;

