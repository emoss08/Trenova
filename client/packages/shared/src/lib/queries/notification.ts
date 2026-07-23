import { notificationService } from "@trenova/shared/services/notification";
import type { NotificationFeedParams } from "@trenova/shared/services/notification";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const notification = createQueryKeys("notification", {
  feed: (params?: NotificationFeedParams) => ({
    queryKey: [params],
    queryFn: async () => notificationService.listNotifications(params),
  }),
  unreadCount: () => ({
    queryKey: ["unread-count"],
    queryFn: async () => notificationService.getUnreadCount(),
  }),
});
