import { HosClockGauges } from "@/components/hos/hos-clock-gauges";
import type { DispatchBoardDriver } from "@/lib/graphql/dispatch-console";
import { useDraggable, useDroppable } from "@dnd-kit/core";
import { Avatar, AvatarFallback, AvatarImage } from "@trenova/shared/components/ui/avatar";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { cn, pluralize } from "@trenova/shared/lib/utils";
import { GripVerticalIcon, SearchIcon } from "lucide-react";
import { useEffect, useMemo, useRef } from "react";
import { CapacityRailRowsSkeleton } from "./console-skeletons";
import {
  AVAILABILITY_SORT_RANK,
  CAPACITY_FILTERS,
  availabilityMeta,
  dutyStatusMeta,
  workerInitials,
  type CapacityFilter,
} from "./dispatch-vocabulary";
import { useDispatchRail } from "./url-state";

function DriverRow({
  driver,
  isSelected,
  onSelect,
}: {
  driver: DispatchBoardDriver;
  isSelected: boolean;
  onSelect: (workerId: string) => void;
}) {
  const availability = availabilityMeta(driver.availability);
  const duty = driver.dutyStatus ? dutyStatusMeta(driver.dutyStatus) : undefined;
  const isBlocked = driver.availability === "Blocked";
  const hasFeed = driver.hosRecordedAt > 0;
  const blockingReason = isBlocked
    ? driver.findings.find((finding) => finding.severity === "Block")?.message
    : undefined;

  // A driver who frees up later is more useful stated as the time than as the word
  // "Working", so the status slot carries whichever of the two the dispatcher can act on.
  const isBusy = !isBlocked && driver.availability !== "Open";
  const statusLabel =
    isBusy && driver.projectedTimeAvailable > 0
      ? `Free ${formatUnixTime(driver.projectedTimeAvailable)}`
      : availability.label;

  const context = [
    driver.formattedLocation || `${driver.city}, ${driver.stateAbbreviation}`,
    driver.tractorCode,
    duty?.label,
    driver.openAssignments > 0
      ? `${driver.openAssignments} ${pluralize("load", driver.openAssignments)}`
      : undefined,
  ].filter(Boolean);

  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `driver:${driver.workerId}`,
    // A blocked driver cannot be dragged at all. Letting the drag start and then refusing
    // the drop teaches nothing; refusing to pick them up says why immediately.
    disabled: isBlocked,
    data: { type: "driver", driver },
  });

  // The browser fires a click on the source element when a drag ends; selecting the
  // driver then would clobber the move selection the drop just made.
  const wasDragged = useRef(false);
  useEffect(() => {
    if (isDragging) wasDragged.current = true;
  }, [isDragging]);
  const handleSelect = () => {
    if (wasDragged.current) {
      wasDragged.current = false;
      return;
    }
    onSelect(driver.workerId);
  };

  // Rows also accept a move bar dragged off the timeline, so covering a load works in
  // whichever direction the dispatcher thinks in.
  const {
    setNodeRef: setDropRef,
    isOver,
    active,
  } = useDroppable({
    id: `driver-target:${driver.workerId}`,
    disabled: isBlocked,
    data: { type: "driver-target", driver },
  });

  const activeMove = active?.data.current?.type === "move" ? active.data.current.move : null;
  const showDropHint = isOver && Boolean(activeMove);

  return (
    <div
      ref={(node) => {
        setNodeRef(node);
        setDropRef(node);
      }}
      {...attributes}
      {...listeners}
      role="button"
      tabIndex={0}
      onClick={handleSelect}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onSelect(driver.workerId);
        }
      }}
      className={cn(
        "group grid w-full grid-cols-[12px_minmax(0,1fr)] items-center gap-x-1.5 border-b px-2 py-2 text-left transition-colors last:border-b-0",
        isSelected ? "bg-muted" : "hover:bg-muted/50",
        isDragging && "opacity-40",
        showDropHint && "bg-brand/10",
        isBlocked ? "cursor-not-allowed" : "cursor-grab active:cursor-grabbing",
      )}
    >
      <span className="flex h-full items-center justify-center" aria-hidden>
        {!isBlocked && (
          <GripVerticalIcon className="size-3 text-muted-foreground/0 transition-colors group-hover:text-muted-foreground/60" />
        )}
      </span>

      <div className="flex min-w-0 flex-col gap-1.5">
        <div className="flex min-w-0 items-center gap-2">
          <Avatar className="size-6 shrink-0">
            {driver.profilePicUrl && (
              <AvatarImage
                src={driver.profilePicUrl}
                alt={`${driver.firstName} ${driver.lastName}`}
              />
            )}
            <AvatarFallback className="text-[9px]">
              {workerInitials(driver.firstName, driver.lastName)}
            </AvatarFallback>
          </Avatar>

          <div className="flex min-w-0 flex-1 flex-col gap-0.5">
            <div className="flex min-w-0 items-baseline justify-between gap-2">
              <span className="truncate text-xs leading-none font-medium">
                {driver.firstName} {driver.lastName}
              </span>
              <span
                className={cn(
                  "flex shrink-0 items-center gap-1 text-[10px] leading-none font-medium",
                  availability.labelClass,
                )}
              >
                <span aria-hidden className={cn("size-1.5 rounded-full", availability.dotClass)} />
                {statusLabel}
              </span>
            </div>
            <p className="truncate text-[10.5px] leading-none text-muted-foreground">
              {context.join(" · ")}
            </p>
          </div>
        </div>

        <HosClockGauges
          driveRemainingMs={driver.driveRemainingMs}
          shiftRemainingMs={driver.shiftRemainingMs}
          cycleRemainingMs={driver.cycleRemainingMs}
          isStale={driver.hosIsStale}
          hasFeed={hasFeed}
        />

        {blockingReason && (
          <p className="text-[10px] leading-tight text-destructive">{blockingReason}</p>
        )}
      </div>
    </div>
  );
}

