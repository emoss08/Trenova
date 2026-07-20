import { z } from "zod";

export const notificationPrioritySchema = z.enum(["critical", "high", "medium", "low"]);
export type NotificationPriority = z.infer<typeof notificationPrioritySchema>;

export const notificationChannelSchema = z.enum(["global", "user", "role"]);
export type NotificationChannel = z.infer<typeof notificationChannelSchema>;

export const notificationStateSchema = z.enum(["inbox", "archived"]);
export type NotificationState = z.infer<typeof notificationStateSchema>;

export const notificationSchema = z.object({
  id: z.string(),
  organizationId: z.string(),
  businessUnitId: z.string().nullable(),
  targetUserId: z.string().nullable(),
  eventType: z.string(),
  priority: notificationPrioritySchema,
  channel: notificationChannelSchema,
  title: z.string(),
  message: z.string(),
  data: z.record(z.string(), z.unknown()).nullable(),
  relatedEntities: z.record(z.string(), z.unknown()).nullable(),
  source: z.string(),
  readAt: z.number().nullable(),
  dismissedAt: z.number().nullable(),
  createdAt: z.number(),
});

export type Notification = z.infer<typeof notificationSchema>;

export const notificationFeedSchema = z.object({
  results: z.array(notificationSchema),
  totalCount: z.number(),
  endCursor: z.string().nullable(),
  hasNextPage: z.boolean(),
});

export type NotificationFeed = z.infer<typeof notificationFeedSchema>;
