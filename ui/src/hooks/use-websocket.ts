import { queries } from "@/lib/queries";
import { webSocketService } from "@/services/websocket";
import { useIsAuthenticated, useUser } from "@/stores/user-store";
import { useWebSocketStore } from "@/stores/websocket-store";
import type {
  WebSocketMessage,
  WebSocketSubscription,
} from "@/types/websocket";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useEffect, useRef } from "react";
import { toast } from "sonner";
import { useWebNotifications } from "./use-web-notifications";

export interface UseWebSocketOptions {
  enabled?: boolean;
  onMessage?: (message: WebSocketMessage) => void;
  onError?: (error: string) => void;
}

export function useWebSocket({
  enabled = true,
  onMessage,
  onError,
}: UseWebSocketOptions = {}) {
  const user = useUser();
  const isAuthenticated = useIsAuthenticated();
  const { showNotification } = useWebNotifications();
  const userOrganization = useQuery({
    ...queries.organization.getOrgById(
      user?.currentOrganizationId ?? "",
      true,
      false,
    ),
    enabled: !!user?.currentOrganizationId && isAuthenticated,
  });

  const org = userOrganization.data;

  const {
    setSocket,
    setConnectionState,
    setSubscription,
    addNotification,
    incrementReconnectAttempts,
    resetReconnectAttempts,
    setLastError,
  } = useWebSocketStore();

  const subscriptionRef = useRef<WebSocketSubscription | null>(null);
  const isConnectingRef = useRef(false);
  const processedMessagesRef = useRef<Set<string>>(new Set());

  const createSubscription = useCallback((): WebSocketSubscription | null => {
    if (!user || !org || !isAuthenticated) {
      return null;
    }

    return {
      userId: user.id!,
      organizationId: org.id!,
      businessUnitId: org.businessUnitId,
      room: `org_${org.id}_user_${user.id}`,
      roles: user.roles?.map((role) => role.id!) || [],
    };
  }, [user, org, isAuthenticated]);

  const handleMessage = useCallback(
    (message: WebSocketMessage) => {
      // Deduplicate messages by ID for notifications
      if (message.type === "notification" && message.data?.id) {
        const messageId = message.data.id;
        if (processedMessagesRef.current.has(messageId)) {
          console.log("Skipping duplicate notification:", messageId);
          return;
        }
        processedMessagesRef.current.add(messageId);

        // Clean up old message IDs to prevent memory leak (keep last 100)
        if (processedMessagesRef.current.size > 100) {
          const idsArray = Array.from(processedMessagesRef.current);
          processedMessagesRef.current = new Set(idsArray.slice(-100));
        }
      }

      switch (message.type) {
        case "notification":
          addNotification(message.data);
          showNotification({
            title: message.data.title,
            options: {
              body: message.data.message,
              icon: message.data.icon,
            },
            onClick: () => {
              // * TODO(wolfred): we need to make a master event handler for this rather than hardcoding it here.
              // We will not send notification for every event type, but we will have a master event handler for all of them.
              if (
                message.data.eventType === "job.shipment.duplicate_complete"
              ) {
                window.location.href = "/shipments/management/";
              }
            },
          });

          break;
        case "entity_update_notification": {
          // Handle entity update notifications
          addNotification(message.data);

          // Show toast with action based on entity type
          const entityLink =
            message.data?.data?.entityId && message.data?.data?.entityType
              ? `/${message.data.data.entityType}/${message.data.data.entityId}`
              : null;

          toast.success(message.data.title, {
            description: message.data.message,
            action: entityLink
              ? {
                  label: "View",
                  onClick: () => {
                    window.location.href = entityLink;
                  },
                }
              : undefined,
          });
          break;
        }
        case "connection_confirmed":
          console.log("✅ WebSocket connection confirmed");
          setConnectionState("connected");
          resetReconnectAttempts();
          break;
        case "error":
          console.error("❌ WebSocket server error:", message.data);
          setLastError(message.data?.message || "Server error");
          break;
        default:
          console.log("📨 WebSocket message:", message.type, message.data);
      }

      onMessage?.(message);
    },
    [
      addNotification,
      setConnectionState,
      resetReconnectAttempts,
      setLastError,
      onMessage,
    ],
  );

  const handleConnectionChange = useCallback(
    (connected: boolean) => {
      setConnectionState(connected ? "connected" : "disconnected");

      if (!connected) {
        incrementReconnectAttempts();
      }
    },
    [setConnectionState, incrementReconnectAttempts],
  );

  const handleError = useCallback(
    (error: string) => {
      setLastError(error);
      onError?.(error);
    },
    [setLastError, onError],
  );

  const connect = useCallback(async () => {
    if (!enabled || isConnectingRef.current) {
      return;
    }

    const subscription = createSubscription();
    if (!subscription) {
      console.log(
        "🚫 WebSocket connection skipped: missing user/organization data",
      );
      return;
    }

    if (
      subscriptionRef.current?.userId === subscription.userId &&
      subscriptionRef.current?.organizationId === subscription.organizationId
    ) {
      // Already connected with same subscription
      return;
    }

    try {
      isConnectingRef.current = true;
      setConnectionState("connecting");

      // Set up event handlers
      webSocketService.setEventHandlers({
        onMessage: handleMessage,
        onConnectionChange: handleConnectionChange,
        onError: handleError,
      });

      console.log("🔗 Connecting WebSocket with subscription:", subscription);
      await webSocketService.connect(subscription);

      subscriptionRef.current = subscription;
      setSocket(
        webSocketService.getConnectionState().isConnected
          ? ({} as WebSocket)
          : null,
      );
      setSubscription(subscription);

      console.log("✅ WebSocket connected successfully");
    } catch (error) {
      console.error("❌ WebSocket connection failed:", error);
      setConnectionState("disconnected");
      setLastError(
        error instanceof Error ? error.message : "Connection failed",
      );
    } finally {
      isConnectingRef.current = false;
    }
  }, [
    setLastError,
    enabled,
    createSubscription,
    setConnectionState,
    setSocket,
    setSubscription,
    handleMessage,
    handleConnectionChange,
    handleError,
  ]);

  const disconnect = useCallback(() => {
    console.log("🔌 Disconnecting WebSocket");
    webSocketService.disconnect();
    subscriptionRef.current = null;
    setSocket(null);
    setSubscription(null);
    setConnectionState("disconnected");
  }, [setSocket, setSubscription, setConnectionState]);

  // Connect/disconnect based on authentication and enabled state
  useEffect(() => {
    const shouldConnect = enabled && isAuthenticated && user && org;

    if (shouldConnect) {
      connect();
    } else if (
      !shouldConnect &&
      webSocketService.getConnectionState().isConnected
    ) {
      console.info("Disconnecting WebSocket", user);
      webSocketService.disconnect();
      subscriptionRef.current = null;
      setSocket(null);
      setSubscription(null);
      setConnectionState("disconnected");
    }
  }, [
    enabled,
    isAuthenticated,
    user,
    org,
    connect,
    setSocket,
    setSubscription,
    setConnectionState,
  ]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      webSocketService.disconnect();
      // Don't update state during unmount to avoid state update warnings
    };
  }, []);

  const markAsRead = useCallback((notificationId: string) => {
    webSocketService.markNotificationAsRead(notificationId);
  }, []);

  const markAsDismissed = useCallback((notificationId: string) => {
    webSocketService.markNotificationAsDismissed(notificationId);
  }, []);

  return {
    connect,
    disconnect,
    markAsRead,
    markAsDismissed,
    connectionState: webSocketService.getConnectionState(),
  };
}
