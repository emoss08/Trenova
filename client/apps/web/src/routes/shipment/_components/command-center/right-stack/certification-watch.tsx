import type { HosCertificationSummary } from "@/lib/graphql/telematics";
import { queries } from "@/lib/queries";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useQuery } from "@tanstack/react-query";
import { format, subDays } from "date-fns";
import { AlertTriangleIcon, ExternalLinkIcon, PlugZapIcon, ShieldCheckIcon } from "lucide-react";
import { useState } from "react";
import { Link } from "react-router";
import { ModuleCard } from "./module-card";

const DAY_KEY_FORMAT = "yyyy-MM-dd";
const CERTIFICATION_WINDOW_DAYS = 8;
const REFETCH_INTERVAL_MS = 5 * 60 * 1000;
const STATUS_STALE_TIME_MS = 5 * 60 * 1000;
const SUMMARY_STALE_TIME_MS = 60_000;

function CertificationRow({
  summary,
  withDivider,
}: {
  summary: HosCertificationSummary;
  withDivider: boolean;
}) {
  return (
    <div
      className={
        withDivider
          ? "flex items-center justify-between gap-2 border-t border-border px-0.5 py-1.5"
          : "flex items-center justify-between gap-2 px-0.5 py-1.5"
      }
    >
      <div className="flex min-w-0 items-center gap-1">
        <AlertTriangleIcon className="size-3 shrink-0 text-warning" />
        <Link
          to={`/dispatch/workers?panelType=edit&panelEntityId=${summary.workerId}&tab=hos`}
          className="truncate text-[11px] font-semibold hover:underline"
        >
          {summary.workerName}
        </Link>
      </div>
      <span className="shrink-0 font-table text-[9.5px] text-muted-foreground tabular-nums">
        {summary.uncertifiedDays} of {summary.totalDays} days
      </span>
    </div>
  );
}

function CertificationSkeletonRow({ withDivider }: { withDivider: boolean }) {
  return (
    <div
      className={
        withDivider
          ? "flex items-center justify-between gap-2 border-t border-border px-0.5 py-2"
          : "flex items-center justify-between gap-2 px-0.5 py-2"
      }
    >
      <Skeleton className="h-3 w-28" />
      <Skeleton className="h-3 w-16" />
    </div>
  );
}

function ConnectSamsaraState() {
  return (
    <div className="cc-fade-in flex flex-col items-center gap-2 px-4 py-6 text-center">
      <span className="inline-flex size-8 items-center justify-center rounded-full bg-muted text-muted-foreground">
        <PlugZapIcon className="size-4" />
      </span>
      <p className="text-[11.5px] font-medium">Connect Samsara to track log certification</p>
      <p className="max-w-55 text-[10.5px] leading-snug text-muted-foreground">
        Uncertified ELD log visibility turns on once the Samsara telematics integration is enabled
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
      <p className="text-[10.5px] text-muted-foreground">
        Certification data could not be loaded from Samsara.
      </p>
      <Button variant="outline" size="xs" onClick={onRetry}>
        Try again
      </Button>
    </div>
  );
}

export function CertificationWatch({ enabled = true }: { enabled?: boolean }) {
  const [dateRange] = useState(() => {
    const today = new Date();
    return {
      startDate: format(subDays(today, CERTIFICATION_WINDOW_DAYS - 1), DAY_KEY_FORMAT),
      endDate: format(today, DAY_KEY_FORMAT),
    };
  });

  const statusQuery = useQuery({
    ...queries.telematics.status(),
    staleTime: STATUS_STALE_TIME_MS,
    retry: false,
    refetchOnWindowFocus: false,
    enabled,
  });
  const telematicsEnabled = statusQuery.data?.enabled ?? false;

  const summaryQuery = useQuery({
    ...queries.telematics.hosCertificationSummary(dateRange.startDate, dateRange.endDate),
    refetchInterval: REFETCH_INTERVAL_MS,
    staleTime: SUMMARY_STALE_TIME_MS,
    retry: false,
    refetchOnWindowFocus: false,
    enabled: enabled && telematicsEnabled,
  });

  const summaries = summaryQuery.data ?? [];
  const isLoading = statusQuery.isLoading || (telematicsEnabled && summaryQuery.isLoading);

  let body: React.ReactNode;
  if (isLoading) {
    body = (
      <>
        <CertificationSkeletonRow withDivider={false} />
        <CertificationSkeletonRow withDivider />
        <CertificationSkeletonRow withDivider />
      </>
    );
  } else if (statusQuery.isError) {
    body = <ErrorState onRetry={() => void statusQuery.refetch()} />;
  } else if (!telematicsEnabled) {
    body = <ConnectSamsaraState />;
  } else if (summaryQuery.isError) {
    body = <ErrorState onRetry={() => void summaryQuery.refetch()} />;
  } else if (summaries.length === 0) {
    body = (
      <div className="cc-fade-in flex flex-col items-center gap-2 px-4 py-6 text-center">
        <span className="inline-flex size-8 items-center justify-center rounded-full bg-success/15 text-success">
          <ShieldCheckIcon className="size-4" />
        </span>
        <p className="text-[11.5px] font-medium">All drivers certified — no outstanding logs.</p>
        <p className="max-w-55 text-[10.5px] leading-snug text-muted-foreground">
          Every driver has certified their ELD logs for the last {CERTIFICATION_WINDOW_DAYS} days.
        </p>
      </div>
    );
  } else {
    body = (
      <>
        {summaries.map((summary, i) => (
          <CertificationRow key={summary.workerId} summary={summary} withDivider={i > 0} />
        ))}
      </>
    );
  }

  return (
    <ModuleCard
      id="certification"
      title="Uncertified logs"
      count={telematicsEnabled && summaryQuery.data ? summaries.length : undefined}
      countTone="warning"
    >
      {body}
    </ModuleCard>
  );
}
