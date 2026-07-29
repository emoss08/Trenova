import type { QueryClient, QueryKey } from "@tanstack/react-query";
import type { InfiniteData } from "@tanstack/react-query";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";
import type { ShipmentComment, ShipmentCommentFilter } from "@/types/shipment-comment";

export type LocalShipmentComment = ShipmentComment & {
  pending?: boolean;
  failed?: boolean;
};

export type CommentPage = GenericLimitOffsetResponse<LocalShipmentComment>;
export type CommentInfiniteData = InfiniteData<CommentPage, string | null>;

export const commentKeys = {
  root: (shipmentId: string) => ["shipment-comments", shipmentId] as const,
  list: (shipmentId: string, filter: ShipmentCommentFilter | null) =>
    ["shipment-comments", shipmentId, "list", normalizeCommentFilter(filter)] as const,
  replies: (shipmentId: string, parentCommentId: string) =>
    ["shipment-comments", shipmentId, "replies", parentCommentId] as const,
  pinned: (shipmentId: string) => ["shipment-comments", shipmentId, "pinned"] as const,
  compact: (shipmentId: string) => ["shipment-comments", shipmentId, "compact"] as const,
  count: (shipmentId: string) => ["shipment-comment-count", shipmentId] as const,
};

export function normalizeCommentFilter(
  filter: ShipmentCommentFilter | null | undefined,
): ShipmentCommentFilter | null {
  if (!filter) return null;
  const hasAny =
    (filter.types?.length ?? 0) > 0 ||
    (filter.priorities?.length ?? 0) > 0 ||
    (filter.authorIds?.length ?? 0) > 0 ||
    !!filter.mentionsUserId ||
    !!filter.pinnedOnly ||
    !!filter.unresolvedOnly ||
    !!(filter.search && filter.search.trim() !== "");
  return hasAny ? filter : null;
}

export function commentClientRef(comment: LocalShipmentComment): string | null {
  const ref = comment.metadata?.clientRef;
  return typeof ref === "string" && ref !== "" ? ref : null;
}

function isCommentPage(value: unknown): value is CommentPage {
  return (
    !!value &&
    typeof value === "object" &&
    "results" in value &&
    Array.isArray((value as { results: unknown[] }).results)
  );
}

function isInfiniteCommentData(value: unknown): value is CommentInfiniteData {
  return (
    !!value &&
    typeof value === "object" &&
    "pages" in value &&
    Array.isArray((value as { pages: unknown[] }).pages)
  );
}

function adjustPageCount(page: CommentPage, delta: number): CommentPage {
  const count = Math.max(0, page.count + delta);
  return {
    ...page,
    count,
    pageInfo: page.pageInfo
      ? { ...page.pageInfo, totalCount: Math.max(0, (page.pageInfo.totalCount ?? 0) + delta) }
      : page.pageInfo,
  };
}

function matchesComment(row: LocalShipmentComment, id: string, clientRef: string | null): boolean {
  if (row.id === id) return true;
  if (!clientRef) return false;
  return commentClientRef(row) === clientRef;
}

export function insertCommentIntoPages(
  data: CommentInfiniteData,
  comment: LocalShipmentComment,
): { data: CommentInfiniteData; inserted: boolean } {
  const clientRef = commentClientRef(comment);

  for (let pageIdx = 0; pageIdx < data.pages.length; pageIdx++) {
    const page = data.pages[pageIdx];
    const rowIdx = page.results.findIndex((row) => matchesComment(row, comment.id, clientRef));
    if (rowIdx < 0) continue;

    const nextResults = [...page.results];
    nextResults[rowIdx] = comment;
    const nextPages = [...data.pages];
    nextPages[pageIdx] = { ...page, results: nextResults };
    return { data: { ...data, pages: nextPages }, inserted: false };
  }

  if (data.pages.length === 0) {
    return {
      data: {
        pages: [
          {
            results: [comment],
            count: 1,
            next: null,
            prev: null,
            pageInfo: { mode: "cursor", hasNextPage: false, endCursor: null, totalCount: 1 },
          },
        ],
        pageParams: [null],
      },
      inserted: true,
    };
  }

  const nextPages = data.pages.map((page) => adjustPageCount(page, 1));
  nextPages[0] = { ...nextPages[0], results: [comment, ...nextPages[0].results] };
  return { data: { ...data, pages: nextPages }, inserted: true };
}

