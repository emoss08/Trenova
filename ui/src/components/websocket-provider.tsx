import { APP_ENV } from "@/constants/env";
import { useWebSocket } from "@/hooks/use-websocket";
import { useIsAuthenticated } from "@/stores/user-store";
import { createContext, useContext, useEffect, type ReactNode } from "react";

interface WebSocketContextValue {
  isConnected: boolean;
  connect: () => void;
  disconnect: () => void;
  markAsRead: (notificationId: string) => void;
  markAsDismissed: (notificationId: string) => void;
}

const WebSocketContext = createContext<WebSocketContextValue | null>(null);

export function useWebSocketContext() {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error(
      "useWebSocketContext must be used within WebSocketProvider",
    );
  }
  return context;
}

interface WebSocketProviderProps {
  children: ReactNode;
}

export function WebSocketProvider({ children }: WebSocketProviderProps) {
  const isAuthenticated = useIsAuthenticated();

  const { connect, disconnect, markAsRead, markAsDismissed, connectionState } =
    useWebSocket({
      enabled: isAuthenticated,
      onMessage: (message) => {
        if (APP_ENV === "development") {
          console.log("📨 WebSocket message received:", message);
        }
      },
      onError: (error) => {
        console.error("❌ WebSocket error:", error);
      },
    });

  // Debug connection state changes
  useEffect(() => {
    if (APP_ENV === "development") {
      console.log("🔌 WebSocket connection state:", connectionState);
    }
  }, [connectionState]);

  const contextValue: WebSocketContextValue = {
    isConnected: connectionState.isConnected,
    connect,
    disconnect,
    markAsRead,
    markAsDismissed,
  };

  return (
    <WebSocketContext.Provider value={contextValue}>
      {children}
    </WebSocketContext.Provider>
  );
}
