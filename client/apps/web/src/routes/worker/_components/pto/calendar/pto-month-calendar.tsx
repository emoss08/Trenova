import { useT } from "@trenova/shared/i18n/use-t";
import { usePermission } from "@/hooks/use-permission";
import { queries } from "@/lib/queries";
import { fetchUpcomingWorkerPTO } from "@/lib/queries/worker";
import { badgeVariants } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd, KbdGroup } from "@trenova/shared/components/ui/kbd";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatRange, getTodayDate } from "@trenova/shared/lib/date";
import { PTO_STATUS_BAR_CLASS, PTO_TYPE_META, ptoTypeMeta } from "@trenova/shared/lib/pto";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { PTOFilter, PTOType, WorkerPTO } from "@trenova/shared/types/worker";
import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { PTOFormDialog } from "../pto-form-dialog";
import { ptoWorkerName } from "../pto-worker";
import {
  buildMonthGrid,
  buildWeekSegments,
  inclusiveDaysOf,
  isWithin,
  laneCount,
  MAX_VISIBLE_LANES,
  monthEndUnix,
  monthOf,
  monthStartUnix,
  selectionRange,
  shiftMonth,
  splitVisibleSegments,
  type CalendarDay,
  type CalendarSegment,
  type CalendarWeek,
} from "./calendar-layout";
import { PTODayList, PTOSpanDetails } from "./pto-span-popover";
import { WhosOutStrip } from "./whos-out-strip";

const CALENDAR_PAGE_SIZE = 200;
const LANE_HEIGHT = 18;
const LANE_GAP = 2;
const HEADER_HEIGHT = 22;
const OVERFLOW_HEIGHT = 18;
const ROW_PADDING = 4;
const SPOTLIGHT_MS = 1600;
const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const MONTH_FORMAT = new Intl.DateTimeFormat(undefined, { month: "long" });
const MONTH_SHORT_FORMAT = new Intl.DateTimeFormat(undefined, { month: "short" });
const TYPE_ORDER = Object.keys(PTO_TYPE_META);

type SelectionRange = { start: number; end: number };

type LegendEntry = {
  type: string;
  label: string;
  dotClass: string;
  count: number;
};

export type PTOMonthCalendarProps = {
  filters: PTOFilter;
  onMonthChange: (startDate: number, endDate: number) => void;
};

function buildLegend(items: readonly WorkerPTO[]): LegendEntry[] {
  const counts = new Map<string, number>();
  for (const item of items) counts.set(item.type, (counts.get(item.type) ?? 0) + 1);
  const ordered = [...counts.keys()].sort((a, b) => {
    const ai = TYPE_ORDER.indexOf(a);
    const bi = TYPE_ORDER.indexOf(b);
    return (ai === -1 ? TYPE_ORDER.length : ai) - (bi === -1 ? TYPE_ORDER.length : bi);
  });
  return ordered.map((type) => {
    const meta = ptoTypeMeta(type);
    return { type, label: meta.label, dotClass: meta.dotClass, count: counts.get(type) ?? 0 };
  });
}

function isTypingTarget(target: EventTarget | null): boolean {
  return (
    target instanceof HTMLElement &&
    target.closest(
      "[data-slot=popover-content],[data-slot=tooltip-content],[role=dialog],[role=menu],input,textarea,select,[contenteditable=true]",
    ) !== null
  );
}

