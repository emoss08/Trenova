import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Badge } from "@trenova/shared/components/ui/badge";
import type { BadgeVariant } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { RingGauge } from "@trenova/shared/components/ui/ring-gauge";
import type { RingGaugeTone } from "@trenova/shared/components/ui/ring-gauge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatClockDurationMs, formatDurationMs, formatUnixDate } from "@trenova/shared/lib/date";
import {
  fetchMyHosDailyLogs,
  fetchMyHosState,
  fetchMyHosViolations,
  type MyHosDailyLog,
  type MyHosState,
  type MyHosViolation,
} from "@trenova/shared/lib/graphql/driver-portal";
import { metersToMiles, toTitleCase } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { CircleCheckIcon, Clock4Icon, GaugeIcon, TriangleAlertIcon, TruckIcon } from "lucide-react";
import { m } from "motion/react";
import { useState } from "react";

const HOUR_MS = 60 * 60 * 1000;
const DAY_SECONDS = 24 * 60 * 60;

function toDateKey(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function timeAgo(unixSeconds: number): string {
  const seconds = Math.max(0, Math.floor(Date.now() / 1000) - unixSeconds);
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

function gaugeTone(remainingMs: number, defaultTone: RingGaugeTone): RingGaugeTone {
  if (remainingMs < HOUR_MS) return "critical";
  if (remainingMs < 2 * HOUR_MS) return "warning";
  return defaultTone;
}

const DUTY_STATUS: Record<string, { label: string; variant: BadgeVariant }> = {
  driving: { label: "Driving", variant: "info" },
  onDuty: { label: "On Duty", variant: "warning" },
  offDuty: { label: "Off Duty", variant: "secondary" },
  sleeperBed: { label: "Sleeper Berth", variant: "purple" },
  yardMove: { label: "Yard Move", variant: "teal" },
  personalConveyance: { label: "Personal Conveyance", variant: "teal" },
};

type ClockGaugeProps = {
  label: string;
  remainingMs: number;
  limitMs: number;
  defaultTone: RingGaugeTone;
};

function ClockGauge({ label, remainingMs, limitMs, defaultTone }: ClockGaugeProps) {
  const value = limitMs > 0 ? remainingMs / limitMs : 0;
  const tone = gaugeTone(remainingMs, defaultTone);
  return (
    <div className="flex flex-col items-center gap-2">
      <RingGauge
        value={value}
        size={132}
        strokeWidth={9}
        tone={tone}
        aria-label={`${label} remaining`}
      >
        <div className="flex flex-col items-center leading-none">
          <span className="text-xl font-semibold tabular-nums">
            {formatClockDurationMs(remainingMs)}
          </span>
          <span className="mt-1 text-2xs font-medium tracking-wide text-muted-foreground uppercase">
            left
          </span>
        </div>
      </RingGauge>
      <span className="text-xs font-medium text-muted-foreground">{label}</span>
    </div>
  );
}

function ClockHero({ state }: { state: MyHosState }) {
  const duty = state.dutyStatus ? DUTY_STATUS[state.dutyStatus] : null;
  const dutyLabel = duty?.label ?? (state.dutyStatus ? toTitleCase(state.dutyStatus) : "Unknown");

  return (
    <m.section
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.22, ease: "easeOut", delay: 0.04 }}
      className="rounded-2xl border border-border bg-card p-5"
    >
      <div className="grid grid-cols-2 gap-x-4 gap-y-6">
        <ClockGauge
          label="Until break"
          remainingMs={state.breakRemainingMs}
          limitMs={state.breakLimitMs}
          defaultTone="warning"
        />
        <ClockGauge
          label="Drive"
          remainingMs={state.driveRemainingMs}
          limitMs={state.driveLimitMs}
          defaultTone="brand"
        />
        <ClockGauge
          label="Shift"
          remainingMs={state.shiftRemainingMs}
          limitMs={state.shiftLimitMs}
          defaultTone="brand"
        />
        <ClockGauge
          label="Cycle"
          remainingMs={state.cycleRemainingMs}
          limitMs={state.cycleLimitMs}
          defaultTone="brand"
        />
      </div>

      <div className="mt-5 flex flex-wrap items-center gap-2 border-t border-border pt-4">
        <Badge variant={duty?.variant ?? "secondary"}>{dutyLabel}</Badge>
        {state.currentVehicleId ? (
          <span className="inline-flex max-w-40 items-center gap-1 rounded-full border border-border bg-muted/50 px-2.5 py-1 text-xs text-muted-foreground">
            <TruckIcon className="size-3.5 shrink-0" />
            <span className="truncate font-mono">{state.currentVehicleId}</span>
          </span>
        ) : null}
        {state.rulesetCycle ? (
          <span className="text-xs text-muted-foreground">{state.rulesetCycle}</span>
        ) : null}
        <span className="ml-auto text-xs text-muted-foreground">
          Updated {timeAgo(state.recordedAt)}
        </span>
      </div>
    </m.section>
  );
}

