import type { DispatchBoardDriver, DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import type { WatchtowerItem } from "@/lib/graphql/watchtower";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { EmptyTableSketch } from "@trenova/shared/components/ui/empty-table";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { Ref } from "react";
import { Button } from "@trenova/shared/components/ui/button";
import { CAPACITY_FILTERS, type CapacityFilter } from "./dispatch-vocabulary";
import { DRIVER_GRID_CLASS, DriverQueue } from "./driver-queue";
import { QUEUE_GRID_CLASS } from "./console-skeletons";
import { ExceptionList } from "./exception-list";
import { MoveQueue, QUEUE_COLUMNS, QueueEmpty, QueueOutcome, type QueueRead } from "./move-queue";
import type { DispatchOutcome } from "./use-dispatch-actions";
import { useDispatchRail, type QueueTab } from "./url-state";

function capacityLabel(filter: CapacityFilter): string {
  return CAPACITY_FILTERS.find((option) => option.id === filter)?.label ?? "All";
}

function TabButton({
  id,
  active,
  label,
  count,
  onSelect,
}: {
  id: QueueTab;
  active: boolean;
  label: string;
  count: string | null;
  onSelect: (tab: QueueTab) => void;
}) {
  return (
    <button
      type="button"
      role="tab"
      id={`dispatch-tab-${id}`}
      aria-selected={active}
      aria-controls={`dispatch-tabpanel-${id}`}
      tabIndex={active ? 0 : -1}
      onClick={() => onSelect(id)}
      className={cn(
        "-mb-px inline-flex h-10 items-center gap-1.5 border-b-2 text-xs font-medium ui-inset-focus-ring transition-colors",
        active
          ? "border-foreground text-foreground"
          : "border-transparent text-muted-foreground hover:text-foreground",
      )}
    >
      {label}
      {count !== null ? (
        <span className="font-normal text-muted-foreground tabular-nums">{count}</span>
      ) : null}
    </button>
  );
}

function KeyLegend() {
  const t = useT();
  const key = (label: string) => <Kbd className="h-4 min-w-4 text-3xs">{label}</Kbd>;

  return (
    <div className="flex flex-wrap gap-x-4 gap-y-1 border-t border-border bg-card px-3.5 py-2 text-2xs text-muted-foreground">
      <span className="inline-flex items-center gap-1">
        {key("J")}
        {key("K")}
        {t("next and previous")}
      </span>
      <span className="inline-flex items-center gap-1">
        {key("A")}
        {t("approve proposal")}
      </span>
      <span className="inline-flex items-center gap-1">
        {key("D")}
        {t("decline")}
      </span>
      <span className="inline-flex items-center gap-1">
        {key("1")}–{key("9")}
        {t("pick a driver")}
      </span>
      <span className="inline-flex items-center gap-1">
        {key("U")}
        {t("undo")}
      </span>
      <span className="inline-flex items-center gap-1">
        {key("/")}
        {t("search")}
      </span>
      <span className="inline-flex items-center gap-1">
        {key("Esc")}
        {t("close")}
      </span>
    </div>
  );
}

/**
 * The console's main surface: the moves that need a decision, the exceptions covered moves
 * still raise, the drivers, and the freight likely to appear beyond the drivers expected
 * for it. One tab at a time, so the queue keeps its height.
 */
export function QueuePane({
  tab,
  onTab,
  moves,
  moveCount,
  read,
  narrowed,
  includeCovered,
  onClearSearch,
  selectedMoveId,
  departingMoveIds,
  onSelectMove,
  outcome,
  canUndo,
  onUndo,
  exceptions,
  exceptionsRead,
  exceptionsMore,
  drivers,
  driversTruncated,
  selectedWorkerId,
  onSelectDriver,
  showsDrivers,
  listRef,
}: {
  tab: QueueTab;
  onTab: (tab: QueueTab) => void;
  moves: readonly DispatchBoardMove[];
  moveCount: number | null;
  read: QueueRead;
  narrowed: boolean;
  includeCovered: boolean;
  onClearSearch: () => void;
  selectedMoveId: string | null;
  departingMoveIds: ReadonlySet<string>;
  onSelectMove: (moveId: string) => void;
  outcome: DispatchOutcome | null;
  canUndo: boolean;
  onUndo: () => void;
  exceptions: readonly WatchtowerItem[];
  exceptionsRead: QueueRead;
  exceptionsMore: boolean;
  drivers: readonly DispatchBoardDriver[];
  driversTruncated: boolean;
  selectedWorkerId: string | null;
  onSelectDriver: (workerId: string) => void;
  showsDrivers: boolean;
  listRef: Ref<HTMLDivElement>;
}) {
  const t = useT();

  // Drivers are employed drivers, so only a fleet sees them.
  const activeTab: QueueTab = tab === "drivers" && !showsDrivers ? "moves" : tab;
  const { capacityFilter, driverSearch, setCapacityFilter, setDriverSearch } = useDispatchRail();
  // The column heading and the key legend belong to rows; the skeleton, the error sheet
  // and the empty sheet draw their own.
  const hasRows = !read.isPending && !read.isError && moves.length > 0;
  const exceptionCount =
    exceptionsRead.isPending || exceptionsRead.isError
      ? null
      : exceptionsMore
        ? t("{0}+", exceptions.length)
        : String(exceptions.length);

  return (
    <section
      className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden bg-background"
      aria-label={t("Dispatch queue")}
    >
      <div
        className="flex h-10 shrink-0 items-end gap-4.5 border-b border-border px-3.5"
        role="tablist"
        aria-label={t("Queue")}
      >
        <TabButton
          id="moves"
          active={activeTab === "moves"}
          label={includeCovered ? t("Moves") : t("Needs a driver")}
          count={moveCount === null ? null : String(moveCount)}
          onSelect={onTab}
        />
        <TabButton
          id="exceptions"
          active={activeTab === "exceptions"}
          label={t("Exceptions")}
          count={exceptionCount}
          onSelect={onTab}
        />
        {showsDrivers ? (
          <TabButton
            id="drivers"
            active={activeTab === "drivers"}
            label={t("Drivers")}
            count={
              read.isPending || read.isError
                ? null
                : driversTruncated
                  ? t("{0}+", drivers.length)
                  : String(drivers.length)
            }
            onSelect={onTab}
          />
        ) : null}
        <span className="mb-3 ml-auto flex items-center gap-1.5 text-2xs text-muted-foreground">
          {activeTab === "moves" ? (
            t("Ordered by the next decision")
          ) : activeTab === "exceptions" ? (
            t("Soonest due first, from the Watchtower")
          ) : capacityFilter !== "all" ? (
            <>
              {t("Showing {0} only", t(capacityLabel(capacityFilter)))}
              <Button variant="link" size="xxs" onClick={() => setCapacityFilter("all")}>
                {t("Show all")}
              </Button>
            </>
          ) : (
            t("Open capacity first")
          )}
        </span>
      </div>

      <div
        ref={listRef}
        role="tabpanel"
        id={`dispatch-tabpanel-${activeTab}`}
        aria-labelledby={`dispatch-tab-${activeTab}`}
        className="flex min-h-0 flex-1 flex-col"
      >
        {activeTab === "moves" ? (
          <>
            <QueueOutcome outcome={outcome} canUndo={canUndo} onUndo={onUndo} />
            {hasRows ? (
              <div
                aria-hidden
                className={cn(
                  QUEUE_GRID_CLASS,
                  "h-7.5 shrink-0 items-center border-b border-border bg-background uppercase pr-5.5 pl-3.5 text-2xs font-medium text-muted-foreground",
                )}
              >
                <span>{t("Pickup")}</span>
                <span>{t("Move")}</span>
                <span>{t("Next step")}</span>
              </div>
            ) : null}
            <MoveQueue
              moves={moves}
              read={read}
              selectedMoveId={selectedMoveId}
              departingMoveIds={departingMoveIds}
              onSelectMove={onSelectMove}
              errorState={
                <ErrorState
                  error={read.error}
                  layout="sheet"
                  sketch={<EmptyTableSketch columns={QUEUE_COLUMNS} />}
                  title={t("The board didn't load")}
                  description={t(
                    "This isn't an empty board: moves in this window may still need drivers.",
                  )}
                  onRetry={read.refetch}
                />
              }
              emptyState={
                <QueueEmpty
                  narrowed={narrowed}
                  includeCovered={includeCovered}
                  onClearSearch={onClearSearch}
                />
              }
            />
            {hasRows ? <KeyLegend /> : null}
          </>
        ) : activeTab === "exceptions" ? (
          <ExceptionList items={exceptions} hasMore={exceptionsMore} read={exceptionsRead} />
        ) : (
          <>
            {!read.isPending && drivers.length > 0 ? (
              <div
                aria-hidden
                className={cn(
                  DRIVER_GRID_CLASS,
                  "h-7.5 shrink-0 items-center border-b border-border bg-muted/40 pr-5.5 pl-3.5 text-2xs font-medium text-muted-foreground",
                )}
              >
                <span>{t("Driver")}</span>
                <span>{t("When free")}</span>
                <span>{t("Hours left")}</span>
                <span>{t("Work")}</span>
              </div>
            ) : null}
            <DriverQueue
              drivers={drivers}
              truncated={driversTruncated}
              read={read}
              filter={capacityFilter}
              search={driverSearch}
              selectedWorkerId={selectedWorkerId}
              onSelectDriver={onSelectDriver}
              onClearFilters={() => {
                setCapacityFilter("all");
                setDriverSearch("");
              }}
            />
          </>
        )}
      </div>
    </section>
  );
}
