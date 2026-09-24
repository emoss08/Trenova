import {
  fetchAgentChoices,
  type AgentChoice,
  type AgentChoicePage,
  type AgentChoiceQuery,
  type AgentChoiceSource,
  type AgentOrigin,
} from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { keepPreviousData, useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";

/**
 * Agents read per page. One size for every list, so the picker, the corner
 * panel and the Desk's directory share the pages they have already read.
 */
export const AGENT_PAGE_SIZE = 24;

/** How long typing settles before the search goes to the server. */
const SEARCH_DEBOUNCE_MS = 200;

export type UseAgentChoicesOptions = {
  /** As typed; the hook waits for it to settle. */
  search: string;
  origin: AgentOrigin;
  /** The person's own agents, most recent first. They lead the list while nothing narrows it. */
  recentIds: readonly string[];
  pageSize?: number;
  enabled?: boolean;
  /**
   * Whose agents: the person's own, which every chat surface lists, or the
   * organization's, which only AI Control lists. Recent agents lead only the
   * person's own list.
   */
  source?: AgentChoiceSource;
};

export type AgentChoices = {
  /** Agents the person has asked before, shown ahead of the rest. Empty while searching or filtering. */
  recent: AgentChoice[];
  /** Every other askable agent read so far, alphabetical. */
  items: AgentChoice[];
  /** Every askable agent that matches, recent ones included; null until the first page lands. */
  totalCount: number | null;
  hasNextPage: boolean;
  fetchNextPage: () => void;
  isFetchingNextPage: boolean;
  /** Nothing to show yet. */
  isLoading: boolean;
  /** A newer search or filter is on its way; what is shown is the previous answer. */
  isRefreshing: boolean;
  isError: boolean;
  refetch: () => void;
  /** The search as it was sent, for the empty state to quote. */
  settledSearch: string;
};

/**
 * The agents a person can ask, a page at a time.
 *
 * The list is the server's, not the client's: the search runs over names and
 * descriptions there, the filter is a query there, and each page is read by
 * cursor, so an organization with two hundred agents costs the same first
 * paint as one with six. Recent agents are read by id and put first, and the
 * pages that follow leave them out rather than repeat them.
 */
export function useAgentChoices({
  search,
  origin,
  recentIds,
  pageSize = AGENT_PAGE_SIZE,
  enabled = true,
  source = "mine",
}: UseAgentChoicesOptions): AgentChoices {
  const settledSearch = useDebounce(search.trim(), SEARCH_DEBOUNCE_MS);
  const leadsWithRecent = source === "mine" && recentIds.length > 0;
  const browsing = settledSearch === "" && origin === "all" && leadsWithRecent;

  const recentQuery = useQuery({
    ...queries.assistant.agentChoicesByIds(recentIds),
    enabled: enabled && leadsWithRecent,
    staleTime: 60_000,
  });
  const recent = useMemo(
    () => (browsing ? (recentQuery.data ?? []) : []),
    [browsing, recentQuery.data],
  );

  const query = useMemo<AgentChoiceQuery>(
    () => ({
      search: settledSearch,
      origin,
      excludeIds: recent.map((agent) => agent.id),
    }),
    [origin, recent, settledSearch],
  );

  const pagesQuery = useInfiniteQuery<
    AgentChoicePage,
    Error,
    { pages: AgentChoicePage[] },
    ReturnType<typeof queries.assistant.agentChoices>["queryKey"],
    string | undefined
  >({
    queryKey: queries.assistant.agentChoices(query, source).queryKey,
    queryFn: ({ pageParam, signal }) =>
      fetchAgentChoices(
        query,
        { first: pageSize, after: pageParam, includeTotalCount: pageParam === undefined },
        { signal, source },
      ),
    initialPageParam: undefined,
    getNextPageParam: (lastPage) =>
      lastPage.hasNextPage && lastPage.endCursor ? lastPage.endCursor : undefined,
    // The recent agents decide what the pages leave out, so the pages wait
    // for them rather than being read twice.
    enabled: enabled && !(browsing && recentQuery.isLoading),
    placeholderData: keepPreviousData,
    staleTime: 60_000,
  });

  const { fetchNextPage: fetchNext, hasNextPage, isFetchingNextPage } = pagesQuery;
  const fetchNextPage = useCallback(() => {
    if (hasNextPage && !isFetchingNextPage) {
      void fetchNext();
    }
  }, [fetchNext, hasNextPage, isFetchingNextPage]);

  const pages = pagesQuery.data?.pages;
  const items = useMemo(() => pages?.flatMap((page) => page.items) ?? [], [pages]);
  const firstCount = pages?.[0]?.totalCount ?? null;

  return {
    recent,
    items,
    totalCount: firstCount === null ? null : firstCount + recent.length,
    hasNextPage,
    fetchNextPage,
    isFetchingNextPage,
    isLoading: (browsing && recentQuery.isLoading) || pagesQuery.isPending,
    isRefreshing: pagesQuery.isPlaceholderData || settledSearch !== search.trim(),
    isError: pagesQuery.isError || recentQuery.isError,
    refetch: () => {
      void recentQuery.refetch();
      void pagesQuery.refetch();
    },
    settledSearch,
  };
}