export function appendReplyToPages(
  data: CommentInfiniteData,
  reply: LocalShipmentComment,
): { data: CommentInfiniteData; inserted: boolean } {
  const clientRef = commentClientRef(reply);

  for (let pageIdx = 0; pageIdx < data.pages.length; pageIdx++) {
    const page = data.pages[pageIdx];
    const rowIdx = page.results.findIndex((row) => matchesComment(row, reply.id, clientRef));
    if (rowIdx < 0) continue;

    const nextResults = [...page.results];
    nextResults[rowIdx] = reply;
    const nextPages = [...data.pages];
    nextPages[pageIdx] = { ...page, results: nextResults };
    return { data: { ...data, pages: nextPages }, inserted: false };
  }

  if (data.pages.length === 0) {
    return {
      data: {
        pages: [
          {
            results: [reply],
            count: 1,
            next: null,
            prev: null,
            pageInfo: { mode: "cursor", hasNextPage: false, endCursor: null, totalCount: 1 },
          },
        ],
        pageParams: [null],
      },
      inserted: true,
    };
  }

  const nextPages = data.pages.map((page) => adjustPageCount(page, 1));
  const lastIdx = nextPages.length - 1;
  nextPages[lastIdx] = {
    ...nextPages[lastIdx],
    results: [...nextPages[lastIdx].results, reply],
  };
  return { data: { ...data, pages: nextPages }, inserted: true };
}

export function patchCommentInPages(
  data: CommentInfiniteData,
  commentId: string,
  patch: Partial<LocalShipmentComment> | LocalShipmentComment,
): { data: CommentInfiniteData; patched: boolean } {
  let patched = false;
  const nextPages = data.pages.map((page) => {
    const rowIdx = page.results.findIndex((row) => row.id === commentId);
    if (rowIdx < 0) return page;
    patched = true;
    const nextResults = [...page.results];
    nextResults[rowIdx] = { ...nextResults[rowIdx], ...patch };
    return { ...page, results: nextResults };
  });

  return patched ? { data: { ...data, pages: nextPages }, patched } : { data, patched };
}

export function removeCommentFromPages(
  data: CommentInfiniteData,
  commentId: string,
): { data: CommentInfiniteData; removed: boolean } {
  let removed = false;
  const strippedPages = data.pages.map((page) => {
    const nextResults = page.results.filter((row) => row.id !== commentId);
    if (nextResults.length === page.results.length) return page;
    removed = true;
    return { ...page, results: nextResults };
  });

  if (!removed) return { data, removed };

  return {
    data: { ...data, pages: strippedPages.map((page) => adjustPageCount(page, -1)) },
    removed,
  };
}

export function bumpReplyCountInPages(
  data: CommentInfiniteData,
  parentCommentId: string,
  delta: number,
): { data: CommentInfiniteData; patched: boolean } {
  let patched = false;
  const nextPages = data.pages.map((page) => {
    const rowIdx = page.results.findIndex((row) => row.id === parentCommentId);
    if (rowIdx < 0) return page;
    patched = true;
    const nextResults = [...page.results];
    nextResults[rowIdx] = {
      ...nextResults[rowIdx],
      replyCount: Math.max(0, (nextResults[rowIdx].replyCount ?? 0) + delta),
    };
    return { ...page, results: nextResults };
  });

  return patched ? { data: { ...data, pages: nextPages }, patched } : { data, patched };
}

export function patchCommentInPage(
  page: CommentPage,
  commentId: string,
  patch: Partial<LocalShipmentComment> | LocalShipmentComment,
): { data: CommentPage; patched: boolean } {
  const rowIdx = page.results.findIndex((row) => row.id === commentId);
  if (rowIdx < 0) return { data: page, patched: false };
  const nextResults = [...page.results];
  nextResults[rowIdx] = { ...nextResults[rowIdx], ...patch };
  return { data: { ...page, results: nextResults }, patched: true };
}

