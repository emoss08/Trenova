import type { DispatchBoardDriver } from "@/lib/graphql/dispatch-console";
import { useDraggable } from "@dnd-kit/core";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Input } from "@trenova/shared/components/ui/input";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { GripVerticalIcon, TruckIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { availabilityMeta, dutyStatusMeta } from "./dispatch-vocabulary";
import { HosClockBars } from "./hos-clock-bars";

export type CapacityFilter = "all" | "open" | "finishing" | "blocked" | "timeOff";

const FILTERS: { id: CapacityFilter; label: string }[] = [
  { id: "all", label: "All" },
  { id: "open", label: "Open" },
  { id: "finishing", label: "Finishing" },
  { id: "blocked", label: "Blocked" },
  { id: "timeOff", label: "Time off" },
];

function matchesFilter(driver: DispatchBoardDriver, filter: CapacityFilter): boolean {
  switch (filter) {
    case "open":
      return driver.availability === "Open";
    case "finishing":
      return driver.availability === "Finishing";
    case "blocked":
      return driver.availability === "Blocked";
    case "timeOff":
      return driver.availability === "TimeOff";
    default:
      return true;
  }
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
  const availability = availabilityMeta(driver.availability);
  const duty = driver.dutyStatus ? dutyStatusMeta(driver.dutyStatus) : undefined;
  const isBlocked = driver.availability === "Blocked";

  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({
    id: `driver:${driver.workerId}`,
    // A blocked driver cannot be dragged at all. Letting the drag start and then refusing
    // the drop teaches nothing; refusing to pick them up says why immediately.
    disabled: isBlocked,
    data: { type: "driver", driver },
  });

  return (
    <div
      ref={setNodeRef}
      className={cn(
        "flex flex-col gap-1.5 rounded-md border p-2 transition-colors",
        isSelected ? "border-brand bg-brand/5" : "border-border bg-card hover:border-brand/40",
        isDragging && "opacity-50",
        isBlocked ? "cursor-not-allowed" : "cursor-grab active:cursor-grabbing",
      )}
      {...attributes}
      {...listeners}
      onClick={() => onSelect(driver.workerId)}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onSelect(driver.workerId);
        }
      }}
      role="button"
      tabIndex={0}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 flex-col">
          <span className="truncate text-xs font-medium">
            {driver.firstName} {driver.lastName}
          </span>
          <span className="truncate text-[10px] text-muted-foreground">
            {driver.formattedLocation || `${driver.city}, ${driver.stateAbbreviation}`}
          </span>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <Badge variant={availability.variant} className="h-4 rounded px-1 text-[9px]">
            {availability.label}
          </Badge>
          {!isBlocked ? (
            <GripVerticalIcon className="size-3 text-muted-foreground" aria-hidden />
          ) : null}
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-1">
        {driver.tractorCode ? (
          <Badge
            variant="outline"
            className="h-4 rounded border-border px-1 font-mono text-[9px]"
          >
            <TruckIcon className="mr-0.5 size-2.5" aria-hidden />
            {driver.tractorCode}
          </Badge>
        ) : null}
        {duty ? (
          <Badge variant={duty.variant} className="h-4 rounded px-1 text-[9px]">
            {duty.label}
          </Badge>
        ) : null}
        <Badge variant="outline" className="h-4 rounded px-1 text-[9px]">
          {driver.driverType}
        </Badge>
        {driver.openAssignments > 0 ? (
          <Badge variant="outline" className="h-4 rounded px-1 text-[9px]">
            {driver.openAssignments} open
          </Badge>
        ) : null}
      </div>

      <HosClockBars
        driveRemainingMs={driver.driveRemainingMs}
        shiftRemainingMs={driver.shiftRemainingMs}
        cycleRemainingMs={driver.cycleRemainingMs}
        isStale={driver.hosIsStale}
        hasFeed={driver.hosRecordedAt > 0}
      />

      {driver.availability !== "Open" ? (
        <span className="text-[10px] text-muted-foreground">
          Available {formatUnixDateTime(driver.projectedTimeAvailable)}
        </span>
      ) : null}

      {isBlocked && driver.findings.length > 0 ? (
        <span className="text-[10px] leading-tight text-red-600 dark:text-red-400">
          {driver.findings.find((finding) => finding.severity === "Block")?.message}
        </span>
      ) : null}
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
  const [filter, setFilter] = useState<CapacityFilter>("all");
  const [search, setSearch] = useState("");

  const visible = useMemo(() => {
    const term = search.trim().toLowerCase();
    return drivers.filter((driver) => {
      if (!matchesFilter(driver, filter)) return false;
      if (!term) return true;
      return (
        `${driver.firstName} ${driver.lastName}`.toLowerCase().includes(term) ||
        driver.tractorCode.toLowerCase().includes(term)
      );
    });
  }, [drivers, filter, search]);

  const counts = useMemo(() => {
    const result: Record<CapacityFilter, number> = {
      all: drivers.length,
      open: 0,
      finishing: 0,
      blocked: 0,
      timeOff: 0,
    };
    for (const driver of drivers) {
      if (driver.availability === "Open") result.open += 1;
      if (driver.availability === "Finishing") result.finishing += 1;
      if (driver.availability === "Blocked") result.blocked += 1;
      if (driver.availability === "TimeOff") result.timeOff += 1;
    }
    return result;
  }, [drivers]);

  return (
    <section className="flex h-full min-h-0 flex-col gap-2 rounded-md border border-border bg-background p-2">
      <header className="flex flex-col gap-2">
        <div className="flex items-baseline justify-between gap-2">
          <h2 className="text-xs font-semibold uppercase tracking-wide">Capacity</h2>
          <span className="text-[10px] text-muted-foreground">
            {visible.length} of {drivers.length}
          </span>
        </div>
        <Input
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="Find a driver or tractor"
          className="h-7 text-xs"
        />
        <div className="flex flex-wrap gap-1">
          {FILTERS.map((option) => (
            <button
              key={option.id}
              type="button"
              onClick={() => setFilter(option.id)}
              className={cn(
                "rounded px-1.5 py-0.5 text-[10px] transition-colors",
                filter === option.id
                  ? "bg-brand text-white"
                  : "bg-muted text-muted-foreground hover:bg-muted/70",
              )}
            >
              {option.label} {counts[option.id]}
            </button>
          ))}
        </div>
      </header>

      <ScrollArea className="min-h-0 flex-1">
        <div className="flex flex-col gap-1.5 pr-2">
          {isLoading
            ? Array.from({ length: 8 }, (_, index) => (
                <Skeleton key={index} className="h-24 rounded-md" />
              ))
            : visible.map((driver) => (
                <DriverRow
                  key={driver.workerId}
                  driver={driver}
                  isSelected={selectedWorkerId === driver.workerId}
                  onSelect={onSelectDriver}
                />
              ))}
          {!isLoading && visible.length === 0 ? (
            <p className="px-1 py-4 text-center text-xs text-muted-foreground">
              No drivers match this filter.
            </p>
          ) : null}
        </div>
      </ScrollArea>
    </section>
  );
}
