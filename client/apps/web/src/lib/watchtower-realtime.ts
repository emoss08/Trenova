import { fetchWatchtowerItem, type WatchtowerItem } from "@/lib/graphql/watchtower";
import { attention } from "@/lib/queries/attention";
import { watchtower } from "@/lib/queries/watchtower";
import {
  applyNoticeToQueue,
  cachedQueueItem,
  cachedQueueItemsForRun,
  placeInQueue,
  removeFromQueue,
} from "@/lib/watchtower-cache";
import type { QueryClient } from "@tanstack/react-query";
import {
  isBulkAction,
  resolveEntityID,
  type ResourceInvalidationEvent,
} from "@trenova/shared/hooks/realtime-patching";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";

export const WATCHTOWER_RESOURCE = "watchtower";
const AGENT_RUN_RESOURCE = "agent_run";

/** Notices for one item inside this window are read once. */
const COALESCE_MS = 250;

export function isWatchtowerEvent(event: ResourceInvalidationEvent): boolean {
  return event.resource === WATCHTOWER_RESOURCE;
}

export type WatchtowerPatcherOptions = {
  queryClient: QueryClient;
  fetchItem?: (id: string) => Promise<WatchtowerItem>;
  coalesceMs?: number;
};

export type WatchtowerPatcher = {
  /** A watchtower notice: the item it names is read and placed. */
  handle: (event: ResourceInvalidationEvent) => void;
  /** An agent run moved: the queue rows it is working are read again. */
  handleRun: (event: ResourceInvalidationEvent) => void;
  dispose: () => void;
};

function gone(error: unknown): boolean {
  return (
    error instanceof GraphQLRequestError &&
    (error.isNotFoundError() || error.isAuthorizationError())
  );
}

/**
 * Applies the server's notice that an item changed by reading that one item
 * and putting it where it now belongs in the cached queue — in its place by
 * deadline, or out of the queue once it is resolved or snoozed — instead of
 * reading the whole queue again. A new signal arrives the same way. An item
 * the reader can no longer read leaves the queue.
 *
 * Agent progress arrives as the run's own notice; the rows that run is working
 * are read again, so the step bar moves as the run does. The counts are small
 * reads and refresh where they are shown. A bulk notice names no item, so it
 * refreshes the queue on screen.
 */
export function createWatchtowerPatcher({
  queryClient,
  fetchItem = (id) => fetchWatchtowerItem(id),
  coalesceMs = COALESCE_MS,
}: WatchtowerPatcherOptions): WatchtowerPatcher {
  const pending = new Map<string, ReturnType<typeof setTimeout>>();
  let disposed = false;

  const refreshSummaries = () => {
    void queryClient.invalidateQueries({
      queryKey: watchtower.counts().queryKey,
      refetchType: "active",
    });
    void queryClient.invalidateQueries({ queryKey: attention._def, refetchType: "active" });
  };

  const read = async (id: string) => {
    pending.delete(id);
    let item: WatchtowerItem;
    try {
      item = await fetchItem(id);
    } catch (error) {
      if (disposed) {
        return;
      }
      if (gone(error)) {
        removeFromQueue(queryClient, id);
      } else {
        void queryClient.invalidateQueries({
          queryKey: watchtower.queue.queryKey,
          refetchType: "active",
        });
      }
      refreshSummaries();
      return;
    }
    if (disposed) {
      return;
    }

    const known = cachedQueueItem(queryClient, id);
    if (known !== undefined && item.version < known.version) {
      return;
    }
    placeInQueue(
      queryClient,
      known === undefined ? item : { ...item, seen: item.seen || known.seen },
    );
    refreshSummaries();
  };

  const schedule = (id: string) => {
    const waiting = pending.get(id);
    if (waiting !== undefined) {
      clearTimeout(waiting);
    }
    pending.set(
      id,
      setTimeout(() => void read(id), coalesceMs),
    );
  };

  return {
    handle: (event) => {
      if (disposed) {
        return;
      }
      const id = resolveEntityID(event);
      if (id === "" || isBulkAction(event.action ?? "")) {
        void queryClient.invalidateQueries({
          queryKey: watchtower.queue.queryKey,
          refetchType: "active",
        });
        refreshSummaries();
        return;
      }
      applyNoticeToQueue(queryClient, id, event.entity);
      schedule(id);
    },
    handleRun: (event) => {
      if (disposed || event.resource !== AGENT_RUN_RESOURCE) {
        return;
      }
      const runId = resolveEntityID(event);
      if (runId === "") {
        return;
      }
      for (const item of cachedQueueItemsForRun(queryClient, runId)) {
        schedule(item.id);
      }
    },
    dispose: () => {
      disposed = true;
      for (const timer of pending.values()) {
        clearTimeout(timer);
      }
      pending.clear();
    },
  };
}