export type CommentCacheKind =
  | { kind: "list"; filtered: boolean }
  | { kind: "replies"; parentCommentId: string }
  | { kind: "pinned" }
  | { kind: "compact" }
  | { kind: "unknown" };

export function classifyCommentCacheKey(queryKey: QueryKey): CommentCacheKind {
  const segment = queryKey[2];
  if (segment === "list") return { kind: "list", filtered: queryKey[3] != null };
  if (segment === "replies" && typeof queryKey[3] === "string") {
    return { kind: "replies", parentCommentId: queryKey[3] };
  }
  if (segment === "pinned") return { kind: "pinned" };
  if (segment === "compact") return { kind: "compact" };
  return { kind: "unknown" };
}

export interface CommentCacheEffects {
  insertedTopLevel: boolean;
  insertedReply: boolean;
  sawTopLevelCache: boolean;
  sawRepliesCache: boolean;
}

export function applyCommentUpsert(
  queryClient: QueryClient,
  shipmentId: string,
  comment: LocalShipmentComment,
): CommentCacheEffects {
  const effects: CommentCacheEffects = {
    insertedTopLevel: false,
    insertedReply: false,
    sawTopLevelCache: false,
    sawRepliesCache: false,
  };
  const isReply = !!comment.parentCommentId;
  const entries = queryClient.getQueriesData({ queryKey: commentKeys.root(shipmentId) });

  for (const [queryKey, current] of entries) {
    const kind = classifyCommentCacheKey(queryKey);

    if (kind.kind === "list" && kind.filtered) {
      void queryClient.invalidateQueries({ queryKey, refetchType: "all" });
      continue;
    }

    if ((kind.kind === "list" || kind.kind === "compact") && isInfiniteCommentData(current)) {
      if (isReply) {
        const patched = patchCommentInPages(current, comment.id, comment);
        if (patched.patched) queryClient.setQueryData(queryKey, patched.data);
        continue;
      }
      if (kind.kind === "list") effects.sawTopLevelCache = true;
      const result = insertCommentIntoPages(current, comment);
      queryClient.setQueryData(queryKey, result.data);
      if (result.inserted && kind.kind === "list") effects.insertedTopLevel = true;
      continue;
    }

    if (kind.kind === "replies" && isInfiniteCommentData(current)) {
      if (!isReply || kind.parentCommentId !== comment.parentCommentId) {
        const patched = patchCommentInPages(current, comment.id, comment);
        if (patched.patched) queryClient.setQueryData(queryKey, patched.data);
        continue;
      }
      effects.sawRepliesCache = true;
      const result = appendReplyToPages(current, comment);
      queryClient.setQueryData(queryKey, result.data);
      if (result.inserted) effects.insertedReply = true;
      continue;
    }

    if (kind.kind === "pinned" && isCommentPage(current)) {
      queryClient.setQueryData(queryKey, syncPinnedPage(current, comment));
    }
  }

  if (isReply && effects.insertedReply) {
    applyReplyCountDelta(queryClient, shipmentId, comment.parentCommentId!, 1);
  }

  return effects;
}

export function syncPinnedPage(page: CommentPage, comment: LocalShipmentComment): CommentPage {
  const isPinned = comment.pinnedAt != null && comment.deletedAt == null;
  const existingIdx = page.results.findIndex((row) => row.id === comment.id);

  if (isPinned) {
    if (existingIdx >= 0) {
      const nextResults = [...page.results];
      nextResults[existingIdx] = comment;
      return { ...page, results: nextResults };
    }
    return adjustPageCount({ ...page, results: [comment, ...page.results] }, 1);
  }

  if (existingIdx < 0) return page;
  return adjustPageCount(
    { ...page, results: page.results.filter((row) => row.id !== comment.id) },
    -1,
  );
}

