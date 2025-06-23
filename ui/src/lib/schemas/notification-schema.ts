import { Resource } from "@/types/audit-entry";
import {
  Channel,
  DeliveryStatus,
  EventType,
  Priority,
  UpdateType,
} from "@/types/notification";
import { z } from "zod";

export const targetingSchema = z.object({
  channel: z.nativeEnum(Channel),
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
  payload: z.record(z.any()).optional().nullable(),
});

export const notificationSchema = z.object({
  id: z.string(),

  eventType: z.nativeEnum(EventType),
  priority: z.nativeEnum(Priority),
  channel: z.nativeEnum(Channel),
  organizationId: z.string(),
  businessUnitId: z.string().optional().nullable(),
  targetUserId: z.string().optional().nullable(),
  targetRoleId: z.string().optional().nullable(),

  title: z.string(),
  message: z.string(),
  data: z.record(z.any()),
  relatedEntities: z.array(relatedEntitySchema),
  actions: z.array(actionSchema),

  expiresAt: z.number().optional().nullable(),
  deliveredAt: z.number().optional().nullable(),
  readAt: z.number().optional().nullable(),
  dismissedAt: z.number().optional().nullable(),

  createdAt: z.number(),
  updatedAt: z.number(),

  deliveryStatus: z.nativeEnum(DeliveryStatus),
  retryCount: z.number(),
  maxRetries: z.number(),

  source: z.string(),
  jobId: z.string().optional().nullable(),
  correlationId: z.string().optional().nullable(),
  tags: z.array(z.string()),

  version: z.number(),
});

export const notificationPreferenceSchema = z.object({
  id: z.string(),
  userId: z.string().min(1, "User is required"),
  organizationId: z.string().min(1, "Organization is required"),
  businessUnitId: z.string().min(1, "Business Unit is required"),

  resource: z.nativeEnum(Resource),
  updateTypes: z.array(z.nativeEnum(UpdateType)).optional(),
  notifyOnAllUpdates: z.boolean().default(false),
  notifyOnlyOwnedRecords: z.boolean().default(true),

  excludedUserIds: z.array(z.string()).optional(),
  includedRoleIds: z.array(z.string()).optional(),
  preferredChannels: z.array(z.nativeEnum(Channel)).optional(),

  quietHoursEnabled: z.boolean().default(false),
  quietHoursStart: z.string().optional(),
  quietHoursEnd: z.string().optional(),
  timezone: z.string().optional(),

  batchNotifications: z.boolean().default(false),
  batchIntervalMinutes: z.number().min(1).max(1440).default(15),

  isActive: z.boolean().default(true),
  version: z.number().default(0),
  createdAt: z.number(),
  updatedAt: z.number(),
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

