import { z } from "zod";
import { createLimitOffsetResponse } from "./server";

export const notificationPrioritySchema = z.enum(["critical", "high", "medium", "low"]);
export type NotificationPriority = z.infer<typeof notificationPrioritySchema>;

export const notificationChannelSchema = z.enum(["global", "user", "role"]);
export type NotificationChannel = z.infer<typeof notificationChannelSchema>;

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
  source: z.string(),
  readAt: z.number().nullable(),
  createdAt: z.number(),
});

export type Notification = z.infer<typeof notificationSchema>;

export const notificationResponseSchema = createLimitOffsetResponse(notificationSchema);

export type NotificationResponse = z.infer<typeof notificationResponseSchema>;