export function PTOMonthCalendar({ filters, onMonthChange }: PTOMonthCalendarProps) {
  const t = useT();

  const user = useAuthStore((state) => state.user);
  const { allowed: canCreate } = usePermission(Resource.WorkerPTO, Operation.Create);
  const { year, month } = monthOf(filters.startDate);
  const rangeStart = monthStartUnix(year, month);
  const rangeEnd = monthEndUnix(year, month);
  const todayUnix = getTodayDate();
  const today = monthOf(todayUnix);
  const isCurrentMonth = today.year === year && today.month === month;

  const query = useQuery({
    queryKey: [
      ...queries.worker.listUpcomingPTO._def,
      "calendar",
      {
        startDate: rangeStart,
        endDate: rangeEnd,
        type: filters.type,
        workerId: filters.workerId,
        fleetCodeId: filters.fleetCodeId,
        timezone: user?.timezone,
      },
    ],
    queryFn: ({ signal }) =>
      fetchUpcomingWorkerPTO(
        {
          filter: { limit: CALENDAR_PAGE_SIZE },
          type: filters.type as PTOType | undefined,
          startDate: rangeStart,
          endDate: rangeEnd,
          workerId: filters.workerId,
          fleetCodeId: filters.fleetCodeId,
          timezone: user?.timezone,
        },
        { signal },
      ),
    placeholderData: keepPreviousData,
    staleTime: 60 * 1000,
  });

  const items = useMemo(
    () =>
      (query.data?.results ?? []).filter(
        (pto) => pto.status === "Approved" || pto.status === "Requested",
      ),
    [query.data],
  );
  const legend = useMemo(() => buildLegend(items), [items]);
  const [hiddenTypes, setHiddenTypes] = useState<ReadonlySet<string>>(() => new Set());
  const visibleItems = useMemo(
    () => (hiddenTypes.size === 0 ? items : items.filter((pto) => !hiddenTypes.has(pto.type))),
    [hiddenTypes, items],
  );
  const weeks = useMemo(() => buildMonthGrid(year, month), [year, month]);

  const [anchor, setAnchor] = useState<number | null>(null);
  const [focus, setFocus] = useState<number | null>(null);
  const [requestRange, setRequestRange] = useState<SelectionRange | null>(null);
  const [activeSpanId, setActiveSpanId] = useState<string | null>(null);
  const [highlightedDay, setHighlightedDay] = useState<number | null>(null);
  const [spotlightDay, setSpotlightDay] = useState<number | null>(null);
  const dragging = useRef(false);

  const selection = useMemo<SelectionRange | null>(
    () => (anchor !== null && focus !== null ? selectionRange(anchor, focus) : null),
    [anchor, focus],
  );

  const finishSelection = useCallback(() => {
    if (!dragging.current) return;
    dragging.current = false;
    if (anchor !== null && focus !== null && canCreate) {
      setRequestRange(selectionRange(anchor, focus));
    }
    setAnchor(null);
    setFocus(null);
  }, [anchor, canCreate, focus]);

  useEffect(() => {
    window.addEventListener("mouseup", finishSelection);
    return () => window.removeEventListener("mouseup", finishSelection);
  }, [finishSelection]);

  useEffect(() => {
    if (spotlightDay === null) return;
    const timer = window.setTimeout(() => setSpotlightDay(null), SPOTLIGHT_MS);
    return () => window.clearTimeout(timer);
  }, [spotlightDay]);

  const goToMonth = useCallback(
    (delta: number) => {
      const next = shiftMonth(year, month, delta);
      onMonthChange(monthStartUnix(next.year, next.month), monthEndUnix(next.year, next.month));
    },
    [month, onMonthChange, year],
  );

  const goToToday = useCallback(() => {
    onMonthChange(monthStartUnix(today.year, today.month), monthEndUnix(today.year, today.month));
  }, [onMonthChange, today.month, today.year]);

  const selectDay = useCallback(
    (unix: number) => {
      setSpotlightDay(unix);
      const target = monthOf(unix);
      if (target.year !== year || target.month !== month) {
        onMonthChange(
          monthStartUnix(target.year, target.month),
          monthEndUnix(target.year, target.month),
        );
      }
    },
    [month, onMonthChange, year],
  );

  const toggleType = useCallback((type: string) => {
    setHiddenTypes((current) => {
      const next = new Set(current);
      if (next.has(type)) next.delete(type);
      else next.add(type);
      return next;
    });
  }, []);

  const onKeyDown = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (event.defaultPrevented || isTypingTarget(event.target)) return;
    if (event.metaKey || event.ctrlKey || event.altKey) return;
    switch (event.key) {
      case "ArrowLeft":
        event.preventDefault();
        goToMonth(-1);
        break;
      case "ArrowRight":
        event.preventDefault();
        goToMonth(1);
        break;
      case "t":
      case "T":
        event.preventDefault();
        goToToday();
        break;
      case "Escape":
        dragging.current = false;
        setAnchor(null);
        setFocus(null);
        break;
      default:
        break;
    }
  };

  const startSelection = useCallback(
    (unix: number) => {
      if (!canCreate) return;
      dragging.current = true;
      setAnchor(unix);
      setFocus(unix);
    },
    [canCreate],
  );

  const extendSelection = useCallback((unix: number) => {
    if (dragging.current) setFocus(unix);
  }, []);

  const showGrid = !query.isLoading && !query.isError;

  return (
    <div
      className="flex h-full min-h-0 flex-1 flex-col gap-2 overflow-hidden"
      data-testid="pto-month-calendar"
      onKeyDown={onKeyDown}
    >
      <WhosOutStrip
        items={items}
        todayUnix={todayUnix}
        highlightedDay={highlightedDay}
        onHighlightDay={setHighlightedDay}
        onSelectDay={selectDay}
      />

      <div className="flex shrink-0 flex-wrap items-center gap-x-2 gap-y-1">
        <div className="bg-accent/40 flex items-center gap-0.5 rounded-lg p-0.5">
          <Button
            size="icon-xs"
            variant="ghost"
            aria-label={t("Previous month")}
            onClick={() => goToMonth(-1)}
          >
            <ChevronLeftIcon className="size-3.5" />
          </Button>
          <Button
            size="xxs"
            variant="ghost"
            className="px-1.5 text-[11px]"
            disabled={isCurrentMonth}
            onClick={goToToday}
          >
            {t("Today")}
          </Button>
          <Button
            size="icon-xs"
            variant="ghost"
            aria-label={t("Next month")}
            onClick={() => goToMonth(1)}
          >
            <ChevronRightIcon className="size-3.5" />
          </Button>
        </div>
        <h4 className="text-sm leading-none font-semibold">
          {MONTH_FORMAT.format(new Date(year, month, 1))}{" "}
          <span className="text-muted-foreground font-normal tabular-nums">{year}</span>
        </h4>
        {query.isFetching && query.data ? (
          <Spinner className="text-muted-foreground size-3" aria-label={t("Refreshing")} />
        ) : null}

        <div className="ml-auto flex min-w-0 flex-wrap items-center justify-end gap-1">
          {legend.map((entry) => (
            <LegendChip
              key={entry.type}
              entry={entry}
              hidden={hiddenTypes.has(entry.type)}
              onToggle={() => toggleType(entry.type)}
            />
          ))}
          <span className="text-muted-foreground ml-1 text-[11px] tabular-nums">
            {visibleItems.length === 0
              ? "Nothing scheduled"
              : `${visibleItems.length} on the calendar`}
          </span>
          <Tooltip>
            <TooltipTrigger render={<span className="ml-1 hidden cursor-default sm:inline-flex" />}>
              <KbdGroup>
                <Kbd>←</Kbd>
                <Kbd>→</Kbd>
                <Kbd>T</Kbd>
              </KbdGroup>
            </TooltipTrigger>
            <TooltipContent side="bottom">
              {t("Arrow keys move a month and T returns to today {0}", canCreate ? ". Drag across days to request time off." : ".")}
            </TooltipContent>
          </Tooltip>
        </div>
      </div>

      {query.isLoading ? (
        <CalendarSkeleton />
      ) : query.isError ? (
        <div className="border-border flex min-h-0 flex-1 flex-col items-center justify-center gap-2 rounded-lg border border-dashed">
          <p className="text-destructive text-xs">{t("Could not load the calendar.")}</p>
          <Button size="xs" variant="outline" onClick={() => void query.refetch()}>
            {t("Try again")}
          </Button>
        </div>
      ) : null}

      {showGrid ? (
        <div className="relative flex min-h-0 flex-1 flex-col">
          <ScrollArea
            className="border-border bg-background min-h-0 flex-1 rounded-lg border"
            maskHeight={24}
            maskVariant="background"
          >
            <div
              role="grid"
              aria-label={t("PTO calendar")}
              tabIndex={0}
              className="flex min-h-full flex-col outline-none select-none"
              data-testid="pto-calendar-grid"
            >
              <div
                role="row"
                className="bg-background/95 border-border sticky top-0 z-20 grid shrink-0 grid-cols-7 border-b backdrop-blur"
              >
                {WEEKDAYS.map((weekday, index) => (
                  <div
                    key={weekday}
                    role="columnheader"
                    className={cn(
                      "text-muted-foreground px-1.5 py-1 text-[10px] font-medium tracking-wide uppercase",
                      (index === 0 || index === 6) && "text-muted-foreground/70",
                    )}
                  >
                    {weekday}
                  </div>
                ))}
              </div>
              {weeks.map((week) => (
                <WeekRow
                  key={week.key}
                  week={week}
                  items={visibleItems}
                  todayUnix={todayUnix}
                  selection={selection}
                  selectionFocus={focus}
                  activeSpanId={activeSpanId}
                  highlightedDay={highlightedDay}
                  spotlightDay={spotlightDay}
                  canCreate={canCreate}
                  onStartSelection={startSelection}
                  onExtendSelection={extendSelection}
                  onActiveSpanChange={setActiveSpanId}
                />
              ))}
            </div>
          </ScrollArea>
          {visibleItems.length === 0 ? (
            <p className="text-muted-foreground/70 pointer-events-none absolute inset-x-0 bottom-2 z-20 text-center text-[11px]">
              {hiddenTypes.size > 0
                ? "Every type on this month is hidden by the legend"
                : canCreate
                  ? "No time off this month · drag across days to request some"
                  : "No time off this month"}
            </p>
          ) : null}
        </div>
      ) : null}

      {requestRange ? (
        <PTOFormDialog
          open
          onOpenChange={(open) => {
            if (!open) setRequestRange(null);
          }}
          defaultRange={requestRange}
        />
      ) : null}
    </div>
  );
}

