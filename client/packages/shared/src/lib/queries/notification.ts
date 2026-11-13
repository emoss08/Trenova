import { notificationService } from "@trenova/shared/services/notification";
import type { NotificationFeedParams } from "@trenova/shared/services/notification";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const notification = createQueryKeys("notification", {
  feed: (params?: NotificationFeedParams) => ({
    queryKey: [params],
    queryFn: async ({ signal }) => notificationService.listNotifications(params, { signal }),
  }),
  counts: () => ({
    queryKey: ["counts"],
    queryFn: async ({ signal }) => notificationService.getCounts({ signal }),
  }),
  myUnreadCount: () => ({
    queryKey: ["my-unread-count"],
    queryFn: async ({ signal }) => notificationService.getMyUnreadCount({ signal }),
  }),
});
