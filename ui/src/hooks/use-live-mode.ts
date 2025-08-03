/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { API_URL } from "@/constants/env";
import { useCallback, useEffect, useRef, useState } from "react";

export interface LiveModeOptions {
  endpoint: string;
  enabled?: boolean;
  onNewData?: (data: any) => void;
  onError?: (error: string) => void;
  onConnectionChange?: (connected: boolean) => void;
  reconnectDelay?: number;
  maxReconnectDelay?: number;
  maxReconnectAttempts?: number;
}

export interface LiveModeState {
  connected: boolean;
  error: string | null;
  lastHeartbeat: Date | null;
  connectionQuality: "good" | "degraded" | "poor";
}

export function useLiveMode({
  endpoint,
  enabled = false,
  onNewData,
  onError,
  onConnectionChange,
  reconnectDelay = 1000,
  maxReconnectDelay = 30000,
  maxReconnectAttempts = 10,
}: LiveModeOptions) {
  const [state, setState] = useState<LiveModeState>({
    connected: false,
    error: null,
    lastHeartbeat: null,
    connectionQuality: "good",
  });

  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(
    null,
  );
  const heartbeatTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(
    null,
  );
  const reconnectAttempts = useRef(0);
  const isIntentionalDisconnect = useRef(false);
  const lastEventTime = useRef<Date>(new Date());

  const handlersRef = useRef<{
    connected?: (event: MessageEvent) => void;
    newEntry?: (event: MessageEvent) => void;
    heartbeat?: (event: MessageEvent) => void;
    ping?: (event: MessageEvent) => void;
    error?: (event: MessageEvent) => void;
  }>({});

  const cleanup = useCallback(() => {
    if (eventSourceRef.current) {
      const es = eventSourceRef.current;
      const handlers = handlersRef.current;
      if (handlers.connected) {
        es.removeEventListener("connected", handlers.connected);
      }
      if (handlers.newEntry) {
        es.removeEventListener("new-entry", handlers.newEntry);
      }
      if (handlers.heartbeat) {
        es.removeEventListener("heartbeat", handlers.heartbeat);
      }
      if (handlers.ping) {
        es.removeEventListener("ping", handlers.ping);
      }
      if (handlers.error) {
        es.removeEventListener("error", handlers.error);
      }
      es.close();
      eventSourceRef.current = null;
    }
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (heartbeatTimeoutRef.current) {
      clearTimeout(heartbeatTimeoutRef.current);
      heartbeatTimeoutRef.current = null;
    }
  }, []);

  const updateConnectionQuality = useCallback(() => {
    const now = new Date();
    const timeSinceLastEvent = now.getTime() - lastEventTime.current.getTime();

    let quality: "good" | "degraded" | "poor" = "good";
    if (timeSinceLastEvent > 60000) {
      quality = "poor";
    } else if (timeSinceLastEvent > 35000) {
      quality = "degraded";
    }

    setState((prev) => {
      if (prev.connectionQuality !== quality) {
        return { ...prev, connectionQuality: quality };
      }
      return prev;
    });
  }, []);

  const startHeartbeatMonitor = useCallback(() => {
    if (heartbeatTimeoutRef.current) {
      clearTimeout(heartbeatTimeoutRef.current);
    }
    heartbeatTimeoutRef.current = setTimeout(() => {
      setState((prev) => ({ ...prev, connectionQuality: "poor" }));
    }, 45000);
  }, []);

  const resetHeartbeatMonitor = useCallback(() => {
    startHeartbeatMonitor();
  }, [startHeartbeatMonitor]);

  const connect = useCallback(() => {
    if (!enabled || eventSourceRef.current) return;

    try {
      const url = `${API_URL}${endpoint}`;
      const eventSource = new EventSource(url, {
        withCredentials: true,
      });

      eventSourceRef.current = eventSource;

      eventSource.onopen = () => {
        setState((prev) => ({ ...prev, connected: true, error: null }));
        onConnectionChange?.(true);
        reconnectAttempts.current = 0;
      };

      eventSource.onerror = () => {
        // ! SSE error events don't have useful error information
        // ! Check readyState to determine the actual state
        if (eventSource.readyState === EventSource.CLOSED) {
          // ! Check if this is likely a page unload/navigation event
          const isPageUnloading =
            document.readyState === "loading" ||
            window.performance.navigation.type === 1; // Page reload

          if (isPageUnloading) {
            // ! Don't log errors or attempt reconnection during page unload
            cleanup();
            return;
          }

          setState((prev) => ({ ...prev, connected: false }));
          onConnectionChange?.(false);

          // ! Only attempt reconnection if it wasn't an intentional disconnect
          if (
            !isIntentionalDisconnect.current &&
            reconnectAttempts.current < maxReconnectAttempts
          ) {
            const delay = Math.min(
              reconnectDelay * Math.pow(2, reconnectAttempts.current),
              maxReconnectDelay,
            );
            reconnectTimeoutRef.current = setTimeout(() => {
              reconnectAttempts.current++;
              cleanup();
              connect();
            }, delay);
          } else if (reconnectAttempts.current >= maxReconnectAttempts) {
            const errorMsg =
              "Failed to connect to live updates after multiple attempts";
            setState((prev) => ({ ...prev, error: errorMsg }));
            onError?.(errorMsg);
          }
        } else if (eventSource.readyState === EventSource.CONNECTING) {
          setState((prev) => ({
            ...prev,
            connected: false,
            connectionQuality: "degraded",
          }));
        }
      };

      const handleConnected = () => {
        lastEventTime.current = new Date();
        startHeartbeatMonitor();
      };

      const handleNewEntry = (event: MessageEvent) => {
        lastEventTime.current = new Date();
        updateConnectionQuality();
        try {
          const data = JSON.parse(event.data);
          onNewData?.(data);
        } catch (error) {
          console.error("Failed to parse new entry data:", error);
        }
      };

      const handleHeartbeat = (event: MessageEvent) => {
        lastEventTime.current = new Date();
        updateConnectionQuality();
        try {
          const data = JSON.parse(event.data);
          setState((prev) => ({
            ...prev,
            lastHeartbeat: new Date(data.timestamp * 1000),
            connectionQuality: "good",
          }));
        } catch (error) {
          console.error("Failed to parse heartbeat data:", error);
        }
        resetHeartbeatMonitor();
      };

      const handlePing = () => {
        lastEventTime.current = new Date();
        updateConnectionQuality();
        resetHeartbeatMonitor();
      };

      const handleServerError = (event: MessageEvent) => {
        try {
          if (!event.data) {
            console.warn("Received error event without data");
            return;
          }

          const data = JSON.parse(event.data);
          const error = data.error || "Unknown server error";
          setState((prev) => ({ ...prev, error }));
          onError?.(error);
        } catch (error) {
          console.error("Failed to parse error data:", error);
        }
      };

      handlersRef.current = {
        connected: handleConnected,
        newEntry: handleNewEntry,
        heartbeat: handleHeartbeat,
        ping: handlePing,
        error: handleServerError,
      };

      eventSource.addEventListener("connected", handleConnected);
      eventSource.addEventListener("new-entry", handleNewEntry);
      eventSource.addEventListener("heartbeat", handleHeartbeat);
      eventSource.addEventListener("ping", handlePing);
      eventSource.addEventListener("error", handleServerError);
    } catch (error) {
      const errorMessage =
        error instanceof Error
          ? error.message
          : "Failed to establish connection";
      setState((prev) => ({ ...prev, error: errorMessage, connected: false }));
      onError?.(errorMessage);
    }
  }, [
    enabled,
    endpoint,
    onNewData,
    onError,
    onConnectionChange,
    cleanup,
    maxReconnectAttempts,
    reconnectDelay,
    maxReconnectDelay,
    startHeartbeatMonitor,
    updateConnectionQuality,
    resetHeartbeatMonitor,
  ]);

  const disconnect = useCallback(() => {
    isIntentionalDisconnect.current = true;
    cleanup();
    setState({
      connected: false,
      error: null,
      lastHeartbeat: null,
      connectionQuality: "good",
    });
    onConnectionChange?.(false);
    reconnectAttempts.current = 0;
  }, [cleanup, onConnectionChange]);

  useEffect(() => {
    if (enabled) {
      isIntentionalDisconnect.current = false;
      connect();
    } else {
      disconnect();
    }

    return () => {
      isIntentionalDisconnect.current = true;
      cleanup();
    };
  }, [enabled, connect, disconnect, cleanup]);

  useEffect(() => {
    if (!enabled) return;

    const handleVisibilityChange = () => {
      if (!document.hidden) {
        updateConnectionQuality();
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [enabled, updateConnectionQuality]);

  useEffect(() => {
    const handleBeforeUnload = () => {
      isIntentionalDisconnect.current = true;
      cleanup();
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [cleanup]);

  return {
    ...state,
    connect,
    disconnect,
    reconnectAttempts: reconnectAttempts.current,
  };
}
