import { queries } from "@/lib/queries";
import { useQuery } from "@tanstack/react-query";
import {
  DESK_FILTER_IDS,
  DESK_SORT_IDS,
  countDeskFilters,
  deskFloorStats,
  matchesDeskFilter,
  matchesDeskSearch,
  sortDeskEntriesBy,
  summarizeDesk,
  type DeskFilterId,
  type DeskSortId,
} from "@trenova/shared/lib/detention";
import type { DeskEntry } from "@trenova/shared/types/detention";
import { debounce, parseAsString, parseAsStringLiteral, useQueryStates } from "nuqs";
import { useCallback, useEffect, useMemo, useState } from "react";

/** Refresh cadence for the board and for the local ticking clocks. */
const DESK_REFETCH_MS = 30_000;
const TICK_MS = 1_000;

/** Stable identity for the empty case so the memos below do not rerun on every render. */
const NO_ENTRIES: DeskEntry[] = [];

/**
 * Everything that decides what the desk shows lives in the URL, so a clerk can
 * hand a colleague the exact lane, sort, and stop they are looking at. The
 * ticking clock does not: it is derived from the wall clock, not from state.
 */
const deskParsers = {
  lane: parseAsStringLiteral(DESK_FILTER_IDS).withDefault("all"),
  sort: parseAsStringLiteral(DESK_SORT_IDS).withDefault("urgency"),
  q: parseAsString.withDefault(""),
  stop: parseAsString,
};

// Working the lanes back and forth should not bury the browser's back button.
const URL_OPTIONS = { history: "replace" } as const;

/** A whole-second clock, shared by every countdown on the screen so they tick together. */
export function useNowSeconds(): number {
  const [now, setNow] = useState(() => Math.floor(Date.now() / 1000));

  useEffect(() => {
    const id = setInterval(() => setNow(Math.floor(Date.now() / 1000)), TICK_MS);
    return () => clearInterval(id);
  }, []);

  return now;
}

/**
 * The desk's single source of truth. The page header, the exposure strip, the
 * toolbar, and the board all read from one query and one clock, so nothing on
 * screen can disagree about what time it is or how much money is at stake.
 */
export function useDetentionDesk() {
  const nowSeconds = useNowSeconds();
  const [{ lane, sort, q, stop }, setDesk] = useQueryStates(deskParsers, URL_OPTIONS);

  const { data, isLoading, isError, isFetching, dataUpdatedAt, refetch } = useQuery({
    ...queries.detention.desk(),
    refetchInterval: DESK_REFETCH_MS,
  });

  const entries = data ?? NO_ENTRIES;

  // The summary answers "how exposed are we", which the current lane must not change.
  const summary = useMemo(() => summarizeDesk(entries), [entries]);
  const laneCounts = useMemo(() => countDeskFilters(entries), [entries]);
  // Dwell is read off the live clock, so it recomputes with every tick.
  const floor = useMemo(() => deskFloorStats(entries, nowSeconds), [entries, nowSeconds]);

  const visible = useMemo(
    () =>
      sortDeskEntriesBy(
        entries.filter((entry) => matchesDeskFilter(entry, lane) && matchesDeskSearch(entry, q)),
        sort,
      ),
    [entries, lane, q, sort],
  );

  // Stops whose notice can still be sent inside the contractual window — the only
  // set where one action changes whether the charge survives a dispute.
  const noticeQueue = useMemo(() => entries.filter((entry) => entry.noticeWindowOpen), [entries]);

  const setLane = useCallback((next: DeskFilterId) => void setDesk({ lane: next }), [setDesk]);
  const setSort = useCallback((next: DeskSortId) => void setDesk({ sort: next }), [setDesk]);
  // The box reads from state on the keystroke; the URL catches up once typing stops.
  const setSearch = useCallback(
    (next: string) => void setDesk({ q: next }, { limitUrlUpdates: debounce(250) }),
    [setDesk],
  );
  const selectStop = useCallback(
    (occurrenceId: string | null) => void setDesk({ stop: occurrenceId }),
    [setDesk],
  );
  const resetFilters = useCallback(() => void setDesk({ lane: null, q: null }), [setDesk]);

  const isFiltered = lane !== "all" || q.length > 0;
  const updatedSecondsAgo =
    dataUpdatedAt === 0 ? null : Math.max(0, nowSeconds - Math.floor(dataUpdatedAt / 1000));

  return {
    nowSeconds,
    entries,
    visible,
    summary,
    laneCounts,
    floor,
    noticeQueue,
    lane,
    sort,
    search: q,
    selectedId: stop,
    isFiltered,
    isLoading,
    isError,
    isFetching,
    updatedSecondsAgo,
    setLane,
    setSort,
    setSearch,
    selectStop,
    resetFilters,
    refetch,
  };
}

export type DetentionDeskState = ReturnType<typeof useDetentionDesk>;