function CalendarSkeleton() {
  return (
    <div
      className="border-border grid min-h-0 flex-1 grid-cols-7 gap-1 rounded-lg border p-2"
      aria-busy
      data-testid="pto-calendar-skeleton"
    >
      {Array.from({ length: 35 }, (_, index) => (
        <Skeleton key={index} className="h-10 w-full" />
      ))}
    </div>
  );
}

function LegendChip({
  entry,
  hidden,
  onToggle,
}: {
  entry: LegendEntry;
  hidden: boolean;
  onToggle: () => void;
}) {
  const t = useT();

  return (
    <button
      type="button"
      aria-pressed={!hidden}
      aria-label={t(entry.label)}
      title={hidden ? `Show ${entry.label}` : `Hide ${entry.label}`}
      onClick={onToggle}
      className={cn(
        "focus-visible:ring-ring/50 inline-flex h-5.5 items-center gap-1 rounded-md border px-1.5 text-[11px] font-medium transition-colors outline-none focus-visible:ring-[3px]",
        hidden
          ? "text-muted-foreground/60 border-transparent line-through decoration-1"
          : "bg-accent/40 border-border/60 hover:bg-accent",
      )}
    >
      <span
        className={cn("size-1.5 shrink-0 rounded-full", entry.dotClass, hidden && "opacity-40")}
        aria-hidden
      />
      <span>{t(entry.label)}</span>
      <span className="text-muted-foreground tabular-nums">{entry.count}</span>
    </button>
  );
}

