import { WEBSOCKET_URL } from "@/constants/env";
import type {
  NotificationMessage,
  WebSocketConfig,
  WebSocketMessage,
  WebSocketSubscription,
} from "@/types/websocket";

export class WebSocketService {
  private socket: WebSocket | null = null;
  private config: WebSocketConfig;
  private subscription: WebSocketSubscription | null = null;
  private heartbeatInterval: number | null = null;
  private reconnectTimeout: number | null = null;
  private isReconnecting = false;
  private onMessage?: (message: WebSocketMessage) => void;
  private onConnectionChange?: (connected: boolean) => void;
  private onError?: (error: string) => void;

  constructor(config?: Partial<WebSocketConfig>) {
    this.config = {
      url: this.getWebSocketUrl(),
      reconnectInterval: 5000,
      maxReconnectAttempts: 10,
      heartbeatInterval: 30000,
      ...config,
    };
  }

  private getWebSocketUrl(): string {
    if (!WEBSOCKET_URL) {
      throw new Error("WEBSOCKET_URL is not defined");
    }

    return `${WEBSOCKET_URL}/ws/notifications`;
  }

  setEventHandlers({
    onMessage,
    onConnectionChange,
    onError,
  }: {
    onMessage?: (message: WebSocketMessage) => void;
    onConnectionChange?: (connected: boolean) => void;
    onError?: (error: string) => void;
  }) {
    this.onMessage = onMessage;
    this.onConnectionChange = onConnectionChange;
    this.onError = onError;
  }

  connect(subscription: WebSocketSubscription): Promise<void> {
    return new Promise((resolve, reject) => {
      try {
        if (this.socket?.readyState === WebSocket.OPEN) {
          resolve();
          return;
        }

        this.subscription = subscription;
        this.socket = new WebSocket(this.config.url);

        this.socket.onopen = () => {
          console.log("WebSocket connected");
          this.onConnectionChange?.(true);
          this.startHeartbeat();
          this.isReconnecting = false;
          resolve();
        };

        this.socket.onmessage = (event) => {
          try {
            const message: WebSocketMessage = JSON.parse(event.data);
            this.handleMessage(message);
          } catch (error) {
            console.error("Failed to parse WebSocket message:", error);
          }
        };

        this.socket.onclose = (event) => {
          console.log("WebSocket disconnected:", event.code, event.reason);
          this.cleanup();
          this.onConnectionChange?.(false);

          // Attempt reconnection if not intentionally closed
          if (event.code !== 1000 && event.code !== 1001) {
            this.scheduleReconnect();
          }
        };

        this.socket.onerror = (event) => {
          console.error("WebSocket error:", event);
          const errorMessage = "WebSocket connection error occurred";
          this.onError?.(errorMessage);
          reject(new Error(errorMessage));
        };
      } catch (error) {
        const errorMessage = `Failed to create WebSocket connection: ${error}`;
        console.error(errorMessage);
        this.onError?.(errorMessage);
        reject(error);
      }
    });
  }

  disconnect(): void {
    this.isReconnecting = false;

    if (this.reconnectTimeout) {
      clearTimeout(this.reconnectTimeout);
      this.reconnectTimeout = null;
    }

    if (this.socket) {
      this.socket.close(1000, "Client disconnecting");
      this.socket = null;
    }

    this.cleanup();
  }

  private cleanup(): void {
    if (this.heartbeatInterval) {
      clearInterval(this.heartbeatInterval);
      this.heartbeatInterval = null;
    }
  }

  private handleMessage(message: WebSocketMessage): void {
    switch (message.type) {
      case "notification":
        this.handleNotification(message.data as NotificationMessage);
        break;
      case "pong":
        // Heartbeat response
        break;
      case "error":
        console.error("WebSocket server error:", message.data);
        this.onError?.(message.data?.message || "Server error");
        break;
      default:
        console.warn("Unknown WebSocket message type:", message.type);
    }

    this.onMessage?.(message);
  }

  private handleNotification(notification: NotificationMessage): void {
    // Validate notification is for current subscription
    if (!this.subscription) return;

    const isForUser =
      notification.channel === "global" ||
      (notification.channel === "user" &&
        notification.targetUserId === this.subscription.userId) ||
      (notification.channel === "role" &&
        notification.targetRoleId &&
        this.subscription.roles.includes(notification.targetRoleId));

    if (isForUser) {
      this.onMessage?.({
        type: "notification",
        data: notification,
        timestamp: new Date().toISOString(),
      });
    }
  }

  private startHeartbeat(): void {
    this.heartbeatInterval = setInterval(() => {
      if (this.socket?.readyState === WebSocket.OPEN) {
        this.sendMessage({
          type: "ping",
          data: { timestamp: Date.now() },
        });
      }
    }, this.config.heartbeatInterval);
  }

  private scheduleReconnect(): void {
    if (this.isReconnecting || !this.subscription) return;

    this.isReconnecting = true;

    this.reconnectTimeout = setTimeout(() => {
      if (this.subscription) {
        console.log("Attempting to reconnect WebSocket...");
        this.connect(this.subscription).catch((error) => {
          console.error("Reconnection failed:", error);
          this.scheduleReconnect();
        });
      }
    }, this.config.reconnectInterval);
  }

  sendMessage(message: WebSocketMessage): void {
    if (this.socket?.readyState === WebSocket.OPEN) {
      this.socket.send(JSON.stringify(message));
    } else {
      console.warn("Cannot send message: WebSocket not connected");
    }
  }

  markNotificationAsRead(notificationId: string): void {
    this.sendMessage({
      type: "mark_read",
      data: { notificationId },
    });
  }

  markNotificationAsDismissed(notificationId: string): void {
    this.sendMessage({
      type: "dismiss",
      data: { notificationId },
    });
  }

  getConnectionState(): {
    isConnected: boolean;
    readyState?: number;
    subscription: WebSocketSubscription | null;
  } {
    return {
      isConnected: this.socket?.readyState === WebSocket.OPEN,
      readyState: this.socket?.readyState,
      subscription: this.subscription,
    };
  }
}

// Singleton instance
export const webSocketService = new WebSocketService();
