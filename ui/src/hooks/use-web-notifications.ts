import { useCallback, useEffect, useRef, useState } from "react";

// Types
export type NotificationPermission = "default" | "denied" | "granted";

export interface WebNotificationOptions {
  body?: string;
  icon?: string;
  badge?: string;
  tag?: string;
  requireInteraction?: boolean;
  silent?: boolean;
  data?: Record<string, any>;
  image?: string;
  vibrate?: number | number[];
  renotify?: boolean;
  actions?: Array<{
    action: string;
    title: string;
    icon?: string;
  }>;
}

export interface WebNotificationConfig {
  title: string;
  options?: WebNotificationOptions;
  onClick?: (notification: Notification) => void;
  onClose?: (notification: Notification) => void;
  onError?: (error: Error) => void;
  onShow?: (notification: Notification) => void;
}

interface NotificationQueueItem extends WebNotificationConfig {
  id: string;
  timestamp: number;
}

// Constants
const PERMISSION_STORAGE_KEY = "web-notifications-permission";
const NOTIFICATION_RATE_LIMIT = 10; // Max notifications per minute
const NOTIFICATION_RATE_WINDOW = 60000; // 1 minute in ms
const MAX_QUEUE_SIZE = 50;
const DEFAULT_ICON = "/favicon.ico";

