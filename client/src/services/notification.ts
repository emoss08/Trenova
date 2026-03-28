import { api } from "@/lib/api";
import { safeParse } from "@/lib/parse";
import {
  notificationResponseSchema,
  type Notification,
  type NotificationResponse,
} from "@/types/notification";

export class NotificationService {
  public async listNotifications(params?: {
    limit?: number;
    offset?: number;
  }) {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.set("limit", String(params.limit));
    if (params?.offset) searchParams.set("offset", String(params.offset));

    const queryString = searchParams.toString();
    const response = await api.get<NotificationResponse>(
      `/notifications/?${queryString}`,
    );

    return safeParse(
      notificationResponseSchema,
      response,
      "Notification List",
    );
  }

  public async getUnreadCount() {
    const response = await api.get<{ count: number }>(
      "/notifications/unread-count",
    );

    return response.count;
  }

  public async markRead(ids: Notification["id"][]) {
    await api.patch("/notifications/mark-read", { ids });
  }

  public async markAllRead() {
    await api.patch("/notifications/mark-all-read");
  }
}
