import {
  HOS_LIMITS_MS,
  HosClockGauges,
  hosClockSeverity,
  type HosClockSeverity,
} from "@/components/hos/hos-clock-gauges";
import type { WorkerHosState } from "@/lib/graphql/telematics";
import { queries } from "@/lib/queries";
import { formatElapsedTime } from "@/lib/time-utils";
import { useRealtimeStore } from "@/stores/realtime-store";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { formatDurationFromSeconds } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import type { BadgeVariant } from "@trenova/shared/types/badge";
import { useQuery } from "@tanstack/react-query";
import { AlertTriangleIcon, ExternalLinkIcon, PlugZapIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router";
import { ModuleCard } from "./module-card";

const HOS_FETCH_LIMIT = 12;
const HOS_DISPLAY_LIMIT = 8;
const CLOCK_TICK_MS = 15_000;

const DUTY_STATUS: Record<string, { label: string; variant: BadgeVariant }> = {
  driving: { label: "Driving", variant: "info" },
  onDuty: { label: "On duty", variant: "warning" },
  offDuty: { label: "Off duty", variant: "outline" },
  sleeperBed: { label: "Sleeper", variant: "purple" },
  yardMove: { label: "Yard move", variant: "teal" },
  personalConveyance: { label: "Personal", variant: "teal" },
};

const UNKNOWN_DUTY = { label: "Unknown", variant: "outline" as BadgeVariant };

function hasViolation(state: WorkerHosState): boolean {
  return state.shiftDrivingViolationMs > 0 || state.cycleViolationMs > 0;
}

function rowSeverity(state: WorkerHosState): HosClockSeverity {
  if (hasViolation(state)) return "critical";
  return hosClockSeverity(state.driveRemainingMs);
}

function alertLabel(state: WorkerHosState): string {
  const parts: string[] = [];
  if (state.shiftDrivingViolationMs > 0) {
    parts.push(`Shift driving violation · ${formatRemaining(state.shiftDrivingViolationMs)} over`);
  }
  if (state.cycleViolationMs > 0) {
    parts.push(`Cycle violation · ${formatRemaining(state.cycleViolationMs)} over`);
  }
  if (parts.length === 0) {
    return "Less than 1h of drive time remaining";
  }
  return parts.join(" · ");
}

function formatRemaining(ms: number): string {
  return formatDurationFromSeconds(Math.max(0, Math.floor(ms / 1000)));
}

const SEVERITY_TEXT: Record<HosClockSeverity, string> = {
  normal: "text-foreground",
  warning: "text-warning",
  critical: "text-destructive",
};

function HosRow({ state, withDivider }: { state: WorkerHosState; withDivider: boolean }) {
  const severity = rowSeverity(state);
  const duty = (state.dutyStatus && DUTY_STATUS[state.dutyStatus]) || UNKNOWN_DUTY;

  return (
    <div
      className={cn("flex flex-col gap-1 px-0.5 py-1.5", withDivider && "border-border border-t")}
    >
      <div className="flex items-center justify-between gap-2">
        <div className="flex min-w-0 items-center gap-1">
          {severity === "critical" && (
            <Tooltip>
              <TooltipTrigger
                render={<span className="text-destructive flex shrink-0 items-center" />}
              >
                <AlertTriangleIcon className="size-3" />
              </TooltipTrigger>
              <TooltipContent side="left">{alertLabel(state)}</TooltipContent>
            </Tooltip>
          )}
          <Link
            to={`/dispatch/workers?panelType=edit&panelEntityId=${state.workerId}&tab=hos`}
            className={cn(
              "truncate text-[11px] font-semibold hover:underline",
              severity !== "normal" && SEVERITY_TEXT[severity],
            )}
          >
            {state.workerName}
          </Link>
        </div>
        <Badge variant={duty.variant} className="h-4 shrink-0 rounded px-1 text-[8.5px]">
          {duty.label}
        </Badge>
      </div>
      <HosClockGauges
        driveRemainingMs={state.driveRemainingMs}
        shiftRemainingMs={state.shiftRemainingMs}
        cycleRemainingMs={state.cycleRemainingMs}
        driveLimitMs={state.driveLimitMs || HOS_LIMITS_MS.drive}
        shiftLimitMs={state.shiftLimitMs || HOS_LIMITS_MS.shift}
        cycleLimitMs={state.cycleLimitMs || HOS_LIMITS_MS.cycle}
        hasViolation={hasViolation(state)}
        size={26}
      />
    </div>
  );
}

function HosSkeletonRow({ withDivider }: { withDivider: boolean }) {
  return (
    <div
      className={cn("flex flex-col gap-1.5 px-0.5 py-2", withDivider && "border-border border-t")}
    >
      <div className="flex items-center justify-between gap-2">
        <Skeleton className="h-3 w-24" />
        <Skeleton className="h-3.5 w-12" />
      </div>
      <div className="flex items-center gap-2">
        <Skeleton className="size-6.5 rounded-full" />
        <Skeleton className="h-3 flex-1" />
        <Skeleton className="size-6.5 rounded-full" />
        <Skeleton className="h-3 flex-1" />
        <Skeleton className="size-6.5 rounded-full" />
        <Skeleton className="h-3 flex-1" />
      </div>
    </div>
  );
}

function ConnectSamsaraState() {
  return (
    <div className="cc-fade-in flex flex-col items-center gap-2 px-4 py-6 text-center">
      <span className="bg-muted text-muted-foreground inline-flex size-8 items-center justify-center rounded-full">
        <PlugZapIcon className="size-4" />
      </span>
      <p className="text-[11.5px] font-medium">Connect Samsara to watch driver clocks</p>
      <p className="text-muted-foreground max-w-55 text-[10.5px] leading-snug">
        Live hours-of-service visibility turns on once the Samsara telematics integration is enabled
        for your organization.
      </p>
      <Button
        variant="outline"
        size="xs"
        nativeButton={false}
        render={<Link to="/admin/integrations?type=Samsara" />}
      >
        <ExternalLinkIcon className="size-3" />
        Open Integrations
      </Button>
    </div>
  );
}

function ErrorState({ onRetry }: { onRetry: () => void }) {
  return (
    <div className="cc-fade-in flex flex-col items-center gap-2 px-4 py-5 text-center">
      <p className="text-muted-foreground text-[10.5px]">
        HOS data could not be loaded from Samsara.
      </p>
      <Button variant="outline" size="xs" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

export function HosWatch({ enabled = true }: { enabled?: boolean }) {
  const statusQuery = useQuery({
    ...queries.telematics.status(),
    staleTime: 5 * 60 * 1000,
    retry: false,
    refetchOnWindowFocus: false,
    enabled,
  });
  const telematicsEnabled = statusQuery.data?.enabled ?? false;

  const hosQuery = useQuery({
    ...queries.telematics.workerHosStates(HOS_FETCH_LIMIT),
    refetchInterval: 60_000,
    staleTime: 30_000,
    retry: false,
    refetchOnWindowFocus: false,
    enabled: enabled && telematicsEnabled,
  });

  const connectionState = useRealtimeStore.use.connectionState();
  const [now, setNow] = useState(() => Date.now());
  const tickEnabled = enabled && telematicsEnabled;

  useEffect(() => {
    if (!tickEnabled) return;
    const interval = window.setInterval(() => setNow(Date.now()), CLOCK_TICK_MS);
    return () => window.clearInterval(interval);
  }, [tickEnabled]);

  const states = useMemo(() => hosQuery.data ?? [], [hosQuery.data]);
  const rows = states.slice(0, HOS_DISPLAY_LIMIT);
  const urgentCount = states.filter((s) => rowSeverity(s) !== "normal").length;
  const criticalCount = states.filter((s) => rowSeverity(s) === "critical").length;
  const freshestRecordedAt = states.reduce((max, s) => Math.max(max, s.recordedAt), 0);

  const isLoading = statusQuery.isLoading || (telematicsEnabled && hosQuery.isLoading);
  const live = connectionState === "connected";

  let body: React.ReactNode;
  if (isLoading) {
    body = (
      <>
        <HosSkeletonRow withDivider={false} />
        <HosSkeletonRow withDivider />
        <HosSkeletonRow withDivider />
      </>
    );
  } else if (statusQuery.isError) {
    body = <ErrorState onRetry={() => void statusQuery.refetch()} />;
  } else if (!telematicsEnabled) {
    body = <ConnectSamsaraState />;
  } else if (hosQuery.isError) {
    body = <ErrorState onRetry={() => void hosQuery.refetch()} />;
  } else if (rows.length === 0) {
    body = (
      <p className="text-muted-foreground px-2 py-4 text-center text-[10.5px]">
        No HOS data yet — drivers appear once Samsara reports clocks.
      </p>
    );
  } else {
    body = (
      <>
        {rows.map((state, i) => (
          <HosRow key={state.workerId} state={state} withDivider={i > 0} />
        ))}
        <div className="border-border flex items-center justify-between gap-2 border-t px-0.5 py-1.5">
          <span className="font-table text-muted-foreground flex items-center gap-1 text-[9px] tabular-nums">
            <span
              aria-hidden
              className={cn(
                "size-1 rounded-full",
                live ? "bg-success animate-pulse" : "bg-muted-foreground",
              )}
            />
            {live ? "Live" : "Offline"}
          </span>
          {freshestRecordedAt > 0 && (
            <span className="font-table text-muted-foreground text-[9px] tabular-nums">
              Updated {formatElapsedTime(freshestRecordedAt * 1000, now)}
            </span>
          )}
        </div>
      </>
    );
  }

  return (
    <ModuleCard
      id="hos"
      title="HOS watch"
      count={telematicsEnabled && hosQuery.data ? urgentCount : undefined}
      countTone={criticalCount > 0 ? "danger" : "warning"}
    >
      {body}
    </ModuleCard>
  );
}
