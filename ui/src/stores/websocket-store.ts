import type {
  NotificationMessage,
  WebSocketConnectionState,
  WebSocketSubscription,
} from "@/types/websocket";
import { create } from "zustand";
import { devtools } from "zustand/middleware";

interface WebSocketState extends WebSocketConnectionState {
  notifications: NotificationMessage[];
  unreadCount: number;
  subscription: WebSocketSubscription | null;
}

interface WebSocketActions {
  setSocket: (socket: WebSocket | null) => void;
  setConnectionState: (
    state: WebSocketConnectionState["connectionState"],
  ) => void;
  setConnected: (connected: boolean) => void;
  setSubscription: (subscription: WebSocketSubscription | null) => void;
  addNotification: (notification: NotificationMessage) => void;
  markAsRead: (notificationId: string) => void;
  markAllAsRead: () => void;
  dismissNotification: (notificationId: string) => void;
  clearNotifications: () => void;
  removeExpiredNotifications: () => void;
  incrementReconnectAttempts: () => void;
  resetReconnectAttempts: () => void;
  setLastError: (error: string | undefined) => void;
}

export const useWebSocketStore = create<WebSocketState & WebSocketActions>()(
  devtools(
    (set, get) => ({
      // Connection State
      socket: null,
      isConnected: false,
      connectionState: "disconnected",
      reconnectAttempts: 0,
      lastError: undefined,

      // Subscription State
      subscription: null,

      // Notification State
      notifications: [],
      unreadCount: 0,

      // Actions
      setSocket: (socket) => set({ socket }),

      setConnectionState: (connectionState) => {
        set({ connectionState });
        if (connectionState === "connected") {
          set({ isConnected: true, lastError: undefined });
        } else if (connectionState === "disconnected") {
          set({ isConnected: false });
        }
      },

      setConnected: (isConnected) => set({ isConnected }),

      setSubscription: (subscription) => set({ subscription }),

      addNotification: (notification) => {
        const { notifications } = get();

        // Avoid duplicates
        if (notifications.some((n) => n.id === notification.id)) {
          return;
        }

        const newNotifications = [notification, ...notifications];
        const unreadCount = newNotifications.filter((n) => !n.readAt).length;

        set({
          notifications: newNotifications,
          unreadCount,
        });
      },

      markAsRead: (notificationId) => {
        const { notifications } = get();
        const now = Date.now();

        const updatedNotifications = notifications.map((notification) =>
          notification.id === notificationId
            ? { ...notification, readAt: now }
            : notification,
        );

        const unreadCount = updatedNotifications.filter(
          (n) => !n.readAt,
        ).length;

        set({
          notifications: updatedNotifications,
          unreadCount,
        });
      },

      markAllAsRead: () => {
        const { notifications } = get();
        const now = Date.now();

        const updatedNotifications = notifications.map((notification) => ({
          ...notification,
          readAt: notification.readAt || now,
        }));

        set({
          notifications: updatedNotifications,
          unreadCount: 0,
        });
      },

      dismissNotification: (notificationId) => {
        const { notifications } = get();
        const now = Date.now();

        const updatedNotifications = notifications.map((notification) =>
          notification.id === notificationId
            ? { ...notification, dismissedAt: now }
            : notification,
        );

        const unreadCount = updatedNotifications.filter(
          (n) => !n.readAt && !n.dismissedAt,
        ).length;

        set({
          notifications: updatedNotifications,
          unreadCount,
        });
      },

      clearNotifications: () => set({ notifications: [], unreadCount: 0 }),

      removeExpiredNotifications: () => {
        const { notifications } = get();
        const now = Date.now();

        const validNotifications = notifications.filter(
          (notification) =>
            !notification.expiresAt || notification.expiresAt > now,
        );

        const unreadCount = validNotifications.filter(
          (n) => !n.readAt && !n.dismissedAt,
        ).length;

        set({
          notifications: validNotifications,
          unreadCount,
        });
      },

      incrementReconnectAttempts: () => {
        const { reconnectAttempts } = get();
        set({ reconnectAttempts: reconnectAttempts + 1 });
      },

      resetReconnectAttempts: () => set({ reconnectAttempts: 0 }),

      setLastError: (lastError) => set({ lastError }),
    }),
    {
      name: "websocket-store",
    },
  ),
);
