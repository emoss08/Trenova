import { VirtualRows, type VirtualRow, type VirtualRowsHandle } from "@/components/virtual-rows";
import type { DispatchBoardDriver } from "@/lib/graphql/dispatch-console";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import {
  EmptyTable,
  EmptyTableSketch,
  type EmptyTableColumn,
} from "@trenova/shared/components/ui/empty-table";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatClockDurationMs, formatUnixTime } from "@trenova/shared/lib/date";
import { cn, formatCompactCurrency } from "@trenova/shared/lib/utils";
import { useEffect, useRef } from "react";
import { MoveQueueRowsSkeleton } from "./console-skeletons";
import {
  AVAILABILITY_SORT_RANK,
  CAPACITY_FILTERS,
  availabilityMeta,
  dutyStatusMeta,
  formatMiles,
  workerInitials,
  type CapacityFilter,
} from "./dispatch-vocabulary";
import { GroupHeading, type QueueRead } from "./move-queue";

/** The drivers' columns, shared by their rows, their heading and the empty sheet. */
export const DRIVER_GRID_CLASS =
  "grid grid-cols-[minmax(0,1.2fr)_8.5rem_9.5rem_minmax(0,1fr)] gap-3.5";

export const DRIVER_COLUMNS: readonly EmptyTableColumn[] = [
  { label: "Driver" },
  { label: "When free" },
  { label: "Hours left" },
  { label: "Work" },
];

// Two lines of text and the row's padding, as the queue's rows.
const DRIVER_ROW_ESTIMATE_PX = 52;

/** The availability groups in the order a dispatcher reads them: open capacity first. */
const AVAILABILITY_ORDER = ["Open", "Finishing", "Working", "TimeOff", "Blocked"] as const;

function availabilityDescription(
  group: (typeof AVAILABILITY_ORDER)[number],
  t: TranslateFn,
): string {
  switch (group) {
    case "Open":
      return t("available and holding no work");
    case "Finishing":
      return t("on a load that ends soon");
    case "Working":
      return t("on a load");
    case "TimeOff":
      return t("away for the window");
    case "Blocked":
      return t("can't take work until cleared");
  }
}

/** The drivers a search and a lens leave, open capacity first and soonest free within each. */
export function visibleDrivers(
  drivers: readonly DispatchBoardDriver[],
  filter: CapacityFilter,
  search: string,
): DispatchBoardDriver[] {
  const availability = CAPACITY_FILTERS.find((option) => option.id === filter)?.availability;
  const term = search.trim().toLowerCase();
  return drivers
    .filter((driver) => {
      if (availability && driver.availability !== availability) return false;
      if (!term) return true;
      return (
        `${driver.firstName} ${driver.lastName}`.toLowerCase().includes(term) ||
        driver.tractorCode.toLowerCase().includes(term) ||
        driver.fleetCodeName.toLowerCase().includes(term)
      );
    })
    .sort((a, b) => {
      const rank =
        (AVAILABILITY_SORT_RANK[a.availability] ?? 5) -
        (AVAILABILITY_SORT_RANK[b.availability] ?? 5);
      return rank !== 0 ? rank : a.projectedTimeAvailable - b.projectedTimeAvailable;
    });
}

function location(driver: DispatchBoardDriver, t: TranslateFn): string {
  if (driver.formattedLocation) return driver.formattedLocation;
  if (driver.city) return t("{0}, {1}", driver.city, driver.stateAbbreviation);
  return t("Location unknown");
}

/** When the driver can take a load, said the way a dispatcher would ask for it. */
export function whenFree(driver: DispatchBoardDriver, t: TranslateFn): string {
  switch (driver.availability) {
    case "Open":
      return t("Free now");
    case "Blocked":
      return t("Blocked");
    case "TimeOff":
      return t("Time off");
    default:
      return driver.projectedTimeAvailable > 0
        ? t("Free {0}", formatUnixTime(driver.projectedTimeAvailable))
        : t("On a load");
  }
}

/** The hours a driver has left, or why there is nothing to read. */
export function hoursLeft(driver: DispatchBoardDriver, t: TranslateFn) {
  if (driver.hosRecordedAt <= 0) {
    return { line: t("No hours feed"), detail: t("judged on schedule alone"), muted: true };
  }
  const duty = driver.dutyStatus ? dutyStatusMeta(driver.dutyStatus) : undefined;
  return {
    line: t(
      "{0} drive · {1} shift",
      formatClockDurationMs(driver.driveRemainingMs),
      formatClockDurationMs(driver.shiftRemainingMs),
    ),
    detail: driver.hosIsStale
      ? t("hours not updated lately")
      : duty
        ? t(duty.label)
        : t("{0} left in the cycle", formatClockDurationMs(driver.cycleRemainingMs)),
    muted: driver.hosIsStale,
  };
}

function workLine(driver: DispatchBoardDriver, t: TranslateFn) {
  const blocking = driver.findings.find((finding) => finding.severity === "Block");
  if (driver.availability === "Blocked" && blocking) {
    return { line: t("Can't be dispatched"), detail: blocking.message, danger: true };
  }
  if (driver.openAssignments === 0) {
    return { line: t("No work yet"), detail: t("ready for a load"), danger: false };
  }
  const next = driver.commitments[0];
  return {
    line: [
      t("{0, plural, one {# load} other {# loads}}", driver.openAssignments),
      driver.committedMiles > 0 ? formatMiles(driver.committedMiles) : null,
      driver.committedRevenue > 0 ? formatCompactCurrency(driver.committedRevenue) : null,
    ]
      .filter(Boolean)
      .join(" · "),
    detail: next
      ? t("Next: {0} to {1}, {2}", next.proNumber, next.destinationCity, next.destinationState)
      : t("none in this window"),
    danger: false,
  };
}

