import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge, type BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { RingGauge, type RingGaugeTone } from "@trenova/shared/components/ui/ring-gauge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import type {
  WorkerFormSubmission,
  WorkerHosDailyLog,
  WorkerHosLogEntry,
  WorkerHosState,
  WorkerHosViolation,
} from "@/lib/graphql/telematics";
import { queries } from "@/lib/queries";
import {
  formatClockDurationMs,
  formatDurationMs,
  formatUnixDate,
  formatUnixDateTime,
  formatUnixTime,
} from "@trenova/shared/lib/date";
import { cn, metersToMiles, pluralize, toTitleCase } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { format, formatDistanceToNowStrict, startOfDay, subDays } from "date-fns";
import {
  CableIcon,
  CalendarRangeIcon,
  ChevronDownIcon,
  ClipboardListIcon,
  ListXIcon,
  OctagonAlertIcon,
  ShieldCheckIcon,
  TruckIcon,
  UserRoundXIcon,
} from "lucide-react";
import { useMemo, useRef, useState } from "react";
import { Link } from "react-router";

const HOUR_MS = 3_600_000;
const REFETCH_INTERVAL_MS = 60_000;
const STALE_TIME_MS = 60_000;
const DAY_SECONDS = 86_400;
const THIRTY_DAYS_SECONDS = 30 * DAY_SECONDS;
const DAILY_LOG_DAYS = 7;
const DAY_KEY_FORMAT = "yyyy-MM-dd";

const ELD_GRAPH_WIDTH = 1_440;
const ELD_GRAPH_HEIGHT = 128;
const ELD_LANE_COUNT = 4;
const ELD_LANE_HEIGHT = ELD_GRAPH_HEIGHT / ELD_LANE_COUNT;

const eldLaneLabels = ["OFF", "SB", "D", "ON"] as const;

const eldLaneByStatus: Record<string, number> = {
  offDuty: 0,
  personalConveyance: 0,
  sleeperBed: 1,
  driving: 2,
  onDuty: 3,
  yardMove: 3,
};

const eldStrokeClassByStatus: Record<string, string> = {
  offDuty: "text-muted-foreground",
  personalConveyance: "text-muted-foreground",
  sleeperBed: "text-purple-600 dark:text-purple-400",
  driving: "text-brand",
  onDuty: "text-warning",
  yardMove: "text-warning",
};

const DRIVE_LIMIT_MS = 11 * HOUR_MS;
const SHIFT_LIMIT_MS = 14 * HOUR_MS;
const CYCLE_LIMIT_MS = 70 * HOUR_MS;
const BREAK_LIMIT_MS = 8 * HOUR_MS;

const dutyStatusMeta: Record<string, { label: string; variant: BadgeVariant }> = {
  driving: { label: "Driving", variant: "info" },
  onDuty: { label: "On Duty", variant: "warning" },
  offDuty: { label: "Off Duty", variant: "secondary" },
  sleeperBed: { label: "Sleeper Berth", variant: "purple" },
  yardMove: { label: "Yard Move", variant: "teal" },
  personalConveyance: { label: "Personal Conveyance", variant: "teal" },
};

function getDutyStatusMeta(dutyStatus: string | null): { label: string; variant: BadgeVariant } {
  if (!dutyStatus) {
    return { label: "Unknown", variant: "secondary" };
  }
  return dutyStatusMeta[dutyStatus] ?? { label: toTitleCase(dutyStatus), variant: "secondary" };
}

function limitLabel(limitMs: number): string {
  const hours = Math.round(limitMs / HOUR_MS);
  return `${hours}h limit`;
}

function clockTone(remainingMs: number, baseTone: RingGaugeTone): RingGaugeTone {
  if (remainingMs < HOUR_MS) {
    return "critical";
  }
  if (remainingMs < 2 * HOUR_MS) {
    return "warning";
  }
  return baseTone;
}

type HosDaySlot = {
  key: string;
  label: string;
  isToday: boolean;
  startAt: number;
  endAt: number;
  dailyLog: WorkerHosDailyLog | null;
};

