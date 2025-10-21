/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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
  businessUnitId: z.string().nullish(),
  targetUserId: z.string().nullish(),
  targetRoleId: z.string().nullish(),
});

const relatedEntitySchema = z.object({
  type: z.string(),
  id: z.string(),
  name: z.string().nullish(),
  url: z.string().nullish(),
});

const actionSchema = z.object({
  id: z.string(),
  label: z.string(),
  type: z.string(),
  style: z.string(),
  endpoint: z.string().nullish(),
  payload: z.record(z.string(), z.any()).nullish(),
});

export const notificationSchema = z.object({
  id: z.string(),

  eventType: z.enum(EventType),
  priority: z.enum(Priority),
  channel: z.enum(Channel),
  organizationId: z.string(),
  businessUnitId: z.string().nullish(),
  targetUserId: z.string().nullish(),
  targetRoleId: z.string().nullish(),

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
  jobId: z.string().nullish(),
  correlationId: z.string().nullish(),
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
  TargetingSchema,
};
