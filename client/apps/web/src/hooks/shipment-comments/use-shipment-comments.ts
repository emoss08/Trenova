import { keepPreviousData, useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import {
  commentKeys,
  normalizeCommentFilter,
  type LocalShipmentComment,
} from "@/lib/shipment-comment-cache";
import { apiService } from "@/services/api";
import type { ShipmentCommentFilter } from "@/types/shipment-comment";

export const COMMENT_PAGE_SIZE = 20;
export const REPLY_PAGE_SIZE = 10;
export const PINNED_FETCH_LIMIT = 50;

export function useShipmentCommentList(
  shipmentId: string,
  filter: ShipmentCommentFilter | null = null,
) {
  const normalizedFilter = normalizeCommentFilter(filter);

  const listQuery = useInfiniteQuery({
    queryKey: commentKeys.list(shipmentId, normalizedFilter),
    queryFn: ({ pageParam }) =>
      apiService.shipmentCommentService.list(shipmentId, {
        limit: COMMENT_PAGE_SIZE,
        after: pageParam,
        filter: normalizedFilter,
      }),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) =>
      lastPage.pageInfo?.hasNextPage ? lastPage.pageInfo.endCursor : undefined,
    enabled: !!shipmentId,
    placeholderData: keepPreviousData,
  });

  const comments = useMemo(() => {
    const pages = listQuery.data?.pages;
    if (!pages) return [] as LocalShipmentComment[];
    const allPages = [...pages].reverse();
    return allPages.flatMap((page) => [...page.results].reverse()) as LocalShipmentComment[];
  }, [listQuery.data?.pages]);

  return {
    comments,
    total: listQuery.data?.pages[0]?.count ?? 0,
    isLoading: listQuery.isLoading,
    isError: listQuery.isError,
    error: listQuery.error,
    refetch: listQuery.refetch,
    isFiltering: listQuery.isFetching && listQuery.isPlaceholderData,
    hasNextPage: listQuery.hasNextPage,
    isFetchingNextPage: listQuery.isFetchingNextPage,
    fetchNextPage: listQuery.fetchNextPage,
    isFiltered: normalizedFilter != null,
  };
}

export function useShipmentCommentCount(shipmentId: string) {
  const countQuery = useQuery({
    queryKey: commentKeys.count(shipmentId),
    queryFn: () => apiService.shipmentCommentService.getCount(shipmentId),
    enabled: !!shipmentId,
    staleTime: 30_000,
  });

  return {
    commentCount: countQuery.data?.count ?? 0,
    isCountLoading: countQuery.isLoading,
  };
}

export function usePinnedShipmentComments(shipmentId: string) {
  const pinnedQuery = useQuery({
    queryKey: commentKeys.pinned(shipmentId),
    queryFn: () =>
      apiService.shipmentCommentService.list(shipmentId, {
        limit: PINNED_FETCH_LIMIT,
        filter: { pinnedOnly: true },
      }),
    enabled: !!shipmentId,
    staleTime: 60_000,
  });

  const pinnedComments = useMemo(() => {
    const results = (pinnedQuery.data?.results ?? []) as LocalShipmentComment[];
    return [...results].sort((a, b) => (b.pinnedAt ?? 0) - (a.pinnedAt ?? 0));
  }, [pinnedQuery.data?.results]);

  return {
    pinnedComments,
    isPinnedLoading: pinnedQuery.isLoading,
  };
}

export function useShipmentCommentReplies(
  shipmentId: string,
  parentCommentId: string,
  enabled: boolean,
) {
  const repliesQuery = useInfiniteQuery({
    queryKey: commentKeys.replies(shipmentId, parentCommentId),
    queryFn: ({ pageParam }) =>
      apiService.shipmentCommentService.listReplies(shipmentId, parentCommentId, {
        limit: REPLY_PAGE_SIZE,
        after: pageParam,
      }),
    initialPageParam: null as string | null,
    getNextPageParam: (lastPage) =>
      lastPage.pageInfo?.hasNextPage ? lastPage.pageInfo.endCursor : undefined,
    enabled: enabled && !!shipmentId && !!parentCommentId,
  });

  const replies = useMemo(() => {
    const pages = repliesQuery.data?.pages;
    if (!pages) return [] as LocalShipmentComment[];
    return pages.flatMap((page) => page.results) as LocalShipmentComment[];
  }, [repliesQuery.data?.pages]);

  return {
    replies,
    isRepliesLoading: repliesQuery.isLoading,
    isRepliesError: repliesQuery.isError,
    hasMoreReplies: repliesQuery.hasNextPage,
    isFetchingMoreReplies: repliesQuery.isFetchingNextPage,
    fetchMoreReplies: repliesQuery.fetchNextPage,
  };
}