function buildDaySlots(dailyLogs: WorkerHosDailyLog[]): HosDaySlot[] {
  const logsByDay = new Map<string, WorkerHosDailyLog>();
  for (const log of dailyLogs) {
    logsByDay.set(format(new Date(log.startAt * 1000), DAY_KEY_FORMAT), log);
  }
  const today = startOfDay(new Date());
  const slots: HosDaySlot[] = [];
  for (let offset = DAILY_LOG_DAYS - 1; offset >= 0; offset -= 1) {
    const date = subDays(today, offset);
    const key = format(date, DAY_KEY_FORMAT);
    const dailyLog = logsByDay.get(key) ?? null;
    const fallbackStart = Math.floor(date.getTime() / 1000);
    slots.push({
      key,
      label: offset === 0 ? "Today" : format(date, "EEE d"),
      isToday: offset === 0,
      startAt: dailyLog?.startAt ?? fallbackStart,
      endAt: dailyLog?.endAt ?? fallbackStart + DAY_SECONDS,
      dailyLog,
    });
  }
  return slots;
}

type EldSegment = {
  entry: WorkerHosLogEntry;
  lane: number;
  startSec: number;
  endSec: number;
  ongoing: boolean;
};

function buildEldSegments(
  sortedEntries: WorkerHosLogEntry[],
  dayStart: number,
  dayEnd: number,
  nowSec: number,
): EldSegment[] {
  const ongoingCap = Math.min(nowSec, dayEnd);
  const segments: EldSegment[] = [];
  for (const entry of sortedEntries) {
    const startSec = Math.max(entry.logStartAt, dayStart);
    const endSec = Math.min(entry.logEndAt ?? ongoingCap, dayEnd);
    if (endSec <= startSec) {
      continue;
    }
    segments.push({
      entry,
      lane: eldLaneByStatus[entry.hosStatusType] ?? 3,
      startSec,
      endSec,
      ongoing: entry.logEndAt === null || entry.logEndAt === undefined,
    });
  }
  return segments;
}

function buildLaneTotalsMs(dailyLog: WorkerHosDailyLog | null, segments: EldSegment[]): number[] {
  if (dailyLog) {
    return [
      dailyLog.offDutyDurationMs + dailyLog.personalConveyanceDurationMs,
      dailyLog.sleeperBerthDurationMs,
      dailyLog.driveDurationMs,
      dailyLog.onDutyDurationMs + dailyLog.yardMoveDurationMs,
    ];
  }
  const totals = [0, 0, 0, 0];
  for (const segment of segments) {
    totals[segment.lane] += (segment.endSec - segment.startSec) * 1000;
  }
  return totals;
}

function HosClockCard({
  label,
  limitLabel,
  remainingMs,
  limitMs,
  baseTone = "brand",
  extra,
}: {
  label: string;
  limitLabel: string;
  remainingMs: number;
  limitMs: number;
  baseTone?: RingGaugeTone;
  extra?: string;
}) {
  const clamped = Math.max(remainingMs, 0);

  return (
    <div className="flex flex-col items-center gap-2.5 rounded-lg border border-border px-3 py-4">
      <RingGauge
        value={clamped / limitMs}
        size={104}
        strokeWidth={7}
        tone={clockTone(clamped, baseTone)}
        aria-label={`${label} time remaining`}
      >
        <span className="text-2xl font-semibold tabular-nums">
          {formatClockDurationMs(clamped)}
        </span>
      </RingGauge>
      <div className="text-center">
        <p className="text-sm font-medium">{label}</p>
        <p className="text-[11px] text-muted-foreground">{limitLabel}</p>
        {extra ? <p className="mt-0.5 text-[11px] text-muted-foreground">{extra}</p> : null}
      </div>
    </div>
  );
}

function HosEmptyState({
  icon,
  title,
  description,
  action,
}: {
  icon: React.ReactNode;
  title: string;
  description: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-dashed p-6 text-center">
      {icon}
      <p className="mt-2 text-sm font-medium">{title}</p>
      <p className="mx-auto mt-1 max-w-md text-xs text-muted-foreground">{description}</p>
      {action}
    </div>
  );
}

