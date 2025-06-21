import { http } from "@/lib/http-client";
import type { NotificationSchema } from "@/lib/schemas/notification-schema";
import type { NotificationQueryParams } from "@/types/notification";
import type { LimitOffsetResponse } from "@/types/server";

export class NotificationAPI {
  async list(params?: NotificationQueryParams) {
    const response = await http.get<LimitOffsetResponse<NotificationSchema>>(
      "/notifications/",
      {
        params: params
          ? {
              limit: params.limit?.toString(),
              offset: params.offset?.toString(),
              unreadOnly: params.unreadOnly?.toString(),
            }
          : undefined,
      },
    );

    return response.data;
  }

  async markAsRead(notificationId: string) {
    await http.post(`/notifications/${notificationId}/read/`);
  }

  async markAllAsRead() {
    await http.post("/notifications/read-all");
  }

  async dismiss(notificationId: string) {
    await http.post(`/notifications/${notificationId}/dismiss`);
  }
}
