import { useT } from "@trenova/shared/i18n/use-t";
import { HosClockGauges } from "@/components/hos/hos-clock-gauges";
import { SectionPanel } from "@/components/section-panel";
import { VirtualRows, type VirtualRowsHandle } from "@/components/virtual-rows";
import type { DispatchBoardDriver } from "@/lib/graphql/dispatch-console";
import { useDraggable, useDroppable } from "@dnd-kit/core";
import { Avatar, AvatarFallback, AvatarImage } from "@trenova/shared/components/ui/avatar";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { GripVerticalIcon, SearchLgIcon } from "@trenova/shared/components/icons";
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

// Name, context line and the HOS clocks; a blocking reason adds a line, measured once mounted.
const DRIVER_ROW_ESTIMATE_PX = 84;

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
      ? t("Free {0}", formatUnixTime(driver.projectedTimeAvailable))
      : t(availability.label);

  const context = [
    driver.formattedLocation || `${driver.city}, ${driver.stateAbbreviation}`,
    driver.tractorCode,
    duty ? t(duty.label) : undefined,
    driver.openAssignments > 0
      ? t("{0, plural, one {# load} other {# loads}}", driver.openAssignments)
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
        "group grid w-full grid-cols-[12px_minmax(0,1fr)] items-center gap-x-1.5 px-2 py-2 text-left transition-colors",
        isSelected ? "bg-muted" : "hover:bg-muted/50",
        isDragging && "opacity-40",
        showDropHint && "bg-brand/10",
        isBlocked ? "cursor-not-allowed" : "cursor-grab active:cursor-grabbing",
      )}
    >
      <span className="flex h-full items-center justify-center" aria-hidden>
        {!isBlocked && (
          <GripVerticalIcon className="text-muted-foreground/0 group-hover:text-muted-foreground/60 size-3 transition-colors" />
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
            <AvatarFallback className="text-3xs">
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
                  "flex shrink-0 items-center gap-1 text-2xs leading-none font-medium",
                  availability.labelClass,
                )}
              >
                <span aria-hidden className={cn("size-1.5 rounded-full", availability.dotClass)} />
                {statusLabel}
              </span>
            </div>
            <p className="text-muted-foreground truncate text-2xs leading-none">
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
          <p className="text-destructive text-2xs leading-tight">{blockingReason}</p>
        )}
      </div>
    </div>
  );
}

/**
 * One availability lens. In the list it carries its driver count; on the closed trigger it
 * is the label alone, since the panel header already states how many drivers are shown.
 */
function CapacityFilterOption({
  option,
  count,
}: {
  option: (typeof CAPACITY_FILTERS)[number];
  count?: number;
}) {
  const t = useT();
  const isListed = count !== undefined;

  return (
    <span className="flex min-w-0 flex-1 items-center gap-1.5">
      {(option.availability || isListed) && (
        <span
          aria-hidden
          className={cn(
            "size-1.5 shrink-0 rounded-full",
            option.availability ? availabilityMeta(option.availability).dotClass : "bg-transparent",
          )}
        />
      )}
      <span className="truncate">{t(option.label)}</span>
      {isListed && (
        <span className="text-muted-foreground ml-auto pl-3 text-2xs tabular-nums">{count}</span>
      )}
    </span>
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
  const t = useT();

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

  const isNarrowed = filter !== "all" || search.trim() !== "";

  // Only the rows near the viewport are mounted, so a driver selected elsewhere is
  // scrolled to rather than left on a row that may not exist yet.
  const listRef = useRef<VirtualRowsHandle>(null);
  useEffect(() => {
    if (selectedWorkerId) listRef.current?.scrollToKey(selectedWorkerId);
  }, [selectedWorkerId]);

  return (
    <SectionPanel
      title={t("Capacity")}
      hint={<span className="tabular-nums">{t("{0} of {1}", visible.length, drivers.length)}</span>}
      className="min-h-0"
    >
      <div className="flex items-center gap-1.5 border-b p-2">
        <Input
          value={search}
          onChange={(event) => setDriverSearch(event.target.value)}
          placeholder={t("Search drivers")}
          leftElement={<SearchLgIcon className="text-muted-foreground size-3.5" />}
          inputContainerClassName="min-w-0 flex-1"
          className="text-xs md:text-xs"
          aria-label={t("Search drivers by name, tractor code, or fleet")}
        />
        <Select
          value={filter}
          onValueChange={(next) => {
            if (next) setCapacityFilter(next as CapacityFilter);
          }}
        >
          <SelectTrigger
            className="shrink-0 text-xs"
            aria-label={t("Filter drivers by availability")}
          >
            <SelectValue>
              {(value: CapacityFilter) => {
                const option =
                  CAPACITY_FILTERS.find((candidate) => candidate.id === value) ??
                  CAPACITY_FILTERS[0];
                return <CapacityFilterOption option={option} />;
              }}
            </SelectValue>
          </SelectTrigger>
          <SelectContent align="end" className="min-w-32 p-1">
            {CAPACITY_FILTERS.map((option) => (
              <SelectItem key={option.id} value={option.id} className="h-6 py-0 text-xs">
                <CapacityFilterOption option={option} count={counts.get(option.id) ?? 0} />
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>

      {isLoading ? (
        <div className="min-h-0 flex-1 overflow-hidden">
          <CapacityRailRowsSkeleton />
        </div>
      ) : (
        <VirtualRows
          rows={visible.map((driver) => ({
            key: driver.workerId,
            render: () => (
              <DriverRow
                driver={driver}
                isSelected={selectedWorkerId === driver.workerId}
                onSelect={onSelectDriver}
              />
            ),
          }))}
          handleRef={listRef}
          mask="card"
          maskHeight={18}
          className="min-h-0 flex-1"
          rowClassName="border-b"
          estimateSize={DRIVER_ROW_ESTIMATE_PX}
          empty={
            <div className="flex flex-col items-center gap-2 p-4">
              <p className="text-muted-foreground text-center text-xs">
                {t("No drivers match this view.")}
              </p>
              {isNarrowed && (
                <Button
                  variant="outline"
                  size="xs"
                  onClick={() => {
                    setCapacityFilter("all");
                    setDriverSearch("");
                  }}
                >
                  {t("Clear filters")}
                </Button>
              )}
            </div>
          }
        />
      )}
    </SectionPanel>
  );
}