type WeekRowProps = {
  week: CalendarWeek;
  items: readonly WorkerPTO[];
  todayUnix: number;
  selection: SelectionRange | null;
  selectionFocus: number | null;
  activeSpanId: string | null;
  highlightedDay: number | null;
  spotlightDay: number | null;
  canCreate: boolean;
  onStartSelection: (unix: number) => void;
  onExtendSelection: (unix: number) => void;
  onActiveSpanChange: (id: string | null) => void;
};

function WeekRow({
  week,
  items,
  todayUnix,
  selection,
  selectionFocus,
  activeSpanId,
  highlightedDay,
  spotlightDay,
  canCreate,
  onStartSelection,
  onExtendSelection,
  onActiveSpanChange,
}: WeekRowProps) {
  const { visible, overflow, lanes } = useMemo(() => {
    const segments = buildWeekSegments(items, week);
    const split = splitVisibleSegments(segments, MAX_VISIBLE_LANES);
    return { ...split, lanes: Math.min(Math.max(laneCount(segments), 1), MAX_VISIBLE_LANES) };
  }, [items, week]);
  const hasOverflow = overflow.some((count) => count > 0);
  const overflowTop = HEADER_HEIGHT + lanes * (LANE_HEIGHT + LANE_GAP);
  const minHeight = overflowTop + (hasOverflow ? OVERFLOW_HEIGHT : 0) + ROW_PADDING;

  return (
    <div
      role="row"
      className="border-border relative grid flex-1 grid-cols-7 border-b last:border-b-0"
      style={{ minHeight }}
    >
      {week.days.map((day, col) => (
        <DayCell
          key={day.key}
          day={day}
          items={items}
          selected={isWithin(day.unix, selection)}
          selectionDays={
            selection && selectionFocus === day.unix
              ? inclusiveDaysOf(selection.start, selection.end)
              : null
          }
          highlighted={highlightedDay === day.unix}
          spotlight={spotlightDay === day.unix}
          canCreate={canCreate}
          overflow={overflow[col]}
          overflowTop={overflowTop}
          onMouseDown={() => onStartSelection(day.unix)}
          onMouseEnter={() => onExtendSelection(day.unix)}
        />
      ))}
      {visible.map((segment) => (
        <SpanBar
          key={`${segment.item.id}-${week.key}`}
          segment={segment}
          todayUnix={todayUnix}
          active={activeSpanId === segment.item.id}
          onActiveChange={onActiveSpanChange}
        />
      ))}
    </div>
  );
}

