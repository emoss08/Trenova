import { z } from "zod";

export const deliveryModeSchema = z.enum(["off", "urgent", "all"]);
export type DeliveryMode = z.infer<typeof deliveryModeSchema>;

const clockSchema = z.string().regex(/^([01][0-9]|2[0-3]):[0-5][0-9]$/);

export const notificationPreferencesSchema = z.object({
  disabledEventTypes: z
    .array(z.string())
    .nullish()
    .transform((value) => value ?? []),
  desktopMode: deliveryModeSchema,
  emailMode: deliveryModeSchema,
  emailDigest: z.boolean(),
  sound: z.boolean(),
  quietHoursEnabled: z.boolean(),
  quietHoursStart: clockSchema,
  quietHoursEnd: clockSchema,
  quietHoursDays: z
    .array(z.number().int().min(0).max(6))
    .nullish()
    .transform((value) => value ?? []),
  quietHoursAllowUrgent: z.boolean(),
  pausedUntil: z
    .number()
    .nullish()
    .transform((value) => value ?? null),
});

export type NotificationPreferences = z.infer<typeof notificationPreferencesSchema>;
