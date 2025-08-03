/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

/**
 * Query Invalidation System
 *
 * This module provides a robust system for invalidating React Query cache entries across browser tabs
 * using the BroadcastChannel API. It supports two invalidation strategies:
 *
 * 1. Predicate-based Invalidation:
 *    - Uses pattern matching to invalidate queries based on partial key matches
 *    - Useful when you need to invalidate multiple related queries at once
 *    - Example: Invalidating all queries that contain "user" in their key:
 *      - "users.list" would match
 *      - "user.details.123" would match
 *      - "settings" would not match
 *
 * 2. Direct Key Invalidation:
 *    - Invalidates queries using exact query key matching
 *    - More precise control over which queries are invalidated
 *    - Example: Invalidating ["users", "list"] would only match that exact query key
 *
 * Usage Examples:
 *
 * ```typescript
 * Predicate-based invalidation (pattern matching)
 *
 * broadcastQueryInvalidation(['user'], {
 *   config: {
 *     predicate: true, // Enable pattern matching
 *     refetchType: 'all'
 *   }
 * });
 *
 * This would invalidate queries with keys like:
 * - ['user']
 * - ['users', 'list']
 * - ['user', '123', 'details']
 *
 * Direct key invalidation
 * broadcastQueryInvalidation([['users', 'list']], {
 *   config: {
 *     predicate: false, // Disable pattern matching
 *     exact: true
 *   }
 * });
 *
 * This would only invalidate the query with exactly ['users', 'list'] as its key
 * ```
 */

import { APP_ENV } from "@/constants/env";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef } from "react";

interface InvalidationConfig {
  exact?: boolean;
  refetchType?: "active" | "inactive" | "all";
  predicate?: boolean;
}

interface InvalidationMessage {
  type: "invalidate";
  queryKeys: string[];
  config?: InvalidationConfig;
  correlationId?: string;
}

type QueryMessage = InvalidationMessage;
type MessageHandler = (message: QueryMessage) => Promise<void>;

const CHANNEL_NAME = "query-invalidation";
const MAX_RETRY_ATTEMPTS = 5;
const INITIAL_RETRY_DELAY = 1000;
const MAX_RETRY_DELAY = 10000; // 10 seconds

// Helper for exponential backoff
const calculateDelay = (attempt: number) =>
  Math.min(INITIAL_RETRY_DELAY * Math.pow(2, attempt), MAX_RETRY_DELAY);

// Debug logger with environment check
const logDebug = (message: string, color: string = "#a742f5") => {
  if (APP_ENV === "development") {
    console.debug(
      `%c[Trenova] ${message}`,
      `color: ${color}; font-weight: bold`,
    );
  }
};

// Type guard for invalidation messages
const isInvalidationMessage = (data: unknown): data is InvalidationMessage =>
  typeof data === "object" &&
  data !== null &&
  "type" in data &&
  data.type === "invalidate" &&
  "queryKeys" in data &&
  Array.isArray(data.queryKeys);

export const useQueryInvalidationListener = () => {
  const queryClient = useQueryClient();
  const channelRef = useRef<BroadcastChannel | null>(null);
  const retryTimeoutRef = useRef<number>(0);
  const abortControllerRef = useRef<AbortController | null>(null);
  const retryAttemptRef = useRef(0);
  const handleInvalidation: MessageHandler = useCallback(
    async (message) => {
      try {
        logDebug(
          `Processing invalidation for keys: ${message.queryKeys.join(", ")}`,
          "#87f542",
        );

        const queryKeys = Array.isArray(message.queryKeys)
          ? message.queryKeys
          : [message.queryKeys];

        const config = message.config || {};
        if (config.predicate) {
          // Use predicate-based invalidation
          queryKeys.forEach((keyPattern) => {
            queryClient.invalidateQueries({
              predicate: (query) =>
                query.queryKey.some(
                  (keyPart) =>
                    typeof keyPart === "string" &&
                    keyPart.includes(String(keyPattern)),
                ),
              refetchType: config.refetchType || "all",
              exact: config.exact || false,
            });
          });
        } else {
          // Use direct key invalidation
          await Promise.all(
            queryKeys.map(async (queryKey) => {
              if (!queryKey) return;

              await queryClient.invalidateQueries({
                queryKey: [queryKey],
                exact: config.exact ?? false,
                refetchType: config.refetchType || "all",
              });
            }),
          );
        }

        logDebug(`Successfully invalidated queries`, "#4caf50");
      } catch (error) {
        console.error("[Trenova] Query invalidation failed:", error);
        // Consider adding retry logic for failed invalidations here
      }
    },
    [queryClient],
  );

  const messageHandler = useCallback(
    async (event: MessageEvent) => {
      try {
        if (!isInvalidationMessage(event.data)) return;

        logDebug(
          `Received invalidation message: ${event.data.queryKeys.join(", ")}`,
        );
        await handleInvalidation(event.data);
      } catch (error) {
        console.error("[Trenova] Message handling failed:", error);
      }
    },
    [handleInvalidation],
  );

  const initializeChannel = useCallback(async () => {
    abortControllerRef.current?.abort();
    abortControllerRef.current = new AbortController();

    try {
      if (channelRef.current) return;

      channelRef.current = new BroadcastChannel(CHANNEL_NAME);
      channelRef.current.addEventListener("message", messageHandler);
      retryAttemptRef.current = 0;

      logDebug("Broadcast channel initialized", "#42a5f5");
    } catch (error) {
      console.error("[Trenova] Channel initialization error:", error);

      if (retryAttemptRef.current < MAX_RETRY_ATTEMPTS) {
        const delay = calculateDelay(retryAttemptRef.current);
        retryAttemptRef.current += 1;

        logDebug(
          `Retrying channel initialization (attempt ${retryAttemptRef.current})`,
        );
        retryTimeoutRef.current = window.setTimeout(initializeChannel, delay);
      }
    }
  }, [messageHandler]);

  useEffect(() => {
    initializeChannel();

    return () => {
      abortControllerRef.current?.abort();
      clearTimeout(retryTimeoutRef.current);

      if (channelRef.current) {
        channelRef.current.removeEventListener("message", messageHandler);
        channelRef.current.close();
        channelRef.current = null;
      }
    };
  }, [initializeChannel, messageHandler]);

  // Add heartbeat/ping-pong mechanism if needed
  useEffect(() => {
    const interval = setInterval(() => {
      if (!channelRef.current) return;
      // Optional: Implement health check here
    }, 30000);

    return () => clearInterval(interval);
  }, []);
};

// Enhanced broadcast function with transaction tracking
export const broadcastQueryInvalidation = async ({
  queryKey,
  config,
  options,
}: {
  queryKey: string[];
  config?: InvalidationConfig;
  options?: { correlationId?: string };
}) => {
  try {
    const channel = new BroadcastChannel(CHANNEL_NAME);
    const message: InvalidationMessage = {
      type: "invalidate",
      queryKeys: queryKey,
      config,
      correlationId: options?.correlationId || crypto.randomUUID(),
    };

    const cleanup = () => {
      setTimeout(() => channel.close(), 1000); // Allow time for message transmission
    };

    channel.postMessage(message);
    logDebug(`Broadcasted invalidation for keys: ${queryKey.join(", ")}`);
    cleanup();
  } catch (error) {
    console.error("[Trenova] Broadcast failed:", error);
    throw error;
  }
};
