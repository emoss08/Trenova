import { queries } from "@/lib/queries";
import {
  addQuickFilter,
  QUICK_FILTER_MENU,
  quickFilterKey,
  quickFilterLabel,
  removeQuickFilter,
  type QuickFilterToken,
} from "@/lib/shipment-board/quick-filters";
import { useT } from "@trenova/shared/i18n/use-t";
import type {
  DataTableExtraChips,
  DataTableSearchSuggestions,
} from "@trenova/shared/types/data-table";
import { useQuery } from "@tanstack/react-query";
import { useCallback, useMemo, useState } from "react";
import { useBoardScope } from "./use-board-scope";
import { useShipmentBoardUrl } from "./url-state";

/**
 * The board's quick filters, offered by the table's search field with how many
 * shipments each would leave, and shown beside the other filters as chips.
 */
export function useQuickFilterSearch(): {
  searchSuggestions: DataTableSearchSuggestions;
  chips: DataTableExtraChips;
} {
  const t = useT();
  const [{ qf }, setUrl] = useShipmentBoardUrl();
  const scope = useBoardScope();
  const [open, setOpen] = useState(false);
  const counts = useQuery({
    ...queries.shipmentBoard.quickFilterCounts(scope),
    enabled: open,
    staleTime: 15_000,
  });

  const setQuickFilters = useCallback(
    (next: QuickFilterToken[]) => void setUrl({ qf: next, expanded: null }),
    [setUrl],
  );

  const searchSuggestions = useMemo<DataTableSearchSuggestions>(
    () => ({
      title: t("Quick filters"),
      onOpenChange: setOpen,
      items: QUICK_FILTER_MENU.map((entry) => ({
        key: entry.filter,
        label: t(entry.label),
        dotClassName: entry.dotClassName,
        count: counts.data?.find((row) => row.filter === entry.filter)?.count,
        selected: qf.some((token) => token.filter === entry.filter),
        onSelect: () => setQuickFilters(addQuickFilter(qf, { filter: entry.filter })),
      })),
    }),
    [counts.data, qf, setQuickFilters, t],
  );

  const chips = useMemo<DataTableExtraChips>(
    () => ({
      items: qf.map((token) => ({
        key: quickFilterKey(token),
        label: quickFilterLabel(token, t),
        onRemove: () => setQuickFilters(removeQuickFilter(qf, token)),
      })),
      onClear: () => setQuickFilters([]),
    }),
    [qf, setQuickFilters, t],
  );

  return { searchSuggestions, chips };
}
