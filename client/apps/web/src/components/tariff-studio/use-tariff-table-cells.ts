import { formulaTemplate } from "@/lib/queries/formula-template";
import {
  emptyTableCellCache,
  mergeTableCells,
  uncachedNeeds,
  type TableCellCache,
} from "@/lib/tariff-table-cells";
import { apiService } from "@/services/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { TariffNeed } from "@trenova/shared/types/tariff";
import { useCallback } from "react";

/** How long cells nobody on screen reads are kept. */
const CELLS_GC_MS = 30 * 60_000;

export type TariffTableCellsState = {
  /** The cells loaded so far at the context's stamp. */
  cache: TableCellCache;
  /**
   * Loads the cells of `needs`, or with `whole` the tables they name, into the cache; settles
   * when the answer is in (a failure shows as `failed`).
   */
  load: (needs: readonly TariffNeed[], whole: boolean) => Promise<void>;
  loading: boolean;
  failed: boolean;
  /** Forgets a failed load, so the next run asks again. */
  reset: () => void;
};

/**
 * The rate table cells the browser's Tariff tests have loaded, shared by every screen at one
 * stamp through the query cache (so a session's end clears them with everything else). A load
 * whose answer was read at another stamp means a matrix version was published or archived since
 * the context was read: the cells are kept under the new stamp and the context is read again.
 */
export function useTariffTableCells(stamp: string): TariffTableCellsState {
  const queryClient = useQueryClient();
  const { data } = useQuery({
    queryKey: formulaTemplate.contextCells(stamp).queryKey,
    queryFn: ({ queryKey }) =>
      queryClient.getQueryData<TableCellCache>(queryKey) ?? emptyTableCellCache(stamp),
    initialData: () => emptyTableCellCache(stamp),
    staleTime: Infinity,
    gcTime: CELLS_GC_MS,
  });
  const cache = data ?? emptyTableCellCache(stamp);

  const mutation = useMutation({
    mutationFn: async ({ needs, whole }: { needs: readonly TariffNeed[]; whole: boolean }) =>
      apiService.formulaTemplateService.getContextCells({ needs, whole }),
    onSuccess: (answer) => {
      queryClient.setQueryData<TableCellCache>(
        formulaTemplate.contextCells(answer.stamp).queryKey,
        (previous) => mergeTableCells(previous ?? emptyTableCellCache(answer.stamp), answer),
      );
      if (answer.stamp !== stamp) {
        void queryClient.invalidateQueries({ queryKey: formulaTemplate.context.queryKey });
      }
    },
  });
  const { mutateAsync, reset } = mutation;

  const load = useCallback(
    async (needs: readonly TariffNeed[], whole: boolean) => {
      const wanted = whole ? [...needs] : uncachedNeeds(cache, needs);
      if (wanted.length === 0) return;
      await mutateAsync({ needs: wanted, whole }).catch(() => undefined);
    },
    [cache, mutateAsync],
  );

  return {
    cache,
    load,
    loading: mutation.isPending,
    failed: mutation.isError,
    reset,
  };
}
