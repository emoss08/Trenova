import {
  fetchWatchtowerCounts,
  fetchWatchtowerFeed,
  type WatchtowerFilter,
} from "@/lib/graphql/watchtower";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const watchtower = createQueryKeys("watchtower", {
  counts: () => ({
    queryKey: ["counts"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchWatchtowerCounts({ signal }),
  }),
  // The filter is part of the key, so switching a chip is a different
  // question rather than the same one asked again.
  feed: (filter: WatchtowerFilter) => ({
    queryKey: [filter],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchWatchtowerFeed(filter, { signal }),
  }),
});