function DailyLogRow({ log }: { log: MyHosDailyLog }) {
  const miles = Math.round(metersToMiles(log.driveDistanceMeters));
  return (
    <li className="flex items-center justify-between gap-3 px-4 py-3">
      <div className="min-w-0">
        <p className="text-sm font-medium">{formatUnixDate(log.startAt)}</p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {formatDurationMs(log.driveDurationMs)} drive · {formatDurationMs(log.onDutyDurationMs)}{" "}
          on-duty · {miles} mi
        </p>
        {!log.isCertified ? (
          <p className="mt-0.5 text-xs text-amber-600 dark:text-amber-400">Not yet certified</p>
        ) : null}
      </div>
      {log.isCertified ? (
        <Badge variant="active">Certified</Badge>
      ) : (
        <Badge variant="warning">Uncertified</Badge>
      )}
    </li>
  );
}

function RecentLogsSection({ enabled }: { enabled: boolean }) {
  const [range] = useState(() => {
    const end = new Date();
    const start = new Date();
    start.setDate(start.getDate() - 6);
    return { start: toDateKey(start), end: toDateKey(end) };
  });

  const logs = useQuery({
    queryKey: ["dash-hos-logs", range.start, range.end],
    queryFn: () => fetchMyHosDailyLogs(range.start, range.end),
    staleTime: 60_000,
    enabled,
  });

  return (
    <m.section
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.22, ease: "easeOut", delay: 0.1 }}
      className="flex flex-col gap-3"
    >
      <h2 className="text-sm font-semibold">Last 7 days</h2>
      {logs.isPending ? (
        <Skeleton className="h-40 w-full rounded-2xl" />
      ) : logs.isError ? (
        <div className="flex flex-col items-center gap-2 rounded-2xl border border-dashed border-border p-6 text-center">
          <p className="text-sm text-muted-foreground">We couldn&apos;t load your daily logs.</p>
          <Button variant="outline" size="sm" className="h-8" onClick={() => logs.refetch()}>
            Try again
          </Button>
        </div>
      ) : logs.data && logs.data.length > 0 ? (
        <ul className="divide-y divide-border rounded-2xl border border-border bg-card">
          {logs.data.map((log) => (
            <DailyLogRow key={log.startAt} log={log} />
          ))}
        </ul>
      ) : (
        <div className="rounded-2xl border border-dashed border-border p-6 text-center text-sm text-muted-foreground">
          No daily logs in the last week.
        </div>
      )}
    </m.section>
  );
}

function ViolationRow({ violation }: { violation: MyHosViolation }) {
  return (
    <li className="flex items-center justify-between gap-3 px-4 py-3">
      <div className="min-w-0">
        <p className="text-sm font-medium">{toTitleCase(violation.violationType)}</p>
        <p className="mt-0.5 text-xs text-muted-foreground">
          {formatUnixDate(violation.violationStartAt)}
          {violation.durationMs > 0 ? ` · ${formatDurationMs(violation.durationMs)}` : ""}
        </p>
        {violation.description ? (
          <p className="mt-0.5 line-clamp-2 text-xs text-muted-foreground">
            {violation.description}
          </p>
        ) : null}
      </div>
      <TriangleAlertIcon className="size-4 shrink-0 text-amber-600 dark:text-amber-400" />
    </li>
  );
}

