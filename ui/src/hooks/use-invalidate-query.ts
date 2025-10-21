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

const calculateDelay = (attempt: number) =>
  Math.min(INITIAL_RETRY_DELAY * Math.pow(2, attempt), MAX_RETRY_DELAY);

const logDebug = (message: string, color: string = "#a742f5") => {
  if (APP_ENV === "development") {
    console.debug(
      `%c[Trenova] ${message}`,
      `color: ${color}; font-weight: bold`,
    );
  }
};

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
  const initializeChannelRef = useRef<() => Promise<void>>(async () => {});
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
        retryTimeoutRef.current = window.setTimeout(
          () => initializeChannelRef.current?.(),
          delay,
        );
      }
    }
  }, [messageHandler]);

  useEffect(() => {
    initializeChannelRef.current = initializeChannel;
  }, [initializeChannel]);

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

  useEffect(() => {
    const interval = setInterval(() => {
      if (!channelRef.current) return;
    }, 30000);

    return () => clearInterval(interval);
  }, []);
};

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
      setTimeout(() => channel.close(), 1000);
    };

    channel.postMessage(message);
    logDebug(`Broadcasted invalidation for keys: ${queryKey.join(", ")}`);
    cleanup();
  } catch (error) {
    console.error("[Trenova] Broadcast failed:", error);
    throw error;
  }
};
