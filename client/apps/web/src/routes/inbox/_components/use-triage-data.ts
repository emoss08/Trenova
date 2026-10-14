import type { InboundMessageFilter, InboundMessageRow } from "@/lib/graphql/inbox";
import { inbox } from "@/lib/queries/inbox";
import { inboxMessagesQuery, inboxReferenceQuery } from "@/lib/queries/inbox";
import { useInfiniteQuery, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback, useMemo } from "react";

/** Lane counts, kinds with waiting mail, sources' waiting numbers. One aggregate. */
export function useTriageCounts() {
  return useQuery(inbox.counts());
}

export function useTriageMailboxes() {
  return useQuery({ ...inbox.mailboxes(), retry: false });
}

export function useTriageViews() {
  return useQuery(inbox.views());
}

export function useTriageRules() {
  return useQuery(inbox.rules());
}

export function useTriageBriefing() {
  return useQuery(inbox.briefing());
}

export function useTriageAssignees() {
  return useQuery(inboxReferenceQuery(inbox.assignees()));
}

export function useTriageDocumentTypes() {
  return useQuery(inboxReferenceQuery(inbox.documentTypes()));
}

export type FolderPages = {
  messages: readonly InboundMessageRow[];
  isLoading: boolean;
  isError: boolean;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  fetchNextPage: () => void;
  retry: () => void;
};

const NO_MESSAGES: readonly InboundMessageRow[] = [];
const NO_FILTER: InboundMessageFilter = {};

/** One folder of the queue, a page at a time; nothing is asked for while `filter` is null. */
export function useFolderPages(filter: InboundMessageFilter | null): FolderPages {
  const query = useInfiniteQuery({
    ...inboxMessagesQuery(filter ?? NO_FILTER),
    enabled: filter !== null,
  });
  const messages = useMemo(
    () => query.data?.pages.flatMap((page) => page.messages) ?? NO_MESSAGES,
    [query.data],
  );
  const { hasNextPage, isFetchingNextPage, fetchNextPage, refetch } = query;

  const loadMore = useCallback(() => {
    if (hasNextPage && !isFetchingNextPage) {
      void fetchNextPage();
    }
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);
  const retry = useCallback(() => void refetch(), [refetch]);

  return {
    messages,
    isLoading: filter !== null && query.isLoading,
    isError: query.isError,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage: loadMore,
    retry,
  };
}

/**
 * Warms one message before it is opened: on a row's hover or focus, and for
 * the message J or K would open next, so opening it draws at once.
 */
export function usePrefetchMessage(): (id: string) => void {
  const queryClient = useQueryClient();

  return useCallback(
    (id: string) => {
      void queryClient.prefetchQuery({ ...inbox.message(id), staleTime: 30_000 });
    },
    [queryClient],
  );
}
