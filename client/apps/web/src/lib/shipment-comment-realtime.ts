import type { QueryClient } from "@tanstack/react-query";
import type { ResourceInvalidationEvent } from "@trenova/shared/hooks/realtime-patching";
import {
  commentKeys,
  patchCommentInPage,
  patchCommentInPages,
  syncPinnedPage,
  type CommentInfiniteData,
  type CommentPage,
  type LocalShipmentComment,
} from "@/lib/shipment-comment-cache";

export const SHIPMENT_COMMENT_RESOURCE = "shipment_comment";

const COMMENT_ACTIONS = new Set([
  "created",
  "updated",
  "deleted",
  "pinned",
  "unpinned",
  "resolved",
  "unresolved",
  "acknowledged",
]);

type CommentInsertListener = () => void;

const insertListeners = new Map<string, Set<CommentInsertListener>>();

export function subscribeToCommentInserts(
  shipmentId: string,
  listener: CommentInsertListener,
): () => void {
  let listeners = insertListeners.get(shipmentId);
  if (!listeners) {
    listeners = new Set();
    insertListeners.set(shipmentId, listeners);
  }
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
    if (listeners.size === 0) insertListeners.delete(shipmentId);
  };
}

function notifyCommentInsert(shipmentId: string) {
  insertListeners.get(shipmentId)?.forEach((listener) => listener());
}

export function isShipmentCommentEvent(evt: ResourceInvalidationEvent): boolean {
  return evt.resource === SHIPMENT_COMMENT_RESOURCE && COMMENT_ACTIONS.has(evt.action ?? "");
}

/**
 * A comment event names the shipment and who acted, never what was said: the
 * stream applies no reader's grants. The thread refetches where it is on
 * screen, through the read that does.
 */
export function handleShipmentCommentEvent(
  queryClient: QueryClient,
  evt: ResourceInvalidationEvent,
  currentUserId: string | undefined,
): void {
  const shipmentId = evt.recordId ?? "";
  if (!shipmentId) {
    void queryClient.invalidateQueries({ queryKey: ["shipment-comments"] });
    void queryClient.invalidateQueries({ queryKey: ["shipment-comment-count"] });
    return;
  }

  void queryClient.invalidateQueries({ queryKey: commentKeys.root(shipmentId) });
  void queryClient.invalidateQueries({ queryKey: commentKeys.count(shipmentId) });

  if (evt.action === "created" && evt.actorUserId !== currentUserId) {
    notifyCommentInsert(shipmentId);
  }
}

export function applyCommentEntitySync(
  queryClient: QueryClient,
  shipmentId: string,
  comment: LocalShipmentComment,
): void {
  patchEverywhere(queryClient, shipmentId, comment);
}

function patchEverywhere(
  queryClient: QueryClient,
  shipmentId: string,
  comment: LocalShipmentComment,
): void {
  const entries = queryClient.getQueriesData({ queryKey: commentKeys.root(shipmentId) });

  for (const [queryKey, current] of entries) {
    const segment = queryKey[2];
    if (segment === "list" && queryKey[3] != null) continue;

    if (segment === "pinned" && isPage(current)) {
      queryClient.setQueryData(queryKey, syncPinnedPage(current, comment));
      continue;
    }

    if (isInfinite(current)) {
      const result = patchCommentInPages(current, comment.id, comment);
      if (result.patched) queryClient.setQueryData(queryKey, result.data);
      continue;
    }

    if (isPage(current)) {
      const result = patchCommentInPage(current, comment.id, comment);
      if (result.patched) queryClient.setQueryData(queryKey, result.data);
    }
  }
}

function isInfinite(value: unknown): value is CommentInfiniteData {
  return (
    !!value &&
    typeof value === "object" &&
    "pages" in value &&
    Array.isArray((value as { pages: unknown[] }).pages)
  );
}

function isPage(value: unknown): value is CommentPage {
  return (
    !!value &&
    typeof value === "object" &&
    "results" in value &&
    Array.isArray((value as { results: unknown[] }).results)
  );
}

/**
 * The realtime scope for one shipment's comment thread, and the endpoints that
 * authorize joining it and typing in it. The server builds the same scope from
 * the shipment id in the path, so the two can never disagree about which
 * thread a signal belongs to.
 */
export function shipmentCommentsRealtime(shipmentId: string) {
  const base = `/shipments/${encodeURIComponent(shipmentId)}/comments`;
  return {
    scope: `shipment-comments:${shipmentId}`,
    presencePath: `${base}/presence/`,
    typingPath: `${base}/typing/`,
  };
}
