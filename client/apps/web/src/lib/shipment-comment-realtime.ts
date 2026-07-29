import type { QueryClient } from "@tanstack/react-query";
import type { ResourceInvalidationEvent } from "@trenova/shared/hooks/realtime-patching";
import {
  adjustCommentCount,
  applyCommentRemoval,
  applyCommentUpsert,
  applyReplyCountDelta,
  commentKeys,
  findCommentInCaches,
  patchCommentInPage,
  patchCommentInPages,
  syncPinnedPage,
  type CommentInfiniteData,
  type CommentPage,
  type LocalShipmentComment,
} from "@/lib/shipment-comment-cache";
import { shipmentCommentSchema } from "@/types/shipment-comment";

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

const seenCommentIds = new Map<string, Set<string>>();

function seenSet(shipmentId: string): Set<string> {
  let set = seenCommentIds.get(shipmentId);
  if (!set) {
    set = new Set();
    seenCommentIds.set(shipmentId, set);
  }
  return set;
}

export function markCommentSeen(shipmentId: string, commentId: string): boolean {
  const set = seenSet(shipmentId);
  if (set.has(commentId)) return false;
  set.add(commentId);
  return true;
}

function forgetComment(shipmentId: string, commentId: string): boolean {
  return seenSet(shipmentId).delete(commentId);
}

type CommentInsertListener = (comment: LocalShipmentComment) => void;

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

function notifyCommentInsert(shipmentId: string, comment: LocalShipmentComment) {
  insertListeners.get(shipmentId)?.forEach((listener) => listener(comment));
}

function parseCommentEntity(entity: unknown): LocalShipmentComment | null {
  const parsed = shipmentCommentSchema.safeParse(entity);
  return parsed.success ? parsed.data : null;
}

export function isShipmentCommentEvent(evt: ResourceInvalidationEvent): boolean {
  return evt.resource === SHIPMENT_COMMENT_RESOURCE && COMMENT_ACTIONS.has(evt.action ?? "");
}

export function handleShipmentCommentEvent(
  queryClient: QueryClient,
  evt: ResourceInvalidationEvent,
  currentUserId: string | undefined,
): void {
  const action = evt.action ?? "";
  const comment = parseCommentEntity(evt.entity);
  const shipmentId = comment?.shipmentId ?? evt.recordId ?? "";

  if (!comment || !shipmentId) {
    void queryClient.invalidateQueries({ queryKey: ["shipment-comments"], refetchType: "all" });
    void queryClient.invalidateQueries({
      queryKey: ["shipment-comment-count"],
      refetchType: "all",
    });
    return;
  }

  switch (action) {
    case "created":
      handleCreated(queryClient, shipmentId, comment, currentUserId);
      return;
    case "deleted":
      handleDeleted(queryClient, shipmentId, comment);
      return;
    default:
      handleEntitySync(queryClient, shipmentId, comment);
  }
}

function handleCreated(
  queryClient: QueryClient,
  shipmentId: string,
  comment: LocalShipmentComment,
  currentUserId: string | undefined,
): void {
  const newlySeen = markCommentSeen(shipmentId, comment.id);
  const effects = applyCommentUpsert(queryClient, shipmentId, comment);
  const isOwn = comment.userId != null && comment.userId === currentUserId;

  if (comment.parentCommentId) {
    if (effects.insertedReply) {
      adjustCommentCount(queryClient, shipmentId, 1);
    } else if (!effects.sawRepliesCache && newlySeen && !isOwn) {
      applyReplyCountDelta(queryClient, shipmentId, comment.parentCommentId, 1);
      adjustCommentCount(queryClient, shipmentId, 1);
    }
    return;
  }

  if (effects.insertedTopLevel) {
    adjustCommentCount(queryClient, shipmentId, 1);
    if (!isOwn) {
      notifyCommentInsert(shipmentId, comment);
    }
  } else if (!effects.sawTopLevelCache && newlySeen && !isOwn) {
    adjustCommentCount(queryClient, shipmentId, 1);
  }
}

function handleDeleted(
  queryClient: QueryClient,
  shipmentId: string,
  comment: LocalShipmentComment,
): void {
  const removed = applyCommentRemoval(queryClient, shipmentId, comment.id);
  const wasSeen = forgetComment(shipmentId, comment.id);

  if (removed || wasSeen) {
    adjustCommentCount(queryClient, shipmentId, -1);
  } else {
    void queryClient.invalidateQueries({
      queryKey: commentKeys.count(shipmentId),
      refetchType: "all",
    });
  }

  if (comment.parentCommentId) {
    applyReplyCountDelta(queryClient, shipmentId, comment.parentCommentId, -1);
  }
}

function handleEntitySync(
  queryClient: QueryClient,
  shipmentId: string,
  comment: LocalShipmentComment,
): void {
  const previous = findCommentInCaches(queryClient, shipmentId, comment.id);
  const becameTombstone =
    comment.deletedAt != null && previous != null && previous.deletedAt == null;

  const entries = queryClient.getQueriesData({ queryKey: commentKeys.root(shipmentId) });
  for (const [queryKey] of entries) {
    const segment = queryKey[2];
    if (segment === "list" && queryKey[3] != null) {
      void queryClient.invalidateQueries({ queryKey, refetchType: "all" });
    }
  }

  patchEverywhere(queryClient, shipmentId, comment);

  if (becameTombstone) {
    adjustCommentCount(queryClient, shipmentId, -1);
    forgetComment(shipmentId, comment.id);
  } else if (comment.deletedAt != null && previous == null) {
    void queryClient.invalidateQueries({
      queryKey: commentKeys.count(shipmentId),
      refetchType: "all",
    });
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
