import { Resource } from "@/types/audit-entry";
import {
  Channel,
  DeliveryStatus,
  EventType,
  Priority,
  UpdateType,
} from "@/types/notification";
import * as z from "zod/v4";
import {
  nullableTimestampSchema,
  optionalStringSchema,
  timestampSchema,
  versionSchema,
} from "./helpers";

export const targetingSchema = z.object({
  channel: z.enum(Channel),
  organizationId: z.string(),
  businessUnitId: z.string().optional().nullable(),
  targetUserId: z.string().optional().nullable(),
  targetRoleId: z.string().optional().nullable(),
});

const relatedEntitySchema = z.object({
  type: z.string(),
  id: z.string(),
  name: z.string().optional().nullable(),
  url: z.string().optional().nullable(),
});

const actionSchema = z.object({
  id: z.string(),
  label: z.string(),
  type: z.string(),
  style: z.string(),
  endpoint: z.string().optional().nullable(),
  payload: z.record(z.string(), z.any()).optional().nullable(),
});

export const notificationSchema = z.object({
  id: z.string(),

  eventType: z.enum(EventType),
  priority: z.enum(Priority),
  channel: z.enum(Channel),
  organizationId: z.string(),
  businessUnitId: z.string().optional().nullable(),
  targetUserId: z.string().optional().nullable(),
  targetRoleId: z.string().optional().nullable(),

  title: z.string(),
  message: z.string(),
  data: z.record(z.string(), z.any()),
  relatedEntities: z.array(relatedEntitySchema),
  actions: z.array(actionSchema),

  expiresAt: nullableTimestampSchema,
  deliveredAt: nullableTimestampSchema,
  readAt: nullableTimestampSchema,
  dismissedAt: nullableTimestampSchema,

  createdAt: z.number(),
  updatedAt: z.number(),

  deliveryStatus: z.enum(DeliveryStatus),
  retryCount: z.number(),
  maxRetries: z.number(),

  source: z.string(),
  jobId: z.string().optional().nullable(),
  correlationId: z.string().optional().nullable(),
  tags: z.array(z.string()),

  version: z.number(),
});

export const notificationPreferenceSchema = z.object({
  id: optionalStringSchema,
  version: versionSchema,
  createdAt: timestampSchema,
  updatedAt: timestampSchema,
  organizationId: optionalStringSchema,
  businessUnitId: optionalStringSchema,
  userId: z.string().min(1, { error: "User is required" }),

  resource: z.enum(Resource),
  updateTypes: z.array(z.enum(UpdateType)).optional(),
  notifyOnAllUpdates: z.boolean().default(false),
  notifyOnlyOwnedRecords: z.boolean().default(true),

  excludedUserIds: z.array(z.string()).optional(),
  includedRoleIds: z.array(z.string()).optional(),
  preferredChannels: z.array(z.enum(Channel)).optional(),

  quietHoursEnabled: z.boolean().default(false),
  quietHoursStart: optionalStringSchema,
  quietHoursEnd: optionalStringSchema,
  timezone: optionalStringSchema,

  batchNotifications: z.boolean().default(false),
  batchIntervalMinutes: z
    .number()
    .min(1, { error: "Batch interval must be greater than 0" })
    .max(1440, { error: "Batch interval must be less than 1440" })
    .default(15),

  isActive: z.boolean().default(true),
});

type NotificationSchema = z.infer<typeof notificationSchema>;
type ActionSchema = z.infer<typeof actionSchema>;
type RelatedEntitySchema = z.infer<typeof relatedEntitySchema>;
type TargetingSchema = z.infer<typeof targetingSchema>;
type NotificationPreferenceSchema = z.infer<
  typeof notificationPreferenceSchema
>;

export type {
  ActionSchema,
  NotificationPreferenceSchema,
  NotificationSchema,
  RelatedEntitySchema,
  TargetingSchema
};

