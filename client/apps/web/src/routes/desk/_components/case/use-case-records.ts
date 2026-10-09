import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { MentionCandidateRecord, MentionPageKind } from "@/types/assistant";
import { keepPreviousData, useInfiniteQuery } from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useMemo } from "react";

const SEARCH_DEBOUNCE_MS = 160;
/** Rows a page asks for: a screenful and a half of the picker. */
export const CASE_RECORD_PAGE = 25;

const NO_RECORDS: MentionCandidateRecord[] = [];

/**
 * The records of one kind a conversation can be made a case about, newest
 * and closest matches first. Nothing is read until the picker is open; then
 * one page at a time, the next asked for as the list scrolls near its end.
 * The previous results stay on screen while a new search loads, so typing
 * never blanks the list.
 */
export function useCaseRecords(search: string, kind: MentionPageKind) {
  const query = useDebounce(search.trim(), SEARCH_DEBOUNCE_MS);
  const result = useInfiniteQuery({
    queryKey: queries.assistant.caseRecords(query, kind).queryKey,
    queryFn: ({ pageParam, signal }) =>
      apiService.assistantService.searchMentionPage(query, kind, {
        offset: pageParam,
        limit: CASE_RECORD_PAGE,
        signal,
      }),
    initialPageParam: 0,
    getNextPageParam: (lastPage, pages) =>
      lastPage.hasMore ? pages.reduce((count, page) => count + page.results.length, 0) : undefined,
    placeholderData: keepPreviousData,
  });

  const records = useMemo(
    () => result.data?.pages.flatMap((page) => page.results) ?? NO_RECORDS,
    [result.data],
  );

  return {
    records,
    settledSearch: query,
    loading: result.isPending,
    refreshing: result.isFetching && !result.isFetchingNextPage && !result.isPending,
    failed: result.isError,
    hasNextPage: result.hasNextPage,
    isFetchingNextPage: result.isFetchingNextPage,
    fetchNextPage: result.fetchNextPage,
    refetch: result.refetch,
  };
}
