import { Button } from "@trenova/shared/components/ui/button";
import { pluralize } from "@trenova/shared/lib/utils";
import { AnimatePresence, m } from "motion/react";
import { DeskEmpty, DeskError, DeskNoMatches, DeskSkeleton } from "./desk-states";
import { DeskSummaryRail } from "./desk-summary-rail";
import { DeskToolbar } from "./desk-toolbar";
import { DetentionDeskRow } from "./detention-desk-row";
import { OccurrenceDetailSheet } from "./occurrence-detail-sheet";
import type { DetentionDeskState } from "./use-detention-desk";

const ROW_TRANSITION = { duration: 0.16, ease: "easeOut" } as const;

/**
 * The desk: what the floor is worth, the lane being worked, and the stops. The
 * claim file opens over the top rather than navigating away, so a clerk never
 * loses the list they were reading.
 */
export function DetentionDesk({ desk }: { desk: DetentionDeskState }) {
  const {
    entries,
    visible,
    summary,
    floor,
    laneCounts,
    lane,
    sort,
    search,
    selectedId,
    nowSeconds,
    isFiltered,
    isLoading,
    isError,
    setLane,
    setSort,
    setSearch,
    selectStop,
    resetFilters,
    refetch,
  } = desk;

  if (isLoading) {
    return <DeskSkeleton />;
  }

  if (isError) {
    return <DeskError onRetry={() => void refetch()} />;
  }

  if (entries.length === 0) {
    return <DeskEmpty />;
  }

  return (
    <>
      <DeskSummaryRail summary={summary} floor={floor} />
      <DeskToolbar
        lane={lane}
        laneCounts={laneCounts}
        sort={sort}
        search={search}
        onLane={setLane}
        onSort={setSort}
        onSearch={setSearch}
      />

      {visible.length === 0 ? (
        <DeskNoMatches onReset={resetFilters} />
      ) : (
        <div className="-mx-3 divide-y divide-border/60 px-4">
          <AnimatePresence initial={false}>
            {visible.map((entry) => (
              <m.div
                key={entry.occurrence.id}
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0 }}
                transition={ROW_TRANSITION}
              >
                <DetentionDeskRow
                  entry={entry}
                  nowSeconds={nowSeconds}
                  isSelected={selectedId === entry.occurrence.id}
                  onOpen={selectStop}
                />
              </m.div>
            ))}
          </AnimatePresence>
        </div>
      )}

      {isFiltered && visible.length > 0 && (
        <div className="flex items-center gap-2 text-xs text-muted-foreground p-2">
          <span className="tabular-nums">
            {visible.length} of {entries.length} {pluralize("stop", entries.length)}
          </span>
          <Button variant="link" size="xxs" className="h-auto p-0 text-xs" onClick={resetFilters}>
            Clear filters
          </Button>
        </div>
      )}
      <OccurrenceDetailSheet
        occurrenceId={selectedId}
        onOpenChange={(open) => {
          if (!open) {
            selectStop(null);
          }
        }}
      />
    </>
  );
}
