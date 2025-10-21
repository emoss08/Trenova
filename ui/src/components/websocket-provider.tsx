/* eslint-disable react-refresh/only-export-components */
/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { useWebSocket } from "@/hooks/use-websocket";

import { useIsAuthenticated } from "@/stores/user-store";

import { createContext, useContext, type ReactNode } from "react";

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
    });

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
