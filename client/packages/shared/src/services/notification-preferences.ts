import {
  MyNotificationPreferencesDocument,
  UpdateMyNotificationPreferencesDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  notificationPreferencesSchema,
  type NotificationPreferences,
} from "@trenova/shared/types/notification-preferences";

export const notificationPreferencesService = {
  async get(options?: { signal?: AbortSignal }): Promise<NotificationPreferences> {
    const response = await requestGraphQL({
      document: MyNotificationPreferencesDocument,
      operationName: "MyNotificationPreferences",
      signal: options?.signal,
    });
    return safeParse(
      notificationPreferencesSchema,
      response.myNotificationPreferences,
      "Notification preferences",
    );
  },

  async update(preferences: NotificationPreferences): Promise<NotificationPreferences> {
    const response = await requestGraphQL({
      document: UpdateMyNotificationPreferencesDocument,
      operationName: "UpdateMyNotificationPreferences",
      variables: {
        input: {
          disabledEventTypes: preferences.disabledEventTypes,
          desktopMode: preferences.desktopMode,
          emailMode: preferences.emailMode,
          emailDigest: preferences.emailDigest,
          sound: preferences.sound,
          quietHoursEnabled: preferences.quietHoursEnabled,
          quietHoursStart: preferences.quietHoursStart,
          quietHoursEnd: preferences.quietHoursEnd,
          quietHoursDays: preferences.quietHoursDays,
          quietHoursAllowUrgent: preferences.quietHoursAllowUrgent,
          pausedUntil: preferences.pausedUntil,
        },
      },
    });
    return safeParse(
      notificationPreferencesSchema,
      response.updateMyNotificationPreferences,
      "Notification preferences",
    );
  },
};
