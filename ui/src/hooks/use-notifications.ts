import { api } from "@/services/api";
import { 
  NotificationPreference, 
  CreateNotificationPreferenceInput, 
  UpdateNotificationPreferenceInput 
} from "@/types/notification";
import { useWebSocketStore } from "@/stores/websocket-store";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect } from "react";
import { toast } from "sonner";

// Query keys
const NOTIFICATION_KEYS = {
  all: ["notifications"] as const,
  preferences: () => [...NOTIFICATION_KEYS.all, "preferences"] as const,
  preference: (id: string) => [...NOTIFICATION_KEYS.preferences(), id] as const,
  userPreferences: (userId: string) => [...NOTIFICATION_KEYS.preferences(), "user", userId] as const,
  history: () => [...NOTIFICATION_KEYS.all, "history"] as const,
  unreadCount: () => [...NOTIFICATION_KEYS.all, "unread-count"] as const,
};

// Hook to get notification preferences
export function useNotificationPreferences(params?: {
  resource?: string;
  isActive?: boolean;
}) {
  return useQuery({
    queryKey: [...NOTIFICATION_KEYS.preferences(), params],
    queryFn: () => api.notifications.getPreferences(params),
  });
}

// Hook to get a specific preference
export function useNotificationPreference(id: string) {
  return useQuery({
    queryKey: NOTIFICATION_KEYS.preference(id),
    queryFn: () => api.notifications.getPreference(id),
    enabled: !!id,
  });
}

// Hook to get user preferences (admin)
export function useUserNotificationPreferences(userId: string) {
  return useQuery({
    queryKey: NOTIFICATION_KEYS.userPreferences(userId),
    queryFn: () => api.notifications.getUserPreferences(userId),
    enabled: !!userId,
  });
}

// Hook to create notification preference
export function useCreateNotificationPreference() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateNotificationPreferenceInput) => 
      api.notifications.createPreference(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NOTIFICATION_KEYS.preferences() });
      toast.success("Notification preference created");
    },
    onError: (error: any) => {
      toast.error(`Failed to create preference: ${error.message}`);
    },
  });
}

// Hook to update notification preference
export function useUpdateNotificationPreference() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateNotificationPreferenceInput }) => 
      api.notifications.updatePreference(id, data),
    onSuccess: (_: any, variables: { id: string; data: UpdateNotificationPreferenceInput }) => {
      queryClient.invalidateQueries({ queryKey: NOTIFICATION_KEYS.preferences() });
      queryClient.invalidateQueries({ queryKey: NOTIFICATION_KEYS.preference(variables.id) });
      toast.success("Notification preference updated");
    },
    onError: (error: any) => {
      toast.error(`Failed to update preference: ${error.message}`);
    },
  });
}

// Hook to delete notification preference
export function useDeleteNotificationPreference() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.notifications.deletePreference(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: NOTIFICATION_KEYS.preferences() });
      toast.success("Notification preference deleted");
    },
    onError: (error: any) => {
      toast.error(`Failed to delete preference: ${error.message}`);
    },
  });
}

// Hook to get notification history
export function useNotificationHistory(params?: {
  limit?: number;
  offset?: number;
  unreadOnly?: boolean;
  resource?: string;
  priority?: string;
  startDate?: number;
  endDate?: number;
}) {
  return useQuery({
    queryKey: [...NOTIFICATION_KEYS.history(), params],
    queryFn: () => api.notifications.getHistory(params),
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
      queryClient.invalidateQueries({ queryKey: NOTIFICATION_KEYS.history() });
      queryClient.invalidateQueries({ queryKey: NOTIFICATION_KEYS.unreadCount() });
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
      queryClient.invalidateQueries({ queryKey: NOTIFICATION_KEYS.history() });
      queryClient.invalidateQueries({ queryKey: NOTIFICATION_KEYS.unreadCount() });
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
      queryClient.invalidateQueries({ queryKey: NOTIFICATION_KEYS.history() });
    },
  });
}

// Hook to get unread count
export function useUnreadCount() {
  const { unreadCount } = useWebSocketStore();
  
  const query = useQuery({
    queryKey: NOTIFICATION_KEYS.unreadCount(),
    queryFn: () => api.notifications.getUnreadCount(),
  });

  // Return local state which is updated in real-time via WebSocket
  return {
    ...query,
    data: { count: unreadCount },
  };
}

// Hook to handle notification actions
export function useNotificationActions() {
  const markAsRead = useMarkAsRead();
  const markAllAsRead = useMarkAllAsRead();
  const dismiss = useDismissNotification();

  const handleNotificationClick = useCallback((notificationId: string, data?: any) => {
    // Mark as read when clicked
    markAsRead.mutate(notificationId);

    // Handle navigation or other actions based on notification data
    if (data?.entityType && data?.entityId) {
      // Navigate to the entity (you can customize this based on your routing)
      const entityPath = `/${data.entityType}/${data.entityId}`;
      window.location.href = entityPath;
    }
  }, [markAsRead]);

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