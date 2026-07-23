import { MetricSkeleton } from "@/components/metric-skeleton";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { LucideIcon } from "lucide-react";
import { CalendarCheck2, CalendarClock, TrendingUp, Users } from "lucide-react";
import type { ApprovedPTOMetrics } from "./approved-pto-metrics";

export function ApprovedPTOKPICards({
  metrics,
  requestedCount,
  chartLoading,
  requestedLoading,
  requestedError,
}: {
  metrics: ApprovedPTOMetrics;
  requestedCount: number;
  chartLoading: boolean;
  requestedLoading: boolean;
  requestedError: boolean;
}) {
  if (chartLoading && requestedLoading) {
    return <MetricSkeleton />;
  }

  return (
    <div className="mb-3 grid grid-cols-1 gap-2.5 sm:grid-cols-2 xl:grid-cols-4">
      <MetricCard
        label="Approved PTO Days"
        value={metrics.approvedPtoDays.toLocaleString()}
        detail="Daily occupancy total"
        icon={CalendarCheck2}
      />
      <MetricCard
        label="Requested PTO Requests"
        value={requestedError ? "--" : requestedCount.toLocaleString()}
        detail="Pending approvals in range"
        icon={CalendarClock}
      />
      <MetricCard
        label="Workers With Approved PTO"
        value={metrics.workersWithApprovedPTO.toLocaleString()}
        detail="Unique workers in range"
        icon={Users}
      />
      <MetricCard
        label="Peak Day Occupancy"
        value={metrics.peakDay.occupancy.toLocaleString()}
        detail={metrics.peakDay.dateLabel ?? "No peak day"}
        icon={TrendingUp}
      />
      {requestedError && !requestedLoading && (
        <p className="col-span-full rounded-md border border-dashed border-border px-2.5 py-2 text-xs text-muted-foreground">
          Requested PTO metric is temporarily unavailable.
        </p>
      )}
    </div>
  );
}

function MetricCard({
  label,
  value,
  detail,
  icon: Icon,
}: {
  label: string;
  value: string;
  detail?: string;
  icon: LucideIcon;
}) {
  return (
    <Card className="group relative gap-0 overflow-hidden border-border/80 transition-colors hover:border-border">
      <CardHeader className="relative flex flex-row items-start justify-between space-y-0 pb-2">
        <CardTitle className="text-[11px] font-semibold tracking-wide text-muted-foreground uppercase">
          {label}
        </CardTitle>
        <span className="inline-flex size-7 shrink-0 items-center justify-center rounded-md bg-accent">
          <Icon className="size-4" />
        </span>
      </CardHeader>
      <CardContent className="relative space-y-1 pt-0">
        <p className="text-3xl leading-none font-semibold tracking-tight">{value}</p>
        {detail ? <p className="text-[11px] text-muted-foreground">{detail}</p> : null}
      </CardContent>
    </Card>
  );
}