function DayCell({
  day,
  items,
  selected,
  selectionDays,
  highlighted,
  spotlight,
  canCreate,
  overflow,
  overflowTop,
  onMouseDown,
  onMouseEnter,
}: {
  day: CalendarDay;
  items: readonly WorkerPTO[];
  selected: boolean;
  selectionDays: number | null;
  highlighted: boolean;
  spotlight: boolean;
  canCreate: boolean;
  overflow: number;
  overflowTop: number;
  onMouseDown: () => void;
  onMouseEnter: () => void;
}) {
  const t = useT();

  const firstOfMonth = day.date === 1;

  return (
    <div
      role="gridcell"
      data-testid={`pto-day-${day.key}`}
      data-selected={selected ? "true" : undefined}
      data-spotlight={spotlight ? "true" : undefined}
      aria-selected={selected}
      onMouseDown={onMouseDown}
      onMouseEnter={onMouseEnter}
      className={cn(
        "border-border/70 relative border-r px-1 pt-0.5 text-[11px] transition-colors last:border-r-0",
        !day.inMonth && "text-muted-foreground/50 bg-muted/25",
        day.isWeekend && day.inMonth && "bg-muted/10",
        canCreate && "hover:bg-accent/40 cursor-pointer",
        highlighted && "bg-primary/5",
        selected && "bg-primary/10",
        spotlight && "ring-primary/50 animate-in fade-in-0 ring-2 duration-300 ring-inset",
      )}
    >
      <div className="flex items-center justify-between" style={{ height: HEADER_HEIGHT - 2 }}>
        <span
          className={cn(
            "inline-flex h-5 min-w-5 items-center justify-center rounded-full px-1 leading-none tabular-nums",
            day.isToday && "bg-primary text-primary-foreground font-semibold",
            firstOfMonth && !day.isToday && "font-semibold",
          )}
        >
          {firstOfMonth
            ? `${MONTH_SHORT_FORMAT.format(new Date(day.year, day.month, 1))} 1`
            : day.date}
        </span>
        {selectionDays !== null ? (
          <span
            data-testid="pto-selection-pill"
            className="bg-primary text-primary-foreground animate-in fade-in-0 zoom-in-95 z-30 rounded-full px-1.5 py-0.5 text-[10px] leading-none font-semibold shadow-sm duration-150"
          >
            {t("{0} day{1}", selectionDays, selectionDays === 1 ? "" : "s")}
          </span>
        ) : null}
      </div>
      {overflow > 0 ? (
        <DayOverflow day={day} items={items} count={overflow} top={overflowTop} />
      ) : null}
    </div>
  );
}

