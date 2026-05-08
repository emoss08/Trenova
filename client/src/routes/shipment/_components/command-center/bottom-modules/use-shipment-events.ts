import { apiService } from "@/services/api";
import type { ShipmentEvent, ShipmentEventType } from "@/types/shipment-event";
import { useInfiniteQuery } from "@tanstack/react-query";

export const ACTIVITY_FEED_PAGE_SIZE = 20;

type Options = {
  shipmentId?: string;
  types?: ShipmentEventType[];
  pageSize?: number;
};

// Returns an infinite query of shipment events. The queryKey starts with
// "shipment-events" so realtime invalidations published on the data-events
// channel for the `shipmentEvents` resource (see realtime-patching.ts) match
// and trigger a refetch.
export function useShipmentEventsInfinite({ shipmentId, types, pageSize }: Options = {}) {
  const limit = pageSize ?? ACTIVITY_FEED_PAGE_SIZE;
  const typesKey = types && types.length > 0 ? [...types].sort().join(",") : "";

  return useInfiniteQuery({
    queryKey: [
      "shipment-events",
      { shipmentId: shipmentId ?? "all", typesKey, limit, types },
    ],
    initialPageParam: 0,
    queryFn: ({ pageParam }) =>
      apiService.shipmentEventService.list({
        shipmentId,
        types,
        limit,
        before: pageParam || undefined,
      }),
    getNextPageParam: (lastPage: ShipmentEvent[]) => {
      if (lastPage.length < limit) return undefined;
      return lastPage[lastPage.length - 1]?.occurredAt ?? undefined;
    },
    staleTime: 5_000,
  });
}
