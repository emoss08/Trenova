import {
  DOT_RANDOM_DRAW_KEY,
  DOT_RANDOM_DRAWS_KEY,
  DOT_RANDOM_POOLS_KEY,
  fetchDotRandomDraw,
  fetchDotRandomDraws,
  fetchDotRandomPools,
} from "@/lib/graphql/worker-drug-alcohol";

export function randomPoolsQuery() {
  return {
    queryKey: [DOT_RANDOM_POOLS_KEY] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchDotRandomPools({ signal }),
  };
}

// Every draw is read once and narrowed on the client: the list is short (a
// handful of rounds a year per pool) and the pool filter must not cost a
// round trip.
export function randomDrawsQuery() {
  return {
    queryKey: [DOT_RANDOM_DRAWS_KEY] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchDotRandomDraws(undefined, { signal }),
  };
}

export function randomDrawQuery(id: string) {
  return {
    queryKey: [DOT_RANDOM_DRAW_KEY, id] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchDotRandomDraw(id, { signal }),
  };
}