export function CapacityRail({
  drivers,
  isLoading,
  selectedWorkerId,
  onSelectDriver,
}: {
  drivers: readonly DispatchBoardDriver[];
  isLoading: boolean;
  selectedWorkerId: string | null;
  onSelectDriver: (workerId: string) => void;
}) {
  const {
    capacityFilter: filter,
    driverSearch: search,
    setCapacityFilter,
    setDriverSearch,
  } = useDispatchRail();

  const visible = useMemo(() => {
    const availability = CAPACITY_FILTERS.find((option) => option.id === filter)?.availability;
    const term = search.trim().toLowerCase();
    const filtered = drivers.filter((driver) => {
      if (availability && driver.availability !== availability) return false;
      if (!term) return true;
      return (
        `${driver.firstName} ${driver.lastName}`.toLowerCase().includes(term) ||
        driver.tractorCode.toLowerCase().includes(term) ||
        driver.fleetCodeName.toLowerCase().includes(term)
      );
    });
    return filtered.sort((a, b) => {
      const rank =
        (AVAILABILITY_SORT_RANK[a.availability] ?? 5) -
        (AVAILABILITY_SORT_RANK[b.availability] ?? 5);
      if (rank !== 0) return rank;
      return a.projectedTimeAvailable - b.projectedTimeAvailable;
    });
  }, [drivers, filter, search]);

  const counts = useMemo(() => {
    const result = new Map<CapacityFilter, number>([["all", drivers.length]]);
    for (const option of CAPACITY_FILTERS) {
      if (!option.availability) continue;
      result.set(
        option.id,
        drivers.reduce(
          (count, driver) => (driver.availability === option.availability ? count + 1 : count),
          0,
        ),
      );
    }
    return result;
  }, [drivers]);

  return (
    <section className="flex min-h-0 flex-col overflow-hidden rounded-lg border bg-card">
      <header className="flex flex-col gap-2 border-b p-2">
        <Input
          value={search}
          onChange={(event) => setDriverSearch(event.target.value)}
          placeholder="Search driver, tractor, fleet"
          leftElement={<SearchIcon className="size-3.5 text-muted-foreground" />}
          className="h-8 pl-7 text-xs"
          aria-label="Search drivers by name, tractor code, or fleet"
        />
        <div className="flex flex-wrap gap-1">
          {CAPACITY_FILTERS.map((option) => {
            const count = counts.get(option.id) ?? 0;
            return (
              <button
                key={option.id}
                type="button"
                onClick={() => setCapacityFilter(option.id)}
                className={cn(
                  "rounded-full border px-2 py-0.5 text-[11px] font-medium transition-colors",
                  filter === option.id
                    ? "border-primary bg-primary text-primary-foreground"
                    : "text-muted-foreground hover:bg-muted",
                )}
              >
                {option.label} {count > 0 && <span className="tabular-nums">{count}</span>}
              </button>
            );
          })}
        </div>
      </header>

      <div className="flex items-center justify-between border-b px-2.5 py-1.5">
        <span className="text-[10.5px] font-semibold tracking-wide text-muted-foreground uppercase">
          Capacity
        </span>
        <span className="text-[10.5px] text-muted-foreground tabular-nums">
          {visible.length} of {drivers.length}
        </span>
      </div>

      <ScrollArea
        className="min-h-0 flex-1"
        viewportClassName="min-h-0"
        maskVariant="card"
        maskHeight={18}
      >
        {isLoading ? (
          <CapacityRailRowsSkeleton />
        ) : visible.length === 0 ? (
          <p className="p-4 text-center text-xs text-muted-foreground">
            No drivers match this view.
          </p>
        ) : (
          visible.map((driver) => (
            <DriverRow
              key={driver.workerId}
              driver={driver}
              isSelected={selectedWorkerId === driver.workerId}
              onSelect={onSelectDriver}
            />
          ))
        )}
      </ScrollArea>
    </section>
  );
}
