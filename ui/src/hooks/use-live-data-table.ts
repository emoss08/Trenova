import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
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

  const handleNewData = useCallback(
    (data: any) => {
      // Add the new item ID to the set for highlighting
      if (data.id) {
        console.log("🆕 Adding new item to highlight set:", data.id);
        setNewItemIds((prev) => {
          const newSet = new Set(prev);
          newSet.add(data.id);
          console.log("📝 Current highlight set:", Array.from(newSet));
          return newSet;
        });

        // Remove the ID after 3 seconds to clear the highlight
        setTimeout(() => {
          console.log("⏰ Removing highlight for:", data.id);
          setNewItemIds((prev) => {
            const newSet = new Set(prev);
            newSet.delete(data.id);
            console.log("📝 Updated highlight set:", Array.from(newSet));
            return newSet;
          });
        }, 3000);
      }

      if (autoRefresh) {
        // Auto-refresh: immediately invalidate and refetch the query
        queryClient.invalidateQueries({ queryKey: [queryKey] });
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
    // Invalidate and refetch the query
    queryClient.invalidateQueries({ queryKey: [queryKey] });
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

  return {
    ...liveMode,
    newItemsCount,
    showNewItemsBanner,
    refreshData,
    dismissBanner,
    isNewItem,
  };
}
