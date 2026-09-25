import { useT } from "@trenova/shared/i18n/use-t";
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
  const t = useT();

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

  // A charge waiting on approval has a stopped clock, so it is not on the board;
  // the billing queue links straight to it, and the claim file opens over
  // whatever state the board is in.
  const sheet = (
    <OccurrenceDetailSheet
      occurrenceId={selectedId}
      onOpenChange={(open) => {
        if (!open) {
          selectStop(null);
        }
      }}
    />
  );

  if (isLoading) {
    return (
      <>
        <DeskSkeleton />
        {sheet}
      </>
    );
  }

  if (isError) {
    return (
      <>
        <DeskError onRetry={() => void refetch()} />
        {sheet}
      </>
    );
  }

  if (entries.length === 0) {
    return (
      <>
        <DeskEmpty />
        {sheet}
      </>
    );
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
        <div className="divide-border/60 -mx-3 divide-y px-4">
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
        <div className="text-muted-foreground flex items-center gap-2 p-2 text-xs">
          <span className="tabular-nums">
            {t("{0} of {1} {2}", visible.length, entries.length, pluralize("stop", entries.length))}
          </span>
          <Button variant="link" size="xxs" className="h-auto p-0 text-xs" onClick={resetFilters}>
            {t("Clear filters")}
          </Button>
        </div>
      )}
      {sheet}
    </>
  );
}
