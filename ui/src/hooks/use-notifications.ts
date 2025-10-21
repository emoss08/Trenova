import { queries } from "@/lib/queries";
import { api } from "@/services/api";
import { useWebSocketStore } from "@/stores/websocket-store";
import type { NotificationQueryParams } from "@/types/notification";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { toast } from "sonner";

export function useNotificationHistory(params?: NotificationQueryParams) {
  return useQuery({
    ...queries.notification.list(params),
    enabled: !!params,
  });
}

export function useMarkAsRead() {
  const queryClient = useQueryClient();
  const { markAsRead } = useWebSocketStore();

  return useMutation({
    mutationFn: (notificationId: string) => {
      // Update local state immediately
      markAsRead(notificationId);
      // Then sync with server
      return api.notifications.markAsRead(notificationId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queries.notification.list._def,
      });
    },
  });
}

export function useMarkAllAsRead() {
  const queryClient = useQueryClient();
  const { markAllAsRead } = useWebSocketStore();

  return useMutation({
    mutationFn: () => {
      markAllAsRead();
      return api.notifications.markAllAsRead();
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queries.notification.list._def,
      });
      toast.success("All notifications marked as read");
    },
  });
}

export function useDismissNotification() {
  const queryClient = useQueryClient();
  const { dismissNotification } = useWebSocketStore();

  return useMutation({
    mutationFn: (notificationId: string) => {
      dismissNotification(notificationId);
      return api.notifications.dismiss(notificationId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queries.notification.list._def,
      });
    },
  });
}

export function useNotificationActions() {
  const markAsRead = useMarkAsRead();
  const markAllAsRead = useMarkAllAsRead();
  const dismiss = useDismissNotification();

  const handleNotificationClick = useCallback(
    (notificationId: string, data?: any) => {
      markAsRead.mutate(notificationId);

      if (data?.entityType && data?.entityId) {
        const entityPath = `/${data.entityType}/${data.entityId}`;
        window.location.href = entityPath;
      }
    },
    [markAsRead],
  );

  return {
    markAsRead,
    markAllAsRead,
    dismiss,
    handleNotificationClick,
  };
}

export function useNotificationCleanup() {
  const { removeExpiredNotifications } = useWebSocketStore();

  useEffect(() => {
    const interval = setInterval(() => {
      removeExpiredNotifications();
    }, 60000);

    return () => clearInterval(interval);
  }, [removeExpiredNotifications]);
}
