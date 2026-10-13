import {
  fetchWatchtowerCounts,
  fetchWatchtowerFeed,
  fetchWatchtowerItem,
  fetchWatchtowerItemHistory,
  fetchWatchtowerResolutionCodes,
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
  item: (id: string) => ({
    queryKey: [id],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchWatchtowerItem(id, { signal }),
  }),
  history: (id: string) => ({
    queryKey: [id],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchWatchtowerItemHistory(id, { signal }),
  }),
  resolutionCodes: () => ({
    queryKey: ["resolution-codes"],
    queryFn: ({ signal }: { signal?: AbortSignal }) => fetchWatchtowerResolutionCodes({ signal }),
  }),
});
