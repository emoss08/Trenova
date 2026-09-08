import {
  fetchOshaLog,
  fetchOshaSummaries,
  OSHA_LOG_KEY,
  OSHA_SUMMARIES_KEY,
} from "@/lib/graphql/worker-injury";

/** One year of the log with its 300A totals counted over it. */
export function oshaLogQuery(year: number) {
  return {
    queryKey: [OSHA_LOG_KEY, year] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchOshaLog(year, { signal }),
  };
}

/** Every year's summary status, for the year picker's captions. */
export function oshaSummariesQuery() {
  return {
    queryKey: [OSHA_SUMMARIES_KEY] as const,
    queryFn: ({ signal }: { signal: AbortSignal }) => fetchOshaSummaries({ signal }),
  };
}
