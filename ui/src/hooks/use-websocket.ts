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
          toast.success(message.data.title, {
            description: message.data.message,
          });
          console.log("message", message.data);
          break;
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
    if (enabled && isAuthenticated && user && org) {
      connect();
    } else {
      console.info("Disconnecting WebSocket", user);
      disconnect();
    }

    return () => {
      if (!enabled) {
        disconnect();
      }
    };
  }, [enabled, isAuthenticated, user, org, connect, disconnect]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      disconnect();
    };
  }, [disconnect]);

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