function HosErrorState({ message, onRetry }: { message: string; onRetry: () => void }) {
  return (
    <div className="rounded-lg border border-dashed p-6 text-center">
      <OctagonAlertIcon className="mx-auto size-5 text-destructive" />
      <p className="mt-2 text-sm font-medium">{message}</p>
      <Button type="button" variant="outline" size="sm" className="mt-3" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

function HosLoadingState() {
  return (
    <div className="flex flex-col gap-3">
      <Skeleton className="h-7 w-2/3 rounded-md" />
      <div className="grid grid-cols-2 gap-2 xl:grid-cols-4">
        <Skeleton className="h-44 w-full rounded-lg" />
        <Skeleton className="h-44 w-full rounded-lg" />
        <Skeleton className="h-44 w-full rounded-lg" />
        <Skeleton className="h-44 w-full rounded-lg" />
      </div>
      <Skeleton className="h-28 w-full rounded-lg" />
    </div>
  );
}

function ActiveViolationAlert({ state }: { state: WorkerHosState }) {
  if (state.shiftDrivingViolationMs <= 0 && state.cycleViolationMs <= 0) {
    return null;
  }

  const parts: string[] = [];
  if (state.shiftDrivingViolationMs > 0) {
    parts.push(
      `Shift driving limit exceeded by ${formatDurationMs(state.shiftDrivingViolationMs)}`,
    );
  }
  if (state.cycleViolationMs > 0) {
    parts.push(`Cycle limit exceeded by ${formatDurationMs(state.cycleViolationMs)}`);
  }

  return (
    <Alert variant="destructive">
      <OctagonAlertIcon />
      <AlertTitle>Active HOS violation</AlertTitle>
      <AlertDescription>{parts.join(". ")}.</AlertDescription>
    </Alert>
  );
}

function ViolationRow({ violation }: { violation: WorkerHosViolation }) {
  return (
    <li className="flex items-center justify-between gap-3 px-4 py-2.5">
      <div className="min-w-0">
        <p className="truncate text-sm font-medium">{toTitleCase(violation.violationType)}</p>
        {violation.description ? (
          <p className="truncate text-xs text-muted-foreground">{violation.description}</p>
        ) : null}
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <span className="text-xs text-muted-foreground">
          {formatUnixDate(violation.violationStartAt)}
        </span>
        <Badge variant="inactive">{formatDurationMs(violation.durationMs)}</Badge>
      </div>
    </li>
  );
}

function ViolationsSection({ workerId, since }: { workerId: string; since: number }) {
  const violationsQuery = useQuery({
    ...queries.telematics.workerHosViolations(workerId, since),
    enabled: workerId.length > 0,
    refetchInterval: REFETCH_INTERVAL_MS,
  });

  return (
    <div className="flex flex-col gap-2">
      <div>
        <h3 className="text-sm font-semibold">Violations</h3>
        <p className="text-xs text-muted-foreground">
          Hours-of-service violations detected in the last 30 days.
        </p>
      </div>
      {violationsQuery.isPending ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-12 w-full rounded-lg" />
          <Skeleton className="h-12 w-full rounded-lg" />
        </div>
      ) : violationsQuery.isError ? (
        <HosErrorState
          message="Violations could not be loaded"
          onRetry={() => void violationsQuery.refetch()}
        />
      ) : violationsQuery.data.length === 0 ? (
        <HosEmptyState
          icon={<ShieldCheckIcon className="mx-auto size-5 text-green-600 dark:text-green-400" />}
          title="No violations in the last 30 days"
          description="This driver has a clean hours-of-service record for the past month."
        />
      ) : (
        <div className="rounded-lg border border-border">
          <ul className="divide-y divide-border">
            {violationsQuery.data.map((violation) => (
              <ViolationRow
                key={`${violation.violationType}-${violation.violationStartAt}-${violation.detectedAt}`}
                violation={violation}
              />
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}

function DayPillButton({
  slot,
  isSelected,
  onSelect,
}: {
  slot: HosDaySlot;
  isSelected: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onSelect}
      aria-pressed={isSelected}
      className={cn(
        "flex shrink-0 items-center gap-1.5 rounded-full border px-3 py-1 text-xs font-medium transition-colors",
        isSelected
          ? "border-transparent bg-primary text-primary-foreground"
          : "border-border text-muted-foreground hover:bg-muted hover:text-foreground",
      )}
    >
      {slot.label}
      {slot.dailyLog?.isCertified ? (
        <span
          aria-label="Certified"
          className={cn(
            "size-1.5 rounded-full",
            isSelected ? "bg-primary-foreground" : "bg-green-600 dark:bg-green-400",
          )}
        />
      ) : null}
    </button>
  );
}

function EldGraph({
  segments,
  dayStart,
  dayEnd,
  laneTotals,
}: {
  segments: EldSegment[];
  dayStart: number;
  dayEnd: number;
  laneTotals: number[];
}) {
  const containerRef = useRef<HTMLDivElement>(null);
  const [hover, setHover] = useState<{ index: number; x: number; y: number } | null>(null);

  const windowSec = Math.max(dayEnd - dayStart, 1);
  const toX = (sec: number) => ((sec - dayStart) / windowSec) * ELD_GRAPH_WIDTH;
  const laneY = (lane: number) => lane * ELD_LANE_HEIGHT + ELD_LANE_HEIGHT / 2;

  const hourTicks = useMemo(() => {
    const ticks: { key: number; fraction: number; hour: number; isMajor: boolean }[] = [];
    for (let h = 0; h < 24; h += 2) {
      const fraction = h / 24;
      ticks.push({
        key: h,
        fraction,
        hour: new Date((dayStart + fraction * windowSec) * 1000).getHours(),
        isMajor: h % 4 === 0,
      });
    }
    return ticks;
  }, [dayStart, windowSec]);

  const handleSegmentMove = (event: React.MouseEvent<SVGLineElement>, index: number) => {
    const rect = containerRef.current?.getBoundingClientRect();
    if (!rect) {
      return;
    }
    setHover({
      index,
      x: Math.min(Math.max(event.clientX - rect.left, 72), Math.max(rect.width - 72, 72)),
      y: event.clientY - rect.top,
    });
  };

  const hovered = hover ? segments[hover.index] : null;
  const hoveredMeta = hovered ? getDutyStatusMeta(hovered.entry.hosStatusType) : null;

  return (
    <div className="flex gap-2">
      <div className="flex w-8 shrink-0 flex-col">
        <div className="h-4" />
        {eldLaneLabels.map((label) => (
          <div
            key={label}
            className="flex h-8 items-center justify-end text-[10px] font-medium text-muted-foreground"
          >
            {label}
          </div>
        ))}
      </div>
      <div className="min-w-0 flex-1">
        <div className="relative h-4">
          {hourTicks.map((tick) => (
            <span
              key={tick.key}
              className={cn(
                "absolute top-0 -translate-x-1/2 text-[10px] tabular-nums text-muted-foreground",
                !tick.isMajor && "max-sm:hidden",
              )}
              style={{ left: `${tick.fraction * 100}%` }}
            >
              {tick.hour}
            </span>
          ))}
        </div>
        <div ref={containerRef} className="relative">
          <svg
            viewBox={`0 0 ${ELD_GRAPH_WIDTH} ${ELD_GRAPH_HEIGHT}`}
            preserveAspectRatio="none"
            className="block h-32 w-full"
            role="img"
            aria-label="Duty status graph for the selected day"
          >
            {Array.from({ length: ELD_LANE_COUNT + 1 }, (_, lane) => (
              <line
                key={`lane-${lane}`}
                x1={0}
                x2={ELD_GRAPH_WIDTH}
                y1={lane * ELD_LANE_HEIGHT}
                y2={lane * ELD_LANE_HEIGHT}
                stroke="currentColor"
                strokeWidth={1}
                vectorEffect="non-scaling-stroke"
                className="text-border"
              />
            ))}
            {Array.from({ length: 23 }, (_, i) => i + 1).map((h) => (
              <line
                key={`hour-${h}`}
                x1={(h / 24) * ELD_GRAPH_WIDTH}
                x2={(h / 24) * ELD_GRAPH_WIDTH}
                y1={0}
                y2={ELD_GRAPH_HEIGHT}
                stroke="currentColor"
                strokeWidth={1}
                vectorEffect="non-scaling-stroke"
                className={h % 4 === 0 ? "text-border" : "text-border/50"}
              />
            ))}
            {segments.map((segment, index) => {
              const previous = index > 0 ? segments[index - 1] : null;
              if (!previous || previous.lane === segment.lane) {
                return null;
              }
              return (
                <line
                  key={`connector-${segment.entry.logStartAt}-${index}`}
                  x1={toX(segment.startSec)}
                  x2={toX(segment.startSec)}
                  y1={laneY(previous.lane)}
                  y2={laneY(segment.lane)}
                  stroke="currentColor"
                  strokeWidth={1}
                  vectorEffect="non-scaling-stroke"
                  className="text-muted-foreground/60"
                />
              );
            })}
            {segments.map((segment, index) => (
              <line
                key={`segment-${segment.entry.logStartAt}-${index}`}
                x1={toX(segment.startSec)}
                x2={toX(segment.endSec)}
                y1={laneY(segment.lane)}
                y2={laneY(segment.lane)}
                stroke="currentColor"
                strokeWidth={2.5}
                strokeLinecap="round"
                vectorEffect="non-scaling-stroke"
                className={eldStrokeClassByStatus[segment.entry.hosStatusType] ?? "text-warning"}
              />
            ))}
            {segments.map((segment, index) => (
              <line
                key={`hit-${segment.entry.logStartAt}-${index}`}
                x1={toX(segment.startSec)}
                x2={toX(segment.endSec)}
                y1={laneY(segment.lane)}
                y2={laneY(segment.lane)}
                stroke="transparent"
                strokeWidth={16}
                vectorEffect="non-scaling-stroke"
                pointerEvents="stroke"
                onMouseMove={(event) => handleSegmentMove(event, index)}
                onMouseLeave={() => setHover(null)}
              />
            ))}
          </svg>
          {hover && hovered && hoveredMeta ? (
            <div
              className="pointer-events-none absolute z-10 -translate-x-1/2 -translate-y-full rounded-md bg-foreground px-3 py-1.5 text-xs text-background shadow-md"
              style={{ left: hover.x, top: hover.y - 8 }}
            >
              <p className="font-medium">{hoveredMeta.label}</p>
              <p className="tabular-nums text-background/80">
                {formatUnixTime(hovered.startSec)} –{" "}
                {hovered.ongoing ? "Ongoing" : formatUnixTime(hovered.endSec)} ·{" "}
                {formatDurationMs((hovered.endSec - hovered.startSec) * 1000)}
              </p>
              {hovered.entry.vehicleName ? (
                <p className="text-background/80">{hovered.entry.vehicleName}</p>
              ) : null}
              {hovered.entry.remark ? (
                <p className="max-w-56 truncate text-background/80">{hovered.entry.remark}</p>
              ) : null}
            </div>
          ) : null}
        </div>
      </div>
      <div className="flex w-14 shrink-0 flex-col">
        <div className="h-4" />
        {laneTotals.map((totalMs, lane) => (
          <div
            key={eldLaneLabels[lane]}
            className="flex h-8 items-center justify-end text-[10px] tabular-nums text-muted-foreground"
          >
            {formatDurationMs(totalMs)}
          </div>
        ))}
      </div>
    </div>
  );
}

function DailySummaryRow({ dailyLog }: { dailyLog: WorkerHosDailyLog }) {
  const chips: { label: string; value: string }[] = [
    { label: "Drive", value: formatDurationMs(dailyLog.driveDurationMs) },
    { label: "On duty", value: formatDurationMs(dailyLog.onDutyDurationMs) },
    { label: "Off duty", value: formatDurationMs(dailyLog.offDutyDurationMs) },
    { label: "Sleeper", value: formatDurationMs(dailyLog.sleeperBerthDurationMs) },
    { label: "Distance", value: `${metersToMiles(dailyLog.driveDistanceMeters).toFixed(1)} mi` },
  ];
  if (dailyLog.vehicleNames && dailyLog.vehicleNames.length > 0) {
    chips.push({
      label: pluralize("Vehicle", dailyLog.vehicleNames.length),
      value: dailyLog.vehicleNames.join(", "),
    });
  }
  if (dailyLog.shippingDocs) {
    chips.push({ label: "Shipping docs", value: dailyLog.shippingDocs });
  }

  const certifiedBadge = dailyLog.isCertified ? (
    <Badge variant="active">Certified</Badge>
  ) : (
    <Badge variant="outline">Uncertified</Badge>
  );

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {chips.map((chip) => (
        <div
          key={chip.label}
          className="flex items-center gap-1.5 rounded-md border border-border px-2 py-1"
        >
          <span className="text-[11px] text-muted-foreground">{chip.label}</span>
          <span className="text-xs font-medium tabular-nums">{chip.value}</span>
        </div>
      ))}
      {dailyLog.isCertified && dailyLog.certifiedAt ? (
        <Tooltip>
          <TooltipTrigger render={certifiedBadge} />
          <TooltipContent>Certified {formatUnixDateTime(dailyLog.certifiedAt)}</TooltipContent>
        </Tooltip>
      ) : (
        certifiedBadge
      )}
    </div>
  );
}

function HosLogEntryRow({ entry, nowCap }: { entry: WorkerHosLogEntry; nowCap: number }) {
  const meta = getDutyStatusMeta(entry.hosStatusType);
  const endAt = entry.logEndAt ?? null;
  const durationMs = Math.max((endAt ?? nowCap) - entry.logStartAt, 0) * 1000;

  return (
    <li className="flex items-center gap-3 px-4 py-2.5">
      <div className="w-36 shrink-0">
        <Badge variant={meta.variant}>{meta.label}</Badge>
      </div>
      <div className="min-w-0 flex-1">
        <p className="text-sm tabular-nums">
          {formatUnixTime(entry.logStartAt)} – {endAt ? formatUnixTime(endAt) : "Ongoing"}
        </p>
        {entry.vehicleName || entry.remark ? (
          <div className="flex min-w-0 items-center gap-1 text-xs text-muted-foreground">
            {entry.vehicleName ? <span className="shrink-0">{entry.vehicleName}</span> : null}
            {entry.vehicleName && entry.remark ? <span className="shrink-0">·</span> : null}
            {entry.remark ? (
              <Tooltip>
                <TooltipTrigger render={<span className="truncate">{entry.remark}</span>} />
                <TooltipContent>{entry.remark}</TooltipContent>
              </Tooltip>
            ) : null}
          </div>
        ) : null}
      </div>
      <Badge variant="secondary">{formatDurationMs(durationMs)}</Badge>
    </li>
  );
}

function DailyLogsSkeleton() {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex gap-1.5">
        {Array.from({ length: DAILY_LOG_DAYS }, (_, index) => (
          <Skeleton key={index} className="h-6 w-16 rounded-full" />
        ))}
      </div>
      <Skeleton className="h-40 w-full rounded-lg" />
      <Skeleton className="h-24 w-full rounded-lg" />
    </div>
  );
}

function DailyLogsSection({ workerId }: { workerId: string }) {
  const [dateRange] = useState(() => {
    const today = new Date();
    return {
      startDate: format(subDays(today, DAILY_LOG_DAYS - 1), DAY_KEY_FORMAT),
      endDate: format(today, DAY_KEY_FORMAT),
    };
  });
  const [selectedDayKey, setSelectedDayKey] = useState<string | null>(null);

  const isTodaySelected = (selectedDayKey ?? dateRange.endDate) === dateRange.endDate;

  const dailyLogsQuery = useQuery({
    ...queries.telematics.workerHosDailyLogs(workerId, dateRange.startDate, dateRange.endDate),
    enabled: workerId.length > 0,
    staleTime: STALE_TIME_MS,
    refetchInterval: isTodaySelected ? REFETCH_INTERVAL_MS : false,
  });

  const daySlots = useMemo(() => buildDaySlots(dailyLogsQuery.data ?? []), [dailyLogsQuery.data]);
  const selectedDay =
    daySlots.find((slot) => slot.key === selectedDayKey) ?? daySlots[daySlots.length - 1];
  const hasHistory = (dailyLogsQuery.data?.length ?? 0) > 0;

  const logsQuery = useQuery({
    ...queries.telematics.workerHosLogs(workerId, selectedDay.startAt, selectedDay.endAt),
    enabled: workerId.length > 0 && hasHistory,
    staleTime: STALE_TIME_MS,
    refetchInterval: selectedDay.isToday ? REFETCH_INTERVAL_MS : false,
  });

  const nowSec = Math.floor(Date.now() / 1000);
  const sortedEntries = [...(logsQuery.data ?? [])].sort((a, b) => a.logStartAt - b.logStartAt);
  const segments = buildEldSegments(sortedEntries, selectedDay.startAt, selectedDay.endAt, nowSec);
  const laneTotals = buildLaneTotalsMs(selectedDay.dailyLog, segments);
  const ongoingCap = Math.min(nowSec, selectedDay.endAt);

  return (
    <div className="flex flex-col gap-2">
      <div>
        <h3 className="text-sm font-semibold">Daily Logs</h3>
        <p className="text-xs text-muted-foreground">
          Duty status graph and log entries for the last 7 days.
        </p>
      </div>
      {dailyLogsQuery.isPending ? (
        <DailyLogsSkeleton />
      ) : dailyLogsQuery.isError ? (
        <HosErrorState
          message="Daily logs could not be loaded"
          onRetry={() => void dailyLogsQuery.refetch()}
        />
      ) : !hasHistory ? (
        <HosEmptyState
          icon={<CalendarRangeIcon className="mx-auto size-5 text-muted-foreground" />}
          title="No log history yet"
          description="Daily logs appear once Samsara reports driver activity."
        />
      ) : (
        <>
          <div className="flex flex-wrap items-center gap-1.5">
            {daySlots.map((slot) => (
              <DayPillButton
                key={slot.key}
                slot={slot}
                isSelected={slot.key === selectedDay.key}
                onSelect={() => setSelectedDayKey(slot.key)}
              />
            ))}
          </div>
          {logsQuery.isPending ? (
            <div className="flex flex-col gap-2">
              <Skeleton className="h-40 w-full rounded-lg" />
              <Skeleton className="h-24 w-full rounded-lg" />
            </div>
          ) : logsQuery.isError ? (
            <HosErrorState
              message="Duty status logs could not be loaded"
              onRetry={() => void logsQuery.refetch()}
            />
          ) : (
            <>
              <div className="rounded-lg border border-border p-3">
                <EldGraph
                  segments={segments}
                  dayStart={selectedDay.startAt}
                  dayEnd={selectedDay.endAt}
                  laneTotals={laneTotals}
                />
              </div>
              {selectedDay.dailyLog ? (
                <DailySummaryRow dailyLog={selectedDay.dailyLog} />
              ) : (
                <p className="text-xs text-muted-foreground">
                  No daily summary reported for this day.
                </p>
              )}
              {sortedEntries.length === 0 ? (
                <HosEmptyState
                  icon={<ListXIcon className="mx-auto size-5 text-muted-foreground" />}
                  title="No duty status changes recorded for this day."
                  description="Entries appear here as Samsara logs duty status transitions."
                />
              ) : (
                <div className="rounded-lg border border-border">
                  <ul className="divide-y divide-border">
                    {sortedEntries.map((entry) => (
                      <HosLogEntryRow
                        key={`${entry.hosStatusType}-${entry.logStartAt}`}
                        entry={entry}
                        nowCap={ongoingCap}
                      />
                    ))}
                  </ul>
                </div>
              )}
            </>
          )}
        </>
      )}
    </div>
  );
}

function FormSubmissionRow({ submission }: { submission: WorkerFormSubmission }) {
  const [open, setOpen] = useState(false);
  const hasFields = submission.fields.length > 0;

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <CollapsibleTrigger
        className="flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-muted/50 disabled:cursor-default disabled:hover:bg-transparent"
        disabled={!hasFields}
        aria-label={`Toggle fields for ${submission.templateName}`}
      >
        <div className="min-w-0 flex-1">
          <p className="truncate text-sm font-medium">{submission.templateName}</p>
          <p className="text-xs text-muted-foreground tabular-nums">
            {formatUnixDateTime(submission.submittedAt)}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          <Badge variant="secondary">
            {submission.fields.length} {pluralize("field", submission.fields.length)}
          </Badge>
          {hasFields ? (
            <ChevronDownIcon
              className={cn(
                "size-4 text-muted-foreground transition-transform duration-200",
                open && "rotate-180",
              )}
            />
          ) : null}
        </div>
      </CollapsibleTrigger>
      {hasFields ? (
        <CollapsibleContent>
          <dl className="grid grid-cols-1 gap-x-4 gap-y-2 border-t border-border bg-muted/20 px-4 py-3 sm:grid-cols-2">
            {submission.fields.map((field, index) => (
              <div key={`${field.label}-${index}`} className="min-w-0">
                <dt className="text-[11px] text-muted-foreground">{field.label}</dt>
                <dd className="text-sm break-words">{field.value || "—"}</dd>
              </div>
            ))}
          </dl>
        </CollapsibleContent>
      ) : null}
    </Collapsible>
  );
}

function FormsSection({ workerId }: { workerId: string }) {
  const [dateWindow] = useState(() => {
    const now = Math.floor(Date.now() / 1000);
    return { startTime: now - THIRTY_DAYS_SECONDS, endTime: now };
  });

  const formsQuery = useQuery({
    ...queries.telematics.workerFormSubmissions(workerId, dateWindow.startTime, dateWindow.endTime),
    enabled: workerId.length > 0,
    staleTime: STALE_TIME_MS,
  });

  const submissions = useMemo(
    () => [...(formsQuery.data ?? [])].sort((a, b) => b.submittedAt - a.submittedAt),
    [formsQuery.data],
  );

  return (
    <div className="flex flex-col gap-2">
      <div>
        <h3 className="text-sm font-semibold">Driver forms</h3>
        <p className="text-xs text-muted-foreground">
          Form submissions reported by this driver in the last 30 days.
        </p>
      </div>
      {formsQuery.isPending ? (
        <div className="flex flex-col gap-2">
          <Skeleton className="h-12 w-full rounded-lg" />
          <Skeleton className="h-12 w-full rounded-lg" />
        </div>
      ) : formsQuery.isError ? (
        <HosErrorState
          message="Form submissions could not be loaded"
          onRetry={() => void formsQuery.refetch()}
        />
      ) : submissions.length === 0 ? (
        <HosEmptyState
          icon={<ClipboardListIcon className="mx-auto size-5 text-muted-foreground" />}
          title="No form submissions in the last 30 days."
          description="Driver form submissions appear here once Samsara reports them for this worker."
        />
      ) : (
        <div className="overflow-hidden rounded-lg border border-border">
          <div className="divide-y divide-border">
            {submissions.map((submission) => (
              <FormSubmissionRow key={submission.id} submission={submission} />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function HosLiveState({
  state,
  workerId,
  since,
}: {
  state: WorkerHosState;
  workerId: string;
  since: number;
}) {
  const statusMeta = getDutyStatusMeta(state.dutyStatus);
  const cycleExtras: string[] = [`Tomorrow: ${formatDurationMs(state.cycleTomorrowMs)}`];
  if (state.cycleStartedAt) {
    cycleExtras.push(`Started ${formatUnixDate(state.cycleStartedAt)}`);
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center gap-2">
        <Badge variant={statusMeta.variant}>{statusMeta.label}</Badge>
        {state.currentVehicleId ? (
          <Badge variant="outline" className="gap-1">
            <TruckIcon className="size-3" />
            <span className="font-mono">{state.currentVehicleId}</span>
          </Badge>
        ) : null}
        <span className="text-xs text-muted-foreground">
          as of{" "}
          {formatDistanceToNowStrict(new Date(state.recordedAt * 1000), {
            addSuffix: true,
          })}
        </span>
      </div>

      <ActiveViolationAlert state={state} />

      <div className="grid grid-cols-2 gap-2 xl:grid-cols-4">
        <HosClockCard
          label="Until Break"
          limitLabel={limitLabel(state.breakLimitMs)}
          remainingMs={state.breakRemainingMs}
          limitMs={state.breakLimitMs || BREAK_LIMIT_MS}
          baseTone="warning"
        />
        <HosClockCard
          label="Drive"
          limitLabel={limitLabel(state.driveLimitMs)}
          remainingMs={state.driveRemainingMs}
          limitMs={state.driveLimitMs || DRIVE_LIMIT_MS}
        />
        <HosClockCard
          label="Shift"
          limitLabel={limitLabel(state.shiftLimitMs)}
          remainingMs={state.shiftRemainingMs}
          limitMs={state.shiftLimitMs || SHIFT_LIMIT_MS}
        />
        <HosClockCard
          label="Cycle"
          limitLabel={limitLabel(state.cycleLimitMs)}
          remainingMs={state.cycleRemainingMs}
          limitMs={state.cycleLimitMs || CYCLE_LIMIT_MS}
          extra={cycleExtras.join(" · ")}
        />
      </div>

      <DailyLogsSection workerId={workerId} />

      <FormsSection workerId={workerId} />

      <ViolationsSection workerId={workerId} since={since} />
    </div>
  );
}

export default function WorkerHosTab({ workerId }: { workerId: string }) {
  const [since] = useState(() => Math.floor(Date.now() / 1000) - THIRTY_DAYS_SECONDS);

  const statusQuery = useQuery({
    ...queries.telematics.status(),
  });
  const telematicsEnabled = statusQuery.data?.enabled ?? false;

  const hosQuery = useQuery({
    ...queries.telematics.workerHosState(workerId),
    enabled: telematicsEnabled && workerId.length > 0,
    refetchInterval: REFETCH_INTERVAL_MS,
  });

  if (statusQuery.isPending) {
    return <HosLoadingState />;
  }

  if (statusQuery.isError) {
    return (
      <HosErrorState
        message="Telematics status could not be loaded"
        onRetry={() => void statusQuery.refetch()}
      />
    );
  }

  if (!telematicsEnabled) {
    return (
      <HosEmptyState
        icon={<CableIcon className="mx-auto size-6 text-muted-foreground" />}
        title="Samsara telematics is not connected"
        description="Connect your Samsara account to stream live hours-of-service clocks, duty status, and violation history for this driver."
        action={
          <Button
            variant="outline"
            size="sm"
            className="mt-3"
            render={<Link to="/admin/integrations?type=Samsara" />}
          >
            Open Integrations
          </Button>
        }
      />
    );
  }

  if (hosQuery.isPending) {
    return <HosLoadingState />;
  }

  if (hosQuery.isError) {
    return (
      <HosErrorState
        message="Hours-of-service data could not be loaded"
        onRetry={() => void hosQuery.refetch()}
      />
    );
  }

  if (!hosQuery.data) {
    return (
      <HosEmptyState
        icon={<UserRoundXIcon className="mx-auto size-6 text-muted-foreground" />}
        title="Not linked to a Samsara driver"
        description="Hours-of-service data appears once this worker is matched to a Samsara driver. Run Worker Sync from the Samsara integration to link them."
        action={
          <Button
            variant="outline"
            size="sm"
            className="mt-3"
            render={<Link to="/admin/integrations?type=Samsara" />}
          >
            Open Samsara Integration
          </Button>
        }
      />
    );
  }

  return <HosLiveState state={hosQuery.data} workerId={workerId} since={since} />;
}