export function applyCommentRemoval(
  queryClient: QueryClient,
  shipmentId: string,
  commentId: string,
): boolean {
  let removedAny = false;
  const entries = queryClient.getQueriesData({ queryKey: commentKeys.root(shipmentId) });

  for (const [queryKey, current] of entries) {
    const kind = classifyCommentCacheKey(queryKey);

    if (kind.kind === "list" && kind.filtered) {
      void queryClient.invalidateQueries({ queryKey, refetchType: "all" });
      continue;
    }

    if (isInfiniteCommentData(current)) {
      const result = removeCommentFromPages(current, commentId);
      if (result.removed) {
        queryClient.setQueryData(queryKey, result.data);
        if (kind.kind === "list") removedAny = true;
      }
      continue;
    }

    if (kind.kind === "pinned" && isCommentPage(current)) {
      const idx = current.results.findIndex((row) => row.id === commentId);
      if (idx >= 0) {
        queryClient.setQueryData(
          queryKey,
          adjustPageCount(
            { ...current, results: current.results.filter((row) => row.id !== commentId) },
            -1,
          ),
        );
      }
    }
  }

  return removedAny;
}

export function applyCommentPatch(
  queryClient: QueryClient,
  shipmentId: string,
  commentId: string,
  patch: Partial<LocalShipmentComment>,
): void {
  const entries = queryClient.getQueriesData({ queryKey: commentKeys.root(shipmentId) });

  for (const [queryKey, current] of entries) {
    const kind = classifyCommentCacheKey(queryKey);

    if (kind.kind === "list" && kind.filtered) {
      void queryClient.invalidateQueries({ queryKey, refetchType: "all" });
      continue;
    }

    if (isInfiniteCommentData(current)) {
      const result = patchCommentInPages(current, commentId, patch);
      if (result.patched) queryClient.setQueryData(queryKey, result.data);
      continue;
    }

    if (kind.kind === "pinned" && isCommentPage(current)) {
      const result = patchCommentInPage(current, commentId, patch);
      if (result.patched) queryClient.setQueryData(queryKey, result.data);
    }
  }
}

export function applyReplyCountDelta(
  queryClient: QueryClient,
  shipmentId: string,
  parentCommentId: string,
  delta: number,
): void {
  const entries = queryClient.getQueriesData({ queryKey: commentKeys.root(shipmentId) });

  for (const [queryKey, current] of entries) {
    const kind = classifyCommentCacheKey(queryKey);
    if (kind.kind === "replies") continue;
    if (!isInfiniteCommentData(current)) continue;
    const result = bumpReplyCountInPages(current, parentCommentId, delta);
    if (result.patched) queryClient.setQueryData(queryKey, result.data);
  }
}

export function adjustCommentCount(
  queryClient: QueryClient,
  shipmentId: string,
  delta: number,
): void {
  queryClient.setQueryData(
    commentKeys.count(shipmentId),
    (current: { count: number } | undefined) =>
      current ? { count: Math.max(0, current.count + delta) } : current,
  );
}

export function snapshotCommentCaches(queryClient: QueryClient, shipmentId: string): () => void {
  const entries = queryClient.getQueriesData({ queryKey: commentKeys.root(shipmentId) });
  const countKey = commentKeys.count(shipmentId);
  const countSnapshot = queryClient.getQueryData(countKey);

  return () => {
    for (const [queryKey, data] of entries) {
      queryClient.setQueryData(queryKey, data);
    }
    queryClient.setQueryData(countKey, countSnapshot);
  };
}

export function findCommentInCaches(
  queryClient: QueryClient,
  shipmentId: string,
  commentId: string,
): LocalShipmentComment | null {
  const entries = queryClient.getQueriesData({ queryKey: commentKeys.root(shipmentId) });

  for (const [, current] of entries) {
    if (isInfiniteCommentData(current)) {
      for (const page of current.pages) {
        const found = page.results.find((row) => row.id === commentId);
        if (found) return found;
      }
    } else if (isCommentPage(current)) {
      const found = current.results.find((row) => row.id === commentId);
      if (found) return found;
    }
  }

  return null;
}
