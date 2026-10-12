import { fetchRecordShipments, type ShipmentScope } from "@/lib/graphql/record-shipments";
import { infiniteQueryOptions } from "@tanstack/react-query";

/** The shipments a card lists for one record, a page at a time. */
export function recordShipmentsQuery(scope: ShipmentScope, pageSize: number) {
  return infiniteQueryOptions({
    queryKey: ["record-shipments", scope, pageSize] as const,
    queryFn: ({ pageParam, signal }) =>
      fetchRecordShipments(scope, { first: pageSize, after: pageParam }, { signal }),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => (last.hasNextPage && last.endCursor ? last.endCursor : undefined),
  });
}