function ViolationsSection({ enabled }: { enabled: boolean }) {
  const [since] = useState(() => Math.floor(Date.now() / 1000) - 30 * DAY_SECONDS);

  const violations = useQuery({
    queryKey: ["dash-hos-violations", since],
    queryFn: () => fetchMyHosViolations(since),
    staleTime: 60_000,
    enabled,
  });

  return (
    <m.section
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.22, ease: "easeOut", delay: 0.14 }}
      className="flex flex-col gap-3"
    >
      <h2 className="text-sm font-semibold">Violations (last 30 days)</h2>
      {violations.isPending ? (
        <Skeleton className="h-24 w-full rounded-2xl" />
      ) : violations.isError ? (
        <div className="flex flex-col items-center gap-2 rounded-2xl border border-dashed border-border p-6 text-center">
          <p className="text-sm text-muted-foreground">We couldn&apos;t load your violations.</p>
          <Button variant="outline" size="sm" className="h-8" onClick={() => violations.refetch()}>
            Try again
          </Button>
        </div>
      ) : violations.data && violations.data.length > 0 ? (
        <ul className="divide-y divide-border rounded-2xl border border-border bg-card">
          {violations.data.map((violation, index) => (
            <ViolationRow
              key={`${violation.violationType}-${violation.violationStartAt}-${index}`}
              violation={violation}
            />
          ))}
        </ul>
      ) : (
        <div className="flex flex-col items-center gap-2 rounded-2xl border border-dashed border-border p-8 text-center">
          <CircleCheckIcon className="size-6 text-green-600 dark:text-green-400" />
          <p className="text-sm text-muted-foreground">No violations — nice work.</p>
        </div>
      )}
    </m.section>
  );
}

export function DashHosPage() {
  const state = useQuery({
    queryKey: ["dash-hos-state"],
    queryFn: fetchMyHosState,
    refetchInterval: 60_000,
    staleTime: 30_000,
  });

  return (
    <div className="flex flex-col gap-6">
      <m.div
        initial={{ opacity: 0, y: 10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.22, ease: "easeOut" }}
      >
        <h1 className="text-xl font-semibold tracking-tight">Hours of service</h1>
        <p className="text-sm text-muted-foreground">Your clocks, logs, and compliance.</p>
      </m.div>

      {state.isPending ? (
        <div className="flex flex-col gap-6">
          <Skeleton className="h-72 w-full rounded-2xl" />
          <Skeleton className="h-40 w-full rounded-2xl" />
          <Skeleton className="h-24 w-full rounded-2xl" />
        </div>
      ) : state.isError ? (
        <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-border p-8 text-center">
          <GaugeIcon className="size-7 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">
            We couldn&apos;t load your hours of service.
          </p>
          <Button variant="outline" size="sm" onClick={() => state.refetch()}>
            Try again
          </Button>
        </div>
      ) : !state.data ? (
        <div className="flex flex-col items-center gap-3 rounded-2xl border border-dashed border-border p-8 text-center">
          <Clock4Icon className="size-7 text-muted-foreground" />
          <p className="text-sm font-medium">Hours of service isn&apos;t available</p>
          <p className="max-w-xs text-sm text-muted-foreground">
            Your carrier hasn&apos;t connected an ELD provider yet, or your driver profile
            isn&apos;t linked to one.
          </p>
        </div>
      ) : (
        <>
          <ClockHero state={state.data} />

          {state.data.shiftDrivingViolationMs > 0 || state.data.cycleViolationMs > 0 ? (
            <Alert variant="destructive">
              <TriangleAlertIcon />
              <AlertTitle>You have an active HOS violation</AlertTitle>
              <AlertDescription>Contact dispatch before you keep driving.</AlertDescription>
            </Alert>
          ) : null}

          <RecentLogsSection enabled />
          <ViolationsSection enabled />
        </>
      )}
    </div>
  );
}
