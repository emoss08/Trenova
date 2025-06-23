import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useRef, useState } from "react";
import { useLiveMode } from "./use-live-mode";

export interface LiveDataTableOptions {
  queryKey: string;
  endpoint: string;
  enabled?: boolean;
  autoRefresh?: boolean; // Auto-refresh on new data instead of showing banner
  onNewData?: (data: any) => void;
}

export function useLiveDataTable({
  queryKey,
  endpoint,
  enabled = false,
  autoRefresh = false,
  onNewData,
}: LiveDataTableOptions) {
  const queryClient = useQueryClient();
  const [newItemsCount, setNewItemsCount] = useState(0);
  const [showNewItemsBanner, setShowNewItemsBanner] = useState(false);
  const [newItemIds, setNewItemIds] = useState<Set<string>>(new Set());
  const timeoutRefs = useRef<Map<string, ReturnType<typeof setTimeout>>>(
    new Map(),
  );

  const handleNewData = useCallback(
    (data: any) => {
      // Add the new item ID to the set for highlighting
      if (data.id) {
        setNewItemIds((prev) => {
          const newSet = new Set(prev);
          newSet.add(data.id);
          return newSet;
        });

        // Clear any existing timeout for this ID
        const existingTimeout = timeoutRefs.current.get(data.id);
        if (existingTimeout) {
          clearTimeout(existingTimeout);
        }

        // Remove the ID after 3 seconds to clear the highlight
        const timeout = setTimeout(() => {
          setNewItemIds((prev) => {
            const newSet = new Set(prev);
            newSet.delete(data.id);
            return newSet;
          });
          timeoutRefs.current.delete(data.id);
        }, 3000);

        timeoutRefs.current.set(data.id, timeout);
      }

      if (autoRefresh) {
        // Auto-refresh: immediately invalidate and refetch the query
        // Use more specific invalidation to reduce impact on other components
        queryClient.invalidateQueries({
          queryKey: [queryKey],
          type: "active",
          exact: false,
        });
      } else {
        // Banner mode: increment the count of new items
        setNewItemsCount((prev) => prev + 1);
        setShowNewItemsBanner(true);
      }

      // Call custom handler if provided
      onNewData?.(data);
    },
    [autoRefresh, queryClient, queryKey, onNewData],
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
      if (isNew) {
        console.log("✨ Item is highlighted:", itemId);
      }
      return isNew;
    },
    [newItemIds],
  );

  // Cleanup timeouts on unmount
  useEffect(() => {
    return () => {
      timeoutRefs.current.forEach((timeout) => clearTimeout(timeout));
      timeoutRefs.current.clear();
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
