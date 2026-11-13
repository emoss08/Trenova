import type { DeskLensCounts } from "@/lib/graphql/desk-attention";
import { watchtower } from "@/lib/queries/watchtower";
import { useDeskSetting, useDeskSettingsStore } from "@/stores/desk-settings-store";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useQueryStates } from "nuqs";
import { useMemo, useState } from "react";
import { severityMarkClass } from "../watchtower/severity";
import type { RecordLens } from "./lens";
import { NeedsFilterMenu } from "./needs-filter-menu";
import {
  NEEDS_FILTER_ADDRESS,
  NO_FILTERS,
  addressOf,
  filterTokens,
  filtersFromAddress,
  offeredKinds,
  withoutToken,
  type NeedsFilters,
} from "./needs-filters";
import type { NeedsHeadLens } from "./needs-head";
import { useAttentionList } from "./use-attention-list";

/**
 * What Everything and Watchtower say in the page's head, and what their list
 * is narrowed to. The search and the filters live in the address, so a link
 * sent to someone opens the same list and a reload comes back to it; the page
 * reads them here once and hands them to the lens.
 *
 * It also answers what waits in each lens, for the tabs. That is read with
 * the list and kept while another list is being read, so the numbers on the
 * tabs hold still as the person goes from lens to lens.
 */
export function useAttentionHead({
  lens,
  enabled,
  since,
  filterOpen,
  onFilterOpen,
}: {
  lens: RecordLens;
  /** Off for a reader who may not read the Watchtower: nothing is read. */
  enabled: boolean;
  /** The start of the reader's day, which "cleared today" counts from. */
  since: number;
  filterOpen: boolean;
  onFilterOpen: (open: boolean) => void;
}): {
  head: NeedsHeadLens;
  filters: NeedsFilters;
  setFilters: (filters: NeedsFilters) => void;
  /** What waits in each lens; null until the list has been read once. */
  counts: DeskLensCounts | null;
} {
  const t = useT();
  const [address, setAddress] = useQueryStates(NEEDS_FILTER_ADDRESS, { history: "replace" });
  const filters = useMemo(() => filtersFromAddress(address), [address]);
  const setFilters = (next: NeedsFilters) => void setAddress(addressOf(next));

  // A person who may see everyone's cases reads their own by default, and
  // everyone's when they ask; the choice is theirs and stays with them.
  const whose = useDeskSetting("cases");
  const setDeskSetting = useDeskSettingsStore((state) => state.set);
  const query = useAttentionList(lens, whose === "mine", filters, enabled);
  const clearedQuery = useQuery({ ...watchtower.cleared(since), enabled });

  // What the head says outlives the list it came with: a lens not read yet
  // has no answer of its own, and the last one still names what waits.
  const [known, setKnown] = useState(query.data);
  if (query.data !== undefined && query.data !== known) {
    setKnown(query.data);
  }

  const kinds = known?.kinds ?? [];
  const cleared = clearedQuery.data ?? 0;
  const total = cleared + (known?.counts.everything ?? 0);
  const tokens = filterTokens(t, filters, kinds).map((token) => ({
    key: token.key,
    label: token.label,
    mark:
      token.type === "severity" ? (
        <i className={cn("size-2 flex-none rounded-[2px]", severityMarkClass(token.severity))} />
      ) : undefined,
    onRemove: () => setFilters(withoutToken(filters, token)),
  }));

  return {
    filters,
    setFilters,
    counts: known?.counts ?? null,
    head: {
      progress: known
        ? { label: t("{0} of {1} cleared today", cleared, total), done: cleared, total }
        : null,
      whose: known?.seesEveryone ? whose : null,
      onWhose: (next) => setDeskSetting("cases", next),
      search: {
        value: filters.search,
        onCommit: (search) => setFilters({ ...filters, search }),
      },
      filter: (
        <NeedsFilterMenu
          filters={filters}
          kinds={offeredKinds(kinds, lens, filters.kinds)}
          open={filterOpen}
          onOpenChange={onFilterOpen}
          onChange={setFilters}
        />
      ),
      tokens,
      onClearFilters: () => setFilters({ ...NO_FILTERS, search: filters.search }),
    },
  };
}
