/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

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

  // Refs for batching and debouncing
  const pendingEventsRef = useRef<any[]>([]);
  const batchTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const debounceTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Process batched events
  const processBatchedEvents = useCallback(() => {
    const events = pendingEventsRef.current;
    if (events.length === 0) return;

    // Clear pending events
    pendingEventsRef.current = [];

    // Process all IDs for highlighting
    const newIds = new Set<string>();
    events.forEach((event) => {
      if (event.id || event.shipment?.id) {
        const itemId = event.id || event.shipment?.id;
        newIds.add(itemId);

        // Clear any existing timeout for this ID
        const existingTimeout = timeoutRefs.current.get(itemId);
        if (existingTimeout) {
          clearTimeout(existingTimeout);
        }

        // Remove the ID after 3 seconds to clear the highlight
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

    // Update all new IDs at once
    setNewItemIds((prev) => {
      const newSet = new Set(prev);
      newIds.forEach((id) => newSet.add(id));
      return newSet;
    });

    if (autoRefresh) {
      // Clear any existing debounce timeout
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current);
      }

      // Debounce the query invalidation
      debounceTimeoutRef.current = setTimeout(() => {
        queryClient.invalidateQueries({
          queryKey: [queryKey],
          type: "active",
          exact: false,
        });
      }, debounceDelay);
    } else {
      // Banner mode: update count with batch size
      setNewItemsCount((prev) => prev + events.length);
      setShowNewItemsBanner(true);
    }

    // Call custom handlers for each event
    events.forEach((event) => onNewData?.(event));
  }, [autoRefresh, queryClient, queryKey, onNewData, debounceDelay]);

  const handleNewData = useCallback(
    (data: any) => {
      // Add event to pending batch
      pendingEventsRef.current.push(data);

      // If no batch timeout is running, start one
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
    // Invalidate and refetch the query with more specific targeting
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

  // Cleanup timeouts on unmount
  useEffect(() => {
    return () => {
      // Clear all highlight timeouts
      timeoutRefs.current.forEach((timeout) => clearTimeout(timeout));
      timeoutRefs.current.clear();

      // Clear batch and debounce timeouts
      if (batchTimeoutRef.current) {
        clearTimeout(batchTimeoutRef.current);
      }
      if (debounceTimeoutRef.current) {
        clearTimeout(debounceTimeoutRef.current);
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
