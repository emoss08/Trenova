/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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

// Hook to mark notification as read
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

// Hook to mark all as read
export function useMarkAllAsRead() {
  const queryClient = useQueryClient();
  const { markAllAsRead } = useWebSocketStore();

  return useMutation({
    mutationFn: () => {
      // Update local state immediately
      markAllAsRead();
      // Then sync with server
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

// Hook to dismiss notification
export function useDismissNotification() {
  const queryClient = useQueryClient();
  const { dismissNotification } = useWebSocketStore();

  return useMutation({
    mutationFn: (notificationId: string) => {
      // Update local state immediately
      dismissNotification(notificationId);
      // Then sync with server
      return api.notifications.dismiss(notificationId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queries.notification.list._def,
      });
    },
  });
}

// Hook to handle notification actions
export function useNotificationActions() {
  const markAsRead = useMarkAsRead();
  const markAllAsRead = useMarkAllAsRead();
  const dismiss = useDismissNotification();

  const handleNotificationClick = useCallback(
    (notificationId: string, data?: any) => {
      // Mark as read when clicked
      markAsRead.mutate(notificationId);

      // Handle navigation or other actions based on notification data
      if (data?.entityType && data?.entityId) {
        // Navigate to the entity (you can customize this based on your routing)
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

// Hook to clean up expired notifications
export function useNotificationCleanup() {
  const { removeExpiredNotifications } = useWebSocketStore();

  useEffect(() => {
    // Check for expired notifications every minute
    const interval = setInterval(() => {
      removeExpiredNotifications();
    }, 60000);

    return () => clearInterval(interval);
  }, [removeExpiredNotifications]);
}
