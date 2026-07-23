import { apiService } from "@/services/api";
import type { NotificationFeedParams } from "@/services/notification";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const notification = createQueryKeys("notification", {
  feed: (params?: NotificationFeedParams) => ({
    queryKey: [params],
    queryFn: async () => apiService.notificationService.listNotifications(params),
  }),
  unreadCount: () => ({
    queryKey: ["unread-count"],
    queryFn: async () => apiService.notificationService.getUnreadCount(),
  }),
});
