import { fetchRecordFailures, type FailureScope } from "@/lib/graphql/record-failures";
import { infiniteQueryOptions } from "@tanstack/react-query";

/** The service failures at a location's stops or on a shipment, newest first, a page at a time. */
export function recordFailuresQuery(scope: FailureScope, pageSize: number) {
  return infiniteQueryOptions({
    queryKey: ["record-failures", scope, pageSize] as const,
    queryFn: ({ pageParam, signal }) =>
      fetchRecordFailures(scope, { first: pageSize, after: pageParam }, { signal }),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => (last.hasNextPage && last.endCursor ? last.endCursor : undefined),
  });
}
