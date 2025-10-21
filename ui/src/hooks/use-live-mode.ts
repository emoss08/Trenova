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
  const connectRef = useRef<(() => void) | null>(null);

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
    if (!enabled || eventSourceRef.current) {
      console.log(
        "[LiveMode] Connect skipped - enabled:",
        enabled,
        "existing connection:",
        !!eventSourceRef.current,
      );
      return;
    }

    try {
      const url = `${API_URL}${endpoint}`;
      console.log("[LiveMode] Attempting to connect to:", url);
      console.log("[LiveMode] API_URL:", API_URL);
      console.log("[LiveMode] Endpoint:", endpoint);
      console.log("[LiveMode] Full URL:", url);
      console.log("[LiveMode] withCredentials: true");

      const eventSource = new EventSource(url, {
        withCredentials: true,
      });

      console.log(
        "[LiveMode] EventSource created, readyState:",
        eventSource.readyState,
      );
      console.log("[LiveMode] EventSource.CONNECTING:", EventSource.CONNECTING);
      console.log("[LiveMode] EventSource.OPEN:", EventSource.OPEN);
      console.log("[LiveMode] EventSource.CLOSED:", EventSource.CLOSED);

      eventSourceRef.current = eventSource;

      eventSource.onopen = () => {
        console.log(
          "[LiveMode] ✅ Connection opened! readyState:",
          eventSource.readyState,
        );
        setState((prev) => ({ ...prev, connected: true, error: null }));
        onConnectionChange?.(true);
        reconnectAttempts.current = 0;
      };

      eventSource.onerror = (e) => {
        console.error("[LiveMode] ❌ EventSource error event:", e);
        console.error("[LiveMode] ReadyState:", eventSource.readyState);
        console.error("[LiveMode] URL:", eventSource.url);

        // ! SSE error events don't have useful error information
        // ! Check readyState to determine the actual state
        if (eventSource.readyState === EventSource.CLOSED) {
          console.error("[LiveMode] Connection closed - readyState is CLOSED");
          // ! Check if this is likely a page unload/navigation event
          const isPageUnloading =
            document.readyState === "loading" ||
            window.performance.navigation.type === 1; // Page reload

          if (isPageUnloading) {
            console.log("[LiveMode] Page unloading - skipping reconnect");
            // ! Don't log errors or attempt reconnection during page unload
            cleanup();
            return;
          }

          console.error("[LiveMode] Connection failed - not page unloading");
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
            console.log(
              `[LiveMode] Reconnecting in ${delay}ms (attempt ${reconnectAttempts.current + 1}/${maxReconnectAttempts})`,
            );
            reconnectTimeoutRef.current = setTimeout(() => {
              reconnectAttempts.current++;
              cleanup();
              connectRef.current?.();
            }, delay);
          } else if (reconnectAttempts.current >= maxReconnectAttempts) {
            const errorMsg =
              "Failed to connect to live updates after multiple attempts";
            console.error("[LiveMode]", errorMsg);
            setState((prev) => ({ ...prev, error: errorMsg }));
            onError?.(errorMsg);
          }
        } else if (eventSource.readyState === EventSource.CONNECTING) {
          console.log(
            "[LiveMode] Currently connecting... readyState: CONNECTING",
          );
          setState((prev) => ({
            ...prev,
            connected: false,
            connectionQuality: "degraded",
          }));
        }
      };

      const handleConnected = (event: MessageEvent) => {
        console.log("[LiveMode] ✅ 'connected' event received:", event.data);
        lastEventTime.current = new Date();
        startHeartbeatMonitor();
      };

      const handleNewEntry = (event: MessageEvent) => {
        console.log("[LiveMode] 📦 'new-entry' event received:", event.data);
        lastEventTime.current = new Date();
        updateConnectionQuality();
        try {
          const data = JSON.parse(event.data);
          onNewData?.(data);
        } catch (error) {
          console.error("[LiveMode] Failed to parse new entry data:", error);
        }
      };

      const handleHeartbeat = (event: MessageEvent) => {
        console.log("[LiveMode] 💓 'heartbeat' event received:", event.data);
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
          console.error("[LiveMode] Failed to parse heartbeat data:", error);
        }
        resetHeartbeatMonitor();
      };

      const handlePing = (event: MessageEvent) => {
        console.log("[LiveMode] 🏓 'ping' event received:", event.data);
        lastEventTime.current = new Date();
        updateConnectionQuality();
        resetHeartbeatMonitor();
      };

      const handleServerError = (event: MessageEvent) => {
        console.error("[LiveMode] ⚠️ Server error event received:", event);
        try {
          if (!event.data) {
            console.warn("[LiveMode] Received error event without data");
            return;
          }

          const data = JSON.parse(event.data);
          const error = data.error || "Unknown server error";
          console.error("[LiveMode] Server error:", error);
          setState((prev) => ({ ...prev, error }));
          onError?.(error);
        } catch (error) {
          console.error("[LiveMode] Failed to parse error data:", error);
        }
      };

      handlersRef.current = {
        connected: handleConnected,
        newEntry: handleNewEntry,
        heartbeat: handleHeartbeat,
        ping: handlePing,
        error: handleServerError,
      };

      console.log("[LiveMode] Registering event listeners...");
      eventSource.addEventListener("connected", handleConnected);
      eventSource.addEventListener("new-entry", handleNewEntry);
      eventSource.addEventListener("heartbeat", handleHeartbeat);
      eventSource.addEventListener("ping", handlePing);
      eventSource.addEventListener("error", handleServerError);
      console.log("[LiveMode] Event listeners registered");
    } catch (error) {
      console.error("[LiveMode] Exception during connect:", error);
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
    connectRef.current = connect;
  });

  useEffect(() => {
    console.log("[LiveMode] Effect triggered - enabled:", enabled);
    if (enabled) {
      console.log("[LiveMode] Enabled is true, attempting to connect...");
      isIntentionalDisconnect.current = false;
      connect();
    } else {
      console.log("[LiveMode] Enabled is false, disconnecting...");
      disconnect();
    }

    return () => {
      console.log("[LiveMode] Effect cleanup");
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
      console.log("Before unload");
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
  };
}
