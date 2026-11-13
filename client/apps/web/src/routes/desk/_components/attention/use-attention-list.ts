import { deskAttention } from "@/lib/queries/desk-attention";
import { useQuery } from "@tanstack/react-query";
import type { RecordLens } from "./lens";
import { attentionFilter, type NeedsFilters } from "./needs-filters";

/**
 * What a lens lists by record. The page's head and the lens's list both read
 * it, as one read: the head for what waits in each lens and what the filter
 * offers, the list for its rows.
 *
 * A list being narrowed keeps the rows it had until the narrower ones
 * arrive, so typing a search never empties the pane between answers. A lens
 * or a change of whose cases is another list, and draws its own.
 */
export function useAttentionList(
  lens: RecordLens,
  mine: boolean,
  filters: NeedsFilters,
  enabled = true,
) {
  return useQuery({
    ...deskAttention.list(attentionFilter(lens, mine, filters)),
    enabled,
    placeholderData: (previous, previousQuery) => {
      const was = previousQuery?.queryKey[2];
      return was?.lens === lens && was.mine === mine ? previous : undefined;
    },
  });
}
