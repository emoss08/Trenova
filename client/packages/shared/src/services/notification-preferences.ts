import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import {
  notificationPreferencesSchema,
  type NotificationPreferences,
} from "@trenova/shared/types/notification-preferences";

const BASE = "/users/me/notification-preferences";

export const notificationPreferencesService = {
  async get(): Promise<NotificationPreferences> {
    const response = await api.get<NotificationPreferences>(`${BASE}/`);
    return safeParse(notificationPreferencesSchema, response, "Notification preferences");
  },

  async update(preferences: NotificationPreferences): Promise<NotificationPreferences> {
    const response = await api.put<NotificationPreferences>(`${BASE}/`, preferences);
    return safeParse(notificationPreferencesSchema, response, "Notification preferences");
  },
};