export function useWebNotifications() {
  // State
  const [permission, setPermission] = useState<NotificationPermission>(() => {
    if (typeof window === "undefined") return "default";
    if (!("Notification" in window)) return "denied";
    return Notification.permission;
  });

  const [isSupported] = useState(() => {
    if (typeof window === "undefined") return false;
    return "Notification" in window && "serviceWorker" in navigator;
  });

  const [isEnabled, setIsEnabled] = useState(() => {
    if (typeof window === "undefined") return false;
    const stored = localStorage.getItem(PERMISSION_STORAGE_KEY);
    if (stored === "disabled") return false;
    return stored === "granted" && Notification.permission === "granted";
  });

  const notificationQueue = useRef<NotificationQueueItem[]>([]);
  const activeNotifications = useRef<Map<string, Notification>>(new Map());
  const notificationTimestamps = useRef<number[]>([]);
  const processingQueue = useRef(false);

  const isDocumentVisible = useCallback(() => {
    return document.visibilityState === "visible";
  }, []);

  const cleanupTimestamps = useCallback(() => {
    const now = Date.now();
    notificationTimestamps.current = notificationTimestamps.current.filter(
      (timestamp) => now - timestamp < NOTIFICATION_RATE_WINDOW,
    );
  }, []);

  const isWithinRateLimit = useCallback(() => {
    cleanupTimestamps();
    return notificationTimestamps.current.length < NOTIFICATION_RATE_LIMIT;
  }, [cleanupTimestamps]);

  // Request permission
  const requestPermission = useCallback(async (): Promise<boolean> => {
    if (!isSupported) {
      console.warn("[WebNotifications] Browser does not support notifications");
      return false;
    }

    try {
      const result = await Notification.requestPermission();
      setPermission(result);

      if (result === "granted") {
        localStorage.setItem(PERMISSION_STORAGE_KEY, "granted");
        setIsEnabled(true);
        return true;
      } else {
        localStorage.setItem(PERMISSION_STORAGE_KEY, result);
        setIsEnabled(false);
        return false;
      }
    } catch (error) {
      console.error("[WebNotifications] Error requesting permission:", error);
      return false;
    }
  }, [isSupported]);

  const enableNotifications = useCallback(async (): Promise<boolean> => {
    if (!isSupported) {
      console.warn("[WebNotifications] Browser does not support notifications");
      return false;
    }

    // Check if we have browser permission
    if (Notification.permission === "granted") {
      setIsEnabled(true);
      localStorage.setItem(PERMISSION_STORAGE_KEY, "granted");
      return true;
    } else if (Notification.permission === "default") {
      // Need to request permission
      return requestPermission();
    } else {
      // Permission is denied at browser level
      console.warn("[WebNotifications] Permission denied by browser");
      return false;
    }
  }, [isSupported, requestPermission]);

  const disableNotifications = useCallback(() => {
    setIsEnabled(false);
    localStorage.setItem(PERMISSION_STORAGE_KEY, "disabled");

    // Close all active notifications
    activeNotifications.current.forEach((notification) => {
      notification.close();
    });
    activeNotifications.current.clear();
  }, []);

  const processQueue = useCallback(() => {
    if (processingQueue.current || notificationQueue.current.length === 0) {
      return;
    }

    processingQueue.current = true;

    while (notificationQueue.current.length > 0 && isWithinRateLimit()) {
      const item = notificationQueue.current.shift();
      if (!item) continue;

      try {
        const notification = new Notification(item.title, {
          ...item.options,
          icon: item.options?.icon || DEFAULT_ICON,
          tag: item.options?.tag || item.id,
        });

        activeNotifications.current.set(item.id, notification);
        notificationTimestamps.current.push(Date.now());

        notification.onclick = (event) => {
          event.preventDefault();
          window.focus();
          item.onClick?.(notification);
          notification.close();
        };

        notification.onclose = () => {
          activeNotifications.current.delete(item.id);
          item.onClose?.(notification);
        };

        notification.onerror = () => {
          activeNotifications.current.delete(item.id);
          item.onError?.(new Error("Notification failed to display"));
        };

        notification.onshow = () => {
          item.onShow?.(notification);
        };
      } catch (error) {
        console.error("[WebNotifications] Error showing notification:", error);
        item.onError?.(
          error instanceof Error ? error : new Error(String(error)),
        );
      }
    }

    processingQueue.current = false;
  }, [isWithinRateLimit]);

  const showNotification = useCallback(
    (config: WebNotificationConfig): string | null => {
      if (!isSupported) {
        console.warn("[WebNotifications] Notifications not supported");
        return null;
      }

      if (permission !== "granted" || !isEnabled) {
        console.warn(
          "[WebNotifications] Permission not granted or notifications disabled",
        );
        return null;
      }

      if (isDocumentVisible()) {
        return null;
      }

      const id = `notif-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      const queueItem: NotificationQueueItem = {
        ...config,
        id,
        timestamp: Date.now(),
      };

      notificationQueue.current.push(queueItem);

      if (notificationQueue.current.length > MAX_QUEUE_SIZE) {
        notificationQueue.current =
          notificationQueue.current.slice(-MAX_QUEUE_SIZE);
      }

      processQueue();

      return id;
    },
    [isSupported, permission, isEnabled, isDocumentVisible, processQueue],
  );

  const clearAll = useCallback(() => {
    notificationQueue.current = [];
    activeNotifications.current.forEach((notification) => {
      notification.close();
    });
    activeNotifications.current.clear();
  }, []);

  const clearNotification = useCallback((notificationId: string) => {
    notificationQueue.current = notificationQueue.current.filter(
      (item) => item.id !== notificationId,
    );

    const notification = activeNotifications.current.get(notificationId);
    if (notification) {
      notification.close();
      activeNotifications.current.delete(notificationId);
    }
  }, []);

  useEffect(() => {
    if (!isSupported) return;

    const handlePermissionChange = () => {
      setPermission(Notification.permission);
      if (Notification.permission !== "granted") {
        setIsEnabled(false);
      }
    };

    if ("permissions" in navigator) {
      navigator.permissions
        .query({ name: "notifications" as PermissionName })
        .then((permissionStatus) => {
          permissionStatus.addEventListener("change", handlePermissionChange);
          return () => {
            permissionStatus.removeEventListener(
              "change",
              handlePermissionChange,
            );
          };
        })
        .catch(console.error);
    }
  }, [isSupported]);

  useEffect(() => {
    const handleVisibilityChange = () => {
      if (!isDocumentVisible()) {
        processQueue();
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [isDocumentVisible, processQueue]);

  useEffect(() => {
    return () => {
      clearAll();
    };
  }, [clearAll]);

  return {
    permission,
    isEnabled,
    isGranted: permission === "granted",
    requestPermission,
    enableNotifications,
    disableNotifications,
    showNotification,
    clearAll,
    clearNotification,
  };
}
