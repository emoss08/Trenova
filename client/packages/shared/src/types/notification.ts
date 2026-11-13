import { z } from "zod";

export const notificationPrioritySchema = z.enum(["critical", "high", "medium", "low"]);
export type NotificationPriority = z.infer<typeof notificationPrioritySchema>;

export const notificationChannelSchema = z.enum(["global", "user", "role"]);
export type NotificationChannel = z.infer<typeof notificationChannelSchema>;

/**
 * A part of the panel. `forYou` and `team` split what is open by why the person got it,
 * `later` is what they set aside, `done` is what they finished or what cleared itself, and
 * `open` is everything open whatever the reason (the driver portal's one list).
 */
export const notificationLensSchema = z.enum(["forYou", "team", "later", "done", "open"]);
export type NotificationLens = z.infer<typeof notificationLensSchema>;

/** Why a person got a notification: it was theirs, or it went to everyone who could act. */
export const notificationReasonSchema = z.enum(["direct", "team"]);
export type NotificationReason = z.infer<typeof notificationReasonSchema>;

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
  reason: notificationReasonSchema,
  /** How many times the subject came up while this notification was open. */
  occurrenceCount: z.number(),
  /** When it shows, in Unix seconds: in the past for an open one, ahead for one set aside. */
  surfacesAt: z.number(),
  /** Set when the work it pointed at was finished and it cleared itself. */
  resolvedAt: z.number().nullable(),
  /** How it cleared itself, as a code the panel puts into words. */
  resolution: z.string(),
});

export type Notification = z.infer<typeof notificationSchema>;

export const notificationFeedSchema = z.object({
  results: z.array(notificationSchema),
  totalCount: z.number(),
  endCursor: z.string().nullable(),
  hasNextPage: z.boolean(),
});

export type NotificationFeed = z.infer<typeof notificationFeedSchema>;

export const notificationCountsSchema = z.object({
  forYou: z.number(),
  forYouUnread: z.number(),
  team: z.number(),
  teamUnread: z.number(),
  later: z.number(),
});

export type NotificationCounts = z.infer<typeof notificationCountsSchema>;
