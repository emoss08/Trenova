import type { CarrierIntelEvent } from "@/lib/graphql/carrier-intelligence";
import {
  CARRIER_INTEL_EVENT_LIST_KEY,
  fetchCarrierIntelEventInbox,
  type CarrierIntelEventInboxVariables,
} from "@/lib/graphql/carrier-monitoring-table";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useMemo } from "react";

export const INBOX_PAGE_SIZE = 50;

export function inboxEventsQueryKey(variables: CarrierIntelEventInboxVariables) {
  return [CARRIER_INTEL_EVENT_LIST_KEY, "inbox", variables] as const;
}

export function useInboxEvents(variables: CarrierIntelEventInboxVariables) {
  const query = useInfiniteQuery({
    queryKey: inboxEventsQueryKey(variables),
    initialPageParam: null as string | null,
    queryFn: ({ pageParam, signal }) =>
      fetchCarrierIntelEventInbox(
        { ...variables, first: INBOX_PAGE_SIZE, after: pageParam },
        { signal },
      ),
    getNextPageParam: (lastPage) =>
      lastPage.hasNextPage && lastPage.endCursor ? lastPage.endCursor : undefined,
    placeholderData: (previous) => previous,
  });

  const events = useMemo<CarrierIntelEvent[]>(
    () => query.data?.pages.flatMap((page) => page.events) ?? [],
    [query.data],
  );

  return { ...query, events };
}