function DayOverflow({
  day,
  items,
  count,
  top,
}: {
  day: CalendarDay;
  items: readonly WorkerPTO[];
  count: number;
  top: number;
}) {
  const t = useT();

  return (
    <Popover>
      <PopoverTrigger
        render={
          <button
            type="button"
            data-testid={`pto-overflow-${day.key}`}
            aria-label={`${count} more on ${day.key}`}
            onMouseDown={(event) => event.stopPropagation()}
            className="text-muted-foreground hover:bg-accent hover:text-foreground absolute left-1 z-10 inline-flex h-4 items-center rounded px-1 text-[10px] font-medium tabular-nums transition-colors"
            style={{ top }}
          />
        }
      >
        {t("+{0} more", count)}
      </PopoverTrigger>
      <PopoverContent align="start" className="w-72">
        <PTODayList items={items} dayUnix={day.unix} />
      </PopoverContent>
    </Popover>
  );
}

function SpanBar({
  segment,
  todayUnix,
  active,
  onActiveChange,
}: {
  segment: CalendarSegment<WorkerPTO>;
  todayUnix: number;
  active: boolean;
  onActiveChange: (id: string | null) => void;
}) {
  const pto = segment.item;
  const meta = ptoTypeMeta(pto.type);
  const left = `${(segment.startCol / 7) * 100}%`;
  const width = `${((segment.endCol - segment.startCol + 1) / 7) * 100}%`;
  const top = HEADER_HEIGHT + segment.lane * (LANE_HEIGHT + LANE_GAP);
  const name = ptoWorkerName(pto);
  const summary = `${name} · ${meta.label} · ${formatRange(pto.startDate, pto.endDate)}`;

  return (
    <Popover>
      <PopoverTrigger
        render={
          <button
            type="button"
            data-testid={`pto-span-${pto.id}`}
            data-active={active ? "true" : undefined}
            aria-label={pto.status === "Requested" ? `${summary} (requested)` : summary}
            title={summary}
            onMouseDown={(event) => event.stopPropagation()}
            onMouseEnter={() => onActiveChange(pto.id ?? null)}
            onMouseLeave={() => onActiveChange(null)}
            onFocus={() => onActiveChange(pto.id ?? null)}
            onBlur={() => onActiveChange(null)}
            className={cn(
              badgeVariants({ variant: meta.badgeVariant }),
              "absolute z-10 h-[18px] w-auto justify-start gap-1 rounded-md px-1.5 text-[11px] leading-none transition-[box-shadow,filter,opacity] outline-none",
              PTO_STATUS_BAR_CLASS[pto.status],
              segment.continuesBefore && "rounded-l-none border-l-0",
              segment.continuesAfter && "rounded-r-none border-r-0",
              active && "ring-ring/40 z-20 shadow-sm ring-2 brightness-95",
            )}
            style={{
              left: `calc(${left} + 2px)`,
              width: `calc(${width} - 4px)`,
              top,
              height: LANE_HEIGHT,
            }}
          />
        }
      >
        <span className="min-w-0 truncate">{name}</span>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-80">
        <PTOSpanDetails pto={pto} todayUnix={todayUnix} />
      </PopoverContent>
    </Popover>
  );
}
