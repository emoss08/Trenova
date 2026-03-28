import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const notification = createQueryKeys("notification", {
  list: (params?: { limit?: number; offset?: number }) => ({
    queryKey: [params],
    queryFn: async () => apiService.notificationService.listNotifications(params),
  }),
  unreadCount: () => ({
    queryKey: ["unread-count"],
    queryFn: async () => apiService.notificationService.getUnreadCount(),
  }),
});