function DriverRow({
  driver,
  isSelected,
  onSelect,
}: {
  driver: DispatchBoardDriver;
  isSelected: boolean;
  onSelect: (workerId: string) => void;
}) {
  const t = useT();
  const availability = availabilityMeta(driver.availability);
  const hours = hoursLeft(driver, t);
  const work = workLine(driver, t);
  const fleet = [driver.fleetCodeName, driver.tractorCode].filter(Boolean).join(" · ");

  return (
    <div className="pt-px">
      <div
        role="option"
        aria-selected={isSelected}
        tabIndex={isSelected ? 0 : -1}
        data-glide-target
        onClick={() => onSelect(driver.workerId)}
        onKeyDown={(event) => {
          if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            onSelect(driver.workerId);
          }
        }}
        className={cn(
          DRIVER_GRID_CLASS,
          "ui-focus-ring relative mx-1.5 cursor-pointer items-center rounded-md px-2 py-1.75 text-xs",
          isSelected && "bg-surface-hover",
        )}
      >
        <div className="flex min-w-0 items-center gap-2.5">
          <span
            aria-hidden
            className="bg-muted text-muted-foreground flex size-6 shrink-0 items-center justify-center rounded-full text-3xs font-medium"
          >
            {workerInitials(driver.firstName, driver.lastName)}
          </span>
          <div className="min-w-0">
            <div className="truncate font-medium">
              {t("{0} {1}", driver.firstName, driver.lastName)}
            </div>
            <div className="text-muted-foreground truncate text-2xs">
              {fleet || t("No fleet or tractor")}
            </div>
          </div>
        </div>
        <div className="min-w-0">
          <div className={cn("truncate font-medium", availability.labelClass)}>
            {whenFree(driver, t)}
          </div>
          <div className="text-muted-foreground truncate text-2xs">{location(driver, t)}</div>
        </div>
        <div className="min-w-0">
          <div
            className={cn(
              "truncate",
              hours.muted ? "text-muted-foreground" : "font-mono tabular-nums",
            )}
          >
            {hours.line}
          </div>
          <div className="text-muted-foreground truncate text-2xs">{hours.detail}</div>
        </div>
        <div className="min-w-0">
          <div className={cn("truncate", work.danger && "text-danger-foreground font-medium")}>
            {work.line}
          </div>
          <div className="text-muted-foreground truncate text-2xs" title={work.detail}>
            {work.detail}
          </div>
        </div>
      </div>
    </div>
  );
}

/**
 * The drivers on the board, grouped by what they can take now: open capacity first, then
 * drivers finishing a load, working, away, and blocked. It reads as the queue does, so
 * finding work for a truck feels the same as finding a truck for work.
 */
export function DriverQueue({
  drivers,
  read,
  filter,
  search,
  selectedWorkerId,
  onSelectDriver,
  onClearFilters,
}: {
  drivers: readonly DispatchBoardDriver[];
  read: QueueRead;
  filter: CapacityFilter;
  search: string;
  selectedWorkerId: string | null;
  onSelectDriver: (workerId: string) => void;
  onClearFilters: () => void;
}) {
  const t = useT();
  const listRef = useRef<VirtualRowsHandle>(null);

  // Only rows near the viewport are mounted, so a driver selected elsewhere is scrolled to.
  useEffect(() => {
    if (selectedWorkerId) listRef.current?.scrollToKey(selectedWorkerId, { align: "auto" });
  }, [selectedWorkerId]);

  if (read.isPending) {
    return <MoveQueueRowsSkeleton />;
  }
  if (read.isError) {
    return (
      <ErrorState
        error={read.error}
        layout="sheet"
        sketch={<EmptyTableSketch columns={DRIVER_COLUMNS} />}
        title={t("The drivers didn't load")}
        description={t("This isn't an empty roster: drivers may still be free for work.")}
        onRetry={read.refetch}
      />
    );
  }

  const visible = visibleDrivers(drivers, filter, search);
  const narrowed = filter !== "all" || search.trim() !== "";

  if (visible.length === 0) {
    return (
      <EmptyTable
        columns={DRIVER_COLUMNS}
        title={narrowed ? t("No drivers match") : t("No drivers on the board")}
        description={
          narrowed
            ? t(
                "Nobody matches the search or the availability shown. Clear them to see every driver.",
              )
            : t("No driver is available for dispatch in this window.")
        }
        onClearFilters={narrowed ? onClearFilters : undefined}
      />
    );
  }

  const rows: VirtualRow[] = [];
  for (const group of AVAILABILITY_ORDER) {
    const inGroup = visible.filter((driver) => driver.availability === group);
    if (inGroup.length === 0) continue;
    const meta = availabilityMeta(group);
    rows.push({
      key: `group-${group}`,
      sticky: true,
      render: () => (
        <GroupHeading
          label={t(meta.label)}
          description={availabilityDescription(group, t)}
          dotClass={meta.dotClass}
          count={inGroup.length}
        />
      ),
    });
    for (const driver of inGroup) {
      rows.push({
        key: driver.workerId,
        render: () => (
          <DriverRow
            driver={driver}
            isSelected={driver.workerId === selectedWorkerId}
            onSelect={onSelectDriver}
          />
        ),
      });
    }
  }

  return (
    <VirtualRows
      rows={rows}
      handleRef={listRef}
      estimateSize={DRIVER_ROW_ESTIMATE_PX}
      initialHeight={640}
      role="listbox"
      aria-label={t("Drivers")}
      hoverGlide
      mask="card"
      maskClassName="top-7"
      maskHeight={18}
      className="min-h-0 flex-1"
      footer={<div className="h-2" aria-hidden />}
    />
  );
}
