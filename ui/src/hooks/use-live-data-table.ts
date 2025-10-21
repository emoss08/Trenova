import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { useLiveMode } from "./use-live-mode";

export interface LiveDataTableOptions {
  queryKey: string;
  endpoint: string;
  enabled?: boolean;
  autoRefresh?: boolean; // Auto-refresh on new data instead of showing banner
  onNewData?: (data: any) => void;
  batchWindow?: number; // Time window in ms to batch events (default: 100ms)
  debounceDelay?: number; // Debounce delay for auto-refresh (default: 300ms)
}

export function useLiveDataTable({
  queryKey,
  endpoint,
  enabled = false,
  autoRefresh = false,
  onNewData,
  batchWindow = 100,
  debounceDelay = 300,
}: LiveDataTableOptions) {
  const queryClient = useQueryClient();
  const [newItemsCount, setNewItemsCount] = useState(0);
  const [showNewItemsBanner, setShowNewItemsBanner] = useState(false);
  const [newItemIds, setNewItemIds] = useState<Set<string>>(new Set());
  const timeoutRefs = useRef<Map<string, ReturnType<typeof setTimeout>>>(
    new Map(),
  );

  const pendingEventsRef = useRef<any[]>([]);
  const batchTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const debounceTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const processBatchedEvents = useCallback(() => {
    const events = pendingEventsRef.current;
    if (events.length === 0) return;

    pendingEventsRef.current = [];

    const newIds = new Set<string>();
    events.forEach((event) => {
      if (event.id || event.shipment?.id) {
        const itemId = event.id || event.shipment?.id;
        newIds.add(itemId);

        const existingTimeout = timeoutRefs.current.get(itemId);
        if (existingTimeout) {
          clearTimeout(existingTimeout);
        }

        const timeout = setTimeout(() => {
          setNewItemIds((prev) => {
            const newSet = new Set(prev);
            newSet.delete(itemId);
            return newSet;
          });
          timeoutRefs.current.delete(itemId);
        }, 3000);

        timeoutRefs.current.set(itemId, timeout);
      }
    });

    setNewItemIds((prev) => {
      const newSet = new Set(prev);
      newIds.forEach((id) => newSet.add(id));
      return newSet;
    });

    if (autoRefresh) {
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current);
      }

      debounceTimeoutRef.current = setTimeout(() => {
        queryClient.invalidateQueries({
          queryKey: [queryKey],
          type: "active",
          exact: false,
        });
      }, debounceDelay);
    } else {
      setNewItemsCount((prev) => prev + events.length);
      setShowNewItemsBanner(true);
    }

    events.forEach((event) => onNewData?.(event));
  }, [autoRefresh, queryClient, queryKey, onNewData, debounceDelay]);

  const handleNewData = useCallback(
    (data: any) => {
      pendingEventsRef.current.push(data);

      if (!batchTimeoutRef.current) {
        batchTimeoutRef.current = setTimeout(() => {
          processBatchedEvents();
          batchTimeoutRef.current = null;
        }, batchWindow);
      }
    },
    [batchWindow, processBatchedEvents],
  );

  const handleError = useCallback((error: string) => {
    console.error("Live mode error:", error);
  }, []);

  const liveMode = useLiveMode({
    endpoint,
    enabled,
    onNewData: handleNewData,
    onError: handleError,
  });

  const refreshData = useCallback(() => {
    queryClient.invalidateQueries({
      queryKey: [queryKey],
      type: "active",
      exact: false,
    });
    setNewItemsCount(0);
    setShowNewItemsBanner(false);
  }, [queryClient, queryKey]);

  const dismissBanner = useCallback(() => {
    setShowNewItemsBanner(false);
    setNewItemsCount(0);
  }, []);

  const isNewItem = useCallback(
    (itemId: string) => {
      const isNew = newItemIds.has(itemId);
      return isNew;
    },
    [newItemIds],
  );

  useEffect(() => {
    const currentTimeoutRefs = timeoutRefs.current;
    const currentBatchTimeout = batchTimeoutRef.current;
    const currentDebounceTimeout = debounceTimeoutRef.current;

    return () => {
      currentTimeoutRefs.forEach((timeout) => clearTimeout(timeout));
      currentTimeoutRefs.clear();

      if (currentBatchTimeout) {
        clearTimeout(currentBatchTimeout);
      }
      if (currentDebounceTimeout) {
        clearTimeout(currentDebounceTimeout);
      }
    };
  }, []);

  return {
    ...liveMode,
    newItemsCount,
    showNewItemsBanner,
    refreshData,
    dismissBanner,
    isNewItem,
  };
}
