import { fetchDeskAttention, type DeskAttentionFilter } from "@/lib/graphql/desk-attention";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type QueryContext = { signal?: AbortSignal };

function normalized(filter?: DeskAttentionFilter): Required<DeskAttentionFilter> {
  return {
    lens: filter?.lens ?? "Everything",
    first: filter?.first ?? 0,
    recordId: filter?.recordId ?? "",
  };
}

/** Needs you: what waits on people, one row per record. */
export const deskAttention = createQueryKeys("deskAttention", {
  list: (filter?: DeskAttentionFilter) => {
    const key = normalized(filter);
    return {
      queryKey: [key],
      queryFn: ({ signal }: QueryContext) =>
        fetchDeskAttention(
          { lens: key.lens, first: key.first || undefined, recordId: key.recordId || null },
          { signal },
        ),
    };
  },
});
