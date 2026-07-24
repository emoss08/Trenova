import type { TelematicsStatus } from "@/lib/graphql/telematics";
import { queries } from "@/lib/queries";
import { formatPreciseTimeAgo } from "@/lib/time-utils";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useState } from "react";

const STATUS_REFETCH_MS = 30_000;
const CLOCK_TICK_MS = 10_000;
const HEALTHY_WINDOW_MS = 5 * 60 * 1000;
const DEGRADED_WINDOW_MS = 60 * 60 * 1000;

type HealthTone = "success" | "warning" | "destructive" | "muted";

const HEALTH_DOT_CLASS: Record<HealthTone, string> = {
  success: "bg-success",
  warning: "bg-warning",
  destructive: "bg-destructive",
  muted: "bg-muted-foreground",
};

const HEALTH_TEXT_CLASS: Record<HealthTone, string> = {
  success: "text-success",
  warning: "text-warning",
  destructive: "text-destructive",
  muted: "text-muted-foreground",
};

function computeHealth(status: TelematicsStatus, now: number): { label: string; tone: HealthTone } {
  if (!status.lastPolledAt) {
    return { label: "Not started", tone: "muted" };
  }
  const successAgeMs = status.lastSuccessAt ? now - status.lastSuccessAt * 1000 : Infinity;
  if (status.failureCount > 0) {
    if (successAgeMs <= DEGRADED_WINDOW_MS) {
      return { label: "Degraded", tone: "warning" };
    }
    return { label: "Failing", tone: "destructive" };
  }
  if (successAgeMs <= HEALTHY_WINDOW_MS) {
    return { label: "Healthy", tone: "success" };
  }
  return { label: "Stale", tone: "warning" };
}

function relativeOrNever(unixSeconds: number | null, now: number): string {
  return unixSeconds ? formatPreciseTimeAgo(unixSeconds * 1000, now) : "Never";
}

function HealthRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 px-3 py-2">
      <span className="shrink-0 text-xs text-muted-foreground">{label}</span>
      <div className="flex min-w-0 items-center justify-end gap-1.5 text-xs font-medium">
        {children}
      </div>
    </div>
  );
}

function HealthSkeleton() {
  return (
    <div className="divide-y divide-border rounded-md border border-border">
      {Array.from({ length: 6 }, (_, i) => (
        <div key={i} className="flex items-center justify-between gap-3 px-3 py-2">
          <Skeleton className="h-3 w-20" />
          <Skeleton className="h-3 w-16" />
        </div>
      ))}
    </div>
  );
}

export function SamsaraSyncHealthSection({ open }: { open: boolean }) {
  const statusQuery = useQuery({
    ...queries.telematics.status(),
    refetchInterval: STATUS_REFETCH_MS,
    staleTime: STATUS_REFETCH_MS,
    retry: false,
    refetchOnWindowFocus: false,
    enabled: open,
  });

  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (!open) return;
    const interval = window.setInterval(() => setNow(Date.now()), CLOCK_TICK_MS);
    return () => window.clearInterval(interval);
  }, [open]);

  const status = statusQuery.data;
  const health = status ? computeHealth(status, now) : null;

  let body: React.ReactNode;
  if (statusQuery.isLoading) {
    body = <HealthSkeleton />;
  } else if (statusQuery.isError || !status) {
    body = (
      <p className="rounded-md border border-border px-3 py-4 text-center text-xs text-muted-foreground">
        Sync status is unavailable right now.
      </p>
    );
  } else {
    const vehiclesShort = status.mappedTractors < status.totalTractors;
    body = (
      <div className="divide-y divide-border rounded-md border border-border">
        <HealthRow label="Provider">
          <span className="capitalize">{status.provider}</span>
        </HealthRow>
        <HealthRow label="Last poll">
          <span className="tabular-nums">{relativeOrNever(status.lastPolledAt, now)}</span>
        </HealthRow>
        <HealthRow label="Last success">
          <span className="tabular-nums">{relativeOrNever(status.lastSuccessAt, now)}</span>
        </HealthRow>
        <HealthRow label="Failure streak">
          {status.failureCount > 0 ? (
            <>
              {status.lastError && (
                <Tooltip>
                  <TooltipTrigger
                    render={
                      <span className="max-w-48 truncate text-[11px] font-normal text-muted-foreground" />
                    }
                  >
                    {status.lastError}
                  </TooltipTrigger>
                  <TooltipContent side="top" className="max-w-80 break-words">
                    {status.lastError}
                  </TooltipContent>
                </Tooltip>
              )}
              <Badge variant="inactive" className="h-4 shrink-0 rounded px-1 text-[9.5px]">
                {status.failureCount} failed
              </Badge>
            </>
          ) : (
            <span className="tabular-nums">0</span>
          )}
        </HealthRow>
        <HealthRow label="Vehicles mapped">
          <span className={cn("tabular-nums", vehiclesShort && "text-warning")}>
            {status.mappedTractors} of {status.totalTractors}
          </span>
        </HealthRow>
        <HealthRow label="Drivers linked">
          <span className="tabular-nums">{status.mappedWorkers}</span>
        </HealthRow>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 border-t border-border pt-4">
      <div className="flex items-center justify-between gap-2">
        <div className="flex flex-col gap-0.5">
          <p className="text-sm font-semibold">Sync Health</p>
          <p className="text-xs text-muted-foreground">
            Live polling status for the Samsara telematics feed.
          </p>
        </div>
        {health && (
          <span
            className={cn(
              "flex shrink-0 items-center gap-1.5 text-xs font-medium",
              HEALTH_TEXT_CLASS[health.tone],
            )}
          >
            <span
              aria-hidden
              className={cn(
                "size-1.5 rounded-full",
                HEALTH_DOT_CLASS[health.tone],
                health.tone === "success" && "animate-pulse",
              )}
            />
            {health.label}
          </span>
        )}
      </div>
      {body}
    </div>
  );
}
