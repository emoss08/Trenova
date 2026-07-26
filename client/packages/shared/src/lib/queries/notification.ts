import { notificationService } from "@trenova/shared/services/notification";
import type {
  NotificationFeedParams,
  NotificationScope,
} from "@trenova/shared/services/notification";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const notification = createQueryKeys("notification", {
  feed: (params?: NotificationFeedParams) => ({
    queryKey: [params],
    queryFn: async () => notificationService.listNotifications(params),
  }),
  unreadCount: (scope: NotificationScope = "all") => ({
    queryKey: ["unread-count", scope],
    queryFn: async () => notificationService.getUnreadCount(scope),
  }),
});
