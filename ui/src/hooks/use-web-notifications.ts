/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

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
    return stored === "granted" && permission === "granted";
  });

  // Refs for managing notifications and rate limiting
  const notificationQueue = useRef<NotificationQueueItem[]>([]);
  const activeNotifications = useRef<Map<string, Notification>>(new Map());
  const notificationTimestamps = useRef<number[]>([]);
  const processingQueue = useRef(false);

  // Check if document is visible
  const isDocumentVisible = useCallback(() => {
    return document.visibilityState === "visible";
  }, []);

  // Clean up old timestamps for rate limiting
  const cleanupTimestamps = useCallback(() => {
    const now = Date.now();
    notificationTimestamps.current = notificationTimestamps.current.filter(
      (timestamp) => now - timestamp < NOTIFICATION_RATE_WINDOW,
    );
  }, []);

  // Check if we're within rate limit
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

  // Disable notifications
  const disableNotifications = useCallback(() => {
    setIsEnabled(false);
    localStorage.setItem(PERMISSION_STORAGE_KEY, "disabled");

    // Close all active notifications
    activeNotifications.current.forEach((notification) => {
      notification.close();
    });
    activeNotifications.current.clear();
  }, []);

  // Process notification queue
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

        // Track active notification
        activeNotifications.current.set(item.id, notification);
        notificationTimestamps.current.push(Date.now());

        // Set up event handlers
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

  // Show notification
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

      // Don't show if document is visible (user is actively using the app)
      if (isDocumentVisible()) {
        console.log(
          "[WebNotifications] Document is visible, skipping notification",
        );
        return null;
      }

      const id = `notif-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
      const queueItem: NotificationQueueItem = {
        ...config,
        id,
        timestamp: Date.now(),
      };

      // Add to queue
      notificationQueue.current.push(queueItem);

      // Trim queue if it's too large
      if (notificationQueue.current.length > MAX_QUEUE_SIZE) {
        notificationQueue.current =
          notificationQueue.current.slice(-MAX_QUEUE_SIZE);
      }

      // Process queue
      processQueue();

      return id;
    },
    [isSupported, permission, isEnabled, isDocumentVisible, processQueue],
  );

  // Clear all notifications
  const clearAll = useCallback(() => {
    notificationQueue.current = [];
    activeNotifications.current.forEach((notification) => {
      notification.close();
    });
    activeNotifications.current.clear();
  }, []);

  // Clear specific notification
  const clearNotification = useCallback((notificationId: string) => {
    // Remove from queue
    notificationQueue.current = notificationQueue.current.filter(
      (item) => item.id !== notificationId,
    );

    // Close if active
    const notification = activeNotifications.current.get(notificationId);
    if (notification) {
      notification.close();
      activeNotifications.current.delete(notificationId);
    }
  }, []);

  // Test notification
  const testNotification = useCallback(() => {
    return showNotification({
      title: "Test Notification",
      options: {
        body: "This is a test notification from Trenova",
        icon: DEFAULT_ICON,
        tag: "test-notification",
      },
      onClick: () => {
        console.log("[WebNotifications] Test notification clicked");
      },
    });
  }, [showNotification]);

  // Handle permission changes
  useEffect(() => {
    if (!isSupported) return;

    const handlePermissionChange = () => {
      setPermission(Notification.permission);
      if (Notification.permission !== "granted") {
        setIsEnabled(false);
      }
    };

    // Some browsers support permission change events
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

  // Process queue when visibility changes
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

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      clearAll();
    };
  }, [clearAll]);

  return {
    // State
    permission,
    isSupported,
    isEnabled,
    isGranted: permission === "granted",

    // Actions
    requestPermission,
    showNotification,
    disableNotifications,
    clearAll,
    clearNotification,
    testNotification,

    // Utilities
    isDocumentVisible,
    queueSize: notificationQueue.current.length,
    activeCount: activeNotifications.current.size,
  };
}
