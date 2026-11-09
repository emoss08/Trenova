import { fetchDeskActivity } from "@/lib/graphql/desk-attention";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type QueryContext = { signal?: AbortSignal };

/** Nova handled: the writes agents settled since a moment, newest first. */
export const deskActivity = createQueryKeys("deskActivity", {
  since: (since: number, first?: number) => ({
    queryKey: [since, first ?? 0],
    queryFn: ({ signal }: QueryContext) => fetchDeskActivity(since, first, { signal }),
  }),
});
