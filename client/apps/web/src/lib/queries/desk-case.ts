import { fetchDeskCase } from "@/lib/graphql/desk-case";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type QueryContext = { signal?: AbortSignal };

/** A record's case: the record and every open signal about it, as one decision. */
export const deskCase = createQueryKeys("deskCase", {
  detail: (kind: string, id: string, timezone: string) => ({
    queryKey: [kind, id, timezone],
    queryFn: ({ signal }: QueryContext) => fetchDeskCase(kind, id, timezone, { signal }),
  }),
});
