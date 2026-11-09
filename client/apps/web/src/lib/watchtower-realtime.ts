import { fetchWatchtowerItem, type WatchtowerItem } from "@/lib/graphql/watchtower";
import { attention } from "@/lib/queries/attention";
import { deskAttention } from "@/lib/queries/desk-attention";
import { watchtower } from "@/lib/queries/watchtower";
import {
  cachedQueueItem,
  cachedQueueItemsForRun,
  placeInQueue,
  removeFromQueue,
} from "@/lib/watchtower-cache";
import type { QueryClient } from "@tanstack/react-query";
import {
  isBulkAction,
  resolveEntityID,
  type QueryKeyRoot,
  type ResourceInvalidationEvent,
} from "@trenova/shared/hooks/realtime-patching";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";

export const WATCHTOWER_RESOURCE = "watchtower";
const AGENT_RUN_RESOURCE = "agent_run";

/** Notices inside this window are answered together. */
const COALESCE_MS = 250;

/**
 * More items than this changing inside one window is a burst: the queue is
 * read again in one request rather than item by item.
 */
const BURST_LIMIT = 8;

const QUEUE_ROOT: QueryKeyRoot = watchtower.queue.queryKey;
const SUMMARY_ROOTS: readonly QueryKeyRoot[] = [
  watchtower.counts().queryKey,
  attention._def,
  deskAttention._def,
];

export function isWatchtowerEvent(event: ResourceInvalidationEvent): boolean {
  return event.resource === WATCHTOWER_RESOURCE;
}

export type WatchtowerPatcherOptions = {
  queryClient: QueryClient;
  /**
   * Marks roots stale and refetches them, coalesced and throttled by the
   * realtime connection so a burst of notices is one refetch per root.
   */
  invalidate: (roots: readonly QueryKeyRoot[], activeOnly: boolean) => void;
  fetchItem?: (id: string) => Promise<WatchtowerItem>;
  coalesceMs?: number;
  burstLimit?: number;
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

function queueOnScreen(queryClient: QueryClient): boolean {
  return (
    queryClient
      .getQueryCache()
      .find({ queryKey: watchtower.queue.queryKey, exact: true })
      ?.isActive() ?? false
  );
}

/**
 * Applies the server's notice that an item changed. While the queue is on
 * screen, the item is read and put where it now belongs — in its place by
 * deadline, or out of the queue once it is resolved or snoozed — instead of
 * reading the whole queue again. A new signal arrives the same way. An item
 * the reader can no longer read leaves the queue.
 *
 * Notices are answered once per window, never one request per notice: with
 * the queue off screen nothing is read and the cached queue is only marked
 * stale, and a window in which more than `burstLimit` items moved reads the
 * queue again in one request. The counts and the attention summary refresh
 * through the connection's throttle, once per window however many items
 * moved. Every connected reader in a tenant hears every notice, so what one
 * notice costs is multiplied by everyone online.
 *
 * Agent progress arrives as the run's own notice; the rows that run is working
 * are read again, so the step bar moves as the run does. A bulk notice names
 * no item, so it refreshes the queue on screen.
 */
export function createWatchtowerPatcher({
  queryClient,
  invalidate,
  fetchItem = (id) => fetchWatchtowerItem(id),
  coalesceMs = COALESCE_MS,
  burstLimit = BURST_LIMIT,
}: WatchtowerPatcherOptions): WatchtowerPatcher {
  const pending = new Set<string>();
  let timer: ReturnType<typeof setTimeout> | null = null;
  let disposed = false;

  const refreshSummaries = () => invalidate(SUMMARY_ROOTS, true);
  const refreshQueue = () => invalidate([QUEUE_ROOT, ...SUMMARY_ROOTS], true);

  const read = async (id: string): Promise<void> => {
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
        invalidate([QUEUE_ROOT], true);
      }
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
  };

  const flush = () => {
    timer = null;
    const ids = [...pending];
    pending.clear();
    if (disposed || ids.length === 0) {
      return;
    }
    if (ids.length > burstLimit || !queueOnScreen(queryClient)) {
      refreshQueue();
      return;
    }
    void Promise.all(ids.map(read)).then(() => {
      if (!disposed) {
        refreshSummaries();
      }
    });
  };

  const schedule = (id: string) => {
    pending.add(id);
    if (timer === null) {
      timer = setTimeout(flush, coalesceMs);
    }
  };

  return {
    handle: (event) => {
      if (disposed) {
        return;
      }
      const id = resolveEntityID(event);
      if (id === "" || isBulkAction(event.action ?? "")) {
        refreshQueue();
        return;
      }
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
      if (timer !== null) {
        clearTimeout(timer);
      }
      timer = null;
      pending.clear();
    },
  };
}
