import { MetricSkeleton } from "@/components/metric-skeleton";
import { Card, CardContent, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
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
        <p className="border-border text-muted-foreground col-span-full rounded-md border border-dashed px-2.5 py-2 text-xs">
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
    <Card className="group border-border/80 hover:border-border relative gap-0 overflow-hidden transition-colors">
      <CardHeader className="relative flex flex-row items-start justify-between space-y-0 pb-2">
        <CardTitle className="text-muted-foreground text-[11px] font-semibold tracking-wide uppercase">
          {label}
        </CardTitle>
        <span className="bg-accent inline-flex size-7 shrink-0 items-center justify-center rounded-md">
          <Icon className="size-4" />
        </span>
      </CardHeader>
      <CardContent className="relative space-y-1 pt-0">
        <p className="text-3xl leading-none font-semibold tracking-tight">{value}</p>
        {detail ? <p className="text-muted-foreground text-[11px]">{detail}</p> : null}
      </CardContent>
    </Card>
  );
}
