import { API_URL } from "@/constants/env";
import { useCallback, useEffect, useRef, useState } from "react";

export interface LiveModeOptions {
  endpoint: string;
  enabled?: boolean;
  onNewData?: (data: any) => void;
  onError?: (error: string) => void;
  onConnectionChange?: (connected: boolean) => void;
}

export interface LiveModeState {
  connected: boolean;
  error: string | null;
  lastHeartbeat: Date | null;
}

export function useLiveMode({
  endpoint,
  enabled = false,
  onNewData,
  onError,
  onConnectionChange,
}: LiveModeOptions) {
  const [state, setState] = useState<LiveModeState>({
    connected: false,
    error: null,
    lastHeartbeat: null,
  });

  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(
    null,
  );
  const reconnectAttempts = useRef(0);
  const maxReconnectAttempts = 5;

  const cleanup = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
      eventSourceRef.current = null;
    }
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
  }, []);

  const connect = useCallback(() => {
    if (!enabled || eventSourceRef.current) return;

    try {
      const url = `${API_URL}${endpoint}`;
      console.log("🔗 Connecting to live mode:", url);
      const eventSource = new EventSource(url, {
        withCredentials: true,
      });

      eventSourceRef.current = eventSource;

      eventSource.onopen = () => {
        console.log("✅ Live mode connected successfully");
        setState((prev) => ({ ...prev, connected: true, error: null }));
        onConnectionChange?.(true);
        reconnectAttempts.current = 0;
      };

      eventSource.onerror = (error) => {
        console.log("❌ Live mode connection error:", error);
        console.log("EventSource readyState:", eventSource.readyState);
        setState((prev) => ({ ...prev, connected: false }));
        onConnectionChange?.(false);

        if (reconnectAttempts.current < maxReconnectAttempts) {
          const delay = Math.min(
            1000 * Math.pow(2, reconnectAttempts.current),
            10000,
          );
          console.log(
            `🔄 Retrying connection in ${delay}ms (attempt ${reconnectAttempts.current + 1}/${maxReconnectAttempts})`,
          );
          reconnectTimeoutRef.current = setTimeout(() => {
            reconnectAttempts.current++;
            cleanup();
            connect();
          }, delay);
        } else {
          const errorMsg =
            "Failed to connect to live updates after multiple attempts";
          console.log("💀 Max reconnection attempts reached");
          setState((prev) => ({ ...prev, error: errorMsg }));
          onError?.(errorMsg);
        }
      };

      // Handle different event types
      eventSource.addEventListener("connected", (event: MessageEvent) => {
        console.log("🟢 Live mode connected event:", event.data);
      });

      eventSource.addEventListener("new-entry", (event: MessageEvent) => {
        console.log("📨 New entry received:", event.data);
        try {
          const data = JSON.parse(event.data);
          onNewData?.(data);
        } catch (error) {
          console.error("Failed to parse new entry data:", error);
        }
      });

      eventSource.addEventListener("heartbeat", (event: MessageEvent) => {
        console.log("💓 Heartbeat received:", event.data);
        try {
          const data = JSON.parse(event.data);
          setState((prev) => ({
            ...prev,
            lastHeartbeat: new Date(data.timestamp),
          }));
        } catch (error) {
          console.error("Failed to parse heartbeat data:", error);
        }
      });

      eventSource.addEventListener("error", (event: MessageEvent) => {
        console.log("🚨 Server error event:", event.data);
        try {
          const data = JSON.parse(event.data);
          const error = data.error || "Unknown server error";
          setState((prev) => ({ ...prev, error }));
          onError?.(error);
        } catch (error) {
          console.error("Failed to parse error data:", error);
        }
      });
    } catch (error) {
      const errorMessage =
        error instanceof Error
          ? error.message
          : "Failed to establish connection";
      setState((prev) => ({ ...prev, error: errorMessage, connected: false }));
      onError?.(errorMessage);
    }
  }, [enabled, endpoint, onNewData, onError, onConnectionChange, cleanup]);

  const disconnect = useCallback(() => {
    cleanup();
    setState({
      connected: false,
      error: null,
      lastHeartbeat: null,
    });
    onConnectionChange?.(false);
  }, [cleanup, onConnectionChange]);

  // Connect/disconnect based on enabled state
  useEffect(() => {
    if (enabled) {
      connect();
    } else {
      disconnect();
    }

    return () => {
      cleanup();
    };
  }, [enabled, connect, disconnect, cleanup]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      cleanup();
    };
  }, [cleanup]);

  return {
    ...state,
    connect,
    disconnect,
  };
}
