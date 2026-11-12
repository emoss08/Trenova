import { fetchRecordShareCandidates, fetchRecordShares } from "@/lib/graphql/record-share";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type QueryContext = { signal?: AbortSignal };

/** Who a record is shared with, and who it can be shared with: one cache for every kind of record. */
export const recordShare = createQueryKeys("record-share", {
  list: (kind: string, id: string) => ({
    queryKey: [kind, id],
    queryFn: ({ signal }: QueryContext) => fetchRecordShares(kind, id, { signal }),
  }),
  candidates: (kind: string, id: string, query: string) => ({
    queryKey: [kind, id, query],
    queryFn: ({ signal }: QueryContext) => fetchRecordShareCandidates(kind, id, query, { signal }),
  }),
});
