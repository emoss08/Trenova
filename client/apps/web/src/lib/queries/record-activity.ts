import { fetchRecordActivity } from "@/lib/graphql/record-activity";
import { infiniteQueryOptions } from "@tanstack/react-query";

/** What was done to one record, newest first, a page at a time. */
export function recordActivityQuery(resourceId: string, pageSize: number) {
  return infiniteQueryOptions({
    queryKey: ["record-activity", resourceId, pageSize] as const,
    queryFn: ({ pageParam, signal }) =>
      fetchRecordActivity(resourceId, { first: pageSize, after: pageParam }, { signal }),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => (last.hasNextPage && last.endCursor ? last.endCursor : undefined),
  });
}
