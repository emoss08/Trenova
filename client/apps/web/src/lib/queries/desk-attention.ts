import { fetchDeskAttention, type DeskAttentionFilter } from "@/lib/graphql/desk-attention";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type QueryContext = { signal?: AbortSignal };

function normalized(filter?: DeskAttentionFilter): Required<DeskAttentionFilter> {
  return {
    lens: filter?.lens ?? "Everything",
    kinds: [...(filter?.kinds ?? [])].sort(),
    severities: [...(filter?.severities ?? [])].sort(),
    first: filter?.first ?? 0,
    recordId: filter?.recordId ?? "",
    working: filter?.working ?? false,
    mine: filter?.mine ?? false,
  };
}

/** How many rows with a working run Today's block reads. */
const WORKING_READ = 12;

/** Needs you: what waits on people, one row per record. */
export const deskAttention = createQueryKeys("deskAttention", {
  list: (filter?: DeskAttentionFilter) => {
    const key = normalized(filter);
    return {
      queryKey: [key],
      queryFn: ({ signal }: QueryContext) =>
        fetchDeskAttention(
          {
            lens: key.lens,
            kinds: key.kinds,
            severities: key.severities,
            first: key.first || undefined,
            recordId: key.recordId || null,
            working: key.working,
            mine: key.mine,
          },
          { signal },
        ),
    };
  },
  /**
   * What agents are working on right now. A read of its own, so a run on a
   * record far down Needs you is listed, and so a run's realtime notice can
   * refresh this and nothing wider.
   */
  working: {
    queryKey: null,
    queryFn: ({ signal }: QueryContext) =>
      fetchDeskAttention({ working: true, first: WORKING_READ }, { signal }),
  },
});
