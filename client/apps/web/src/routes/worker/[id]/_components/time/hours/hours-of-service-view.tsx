import { SectionPanel } from "@/components/section-panel";
import { telematics } from "@/lib/queries/telematics";
import { workerOverviewQuery } from "@/lib/queries/worker";
import { useQuery } from "@tanstack/react-query";
import { ErrorState } from "@trenova/shared/components/errors/error-state";
import { useNow } from "@trenova/shared/hooks/use-now";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { TimeSkeleton } from "../time-skeleton";
import { DailyLogPanel } from "./daily-log-panel";
import { HosBand } from "./hos-band";
import { HosLinkState } from "./hos-link-state";
import {
  buildDaySlots,
  buildLogSegments,
  HOS_LIVE_REFRESH_MS,
  hosWindow,
  laneTotalsMs,
} from "./hos-model";
import { DriverFormsPanel, ViolationsPanel } from "./hos-records-panels";

/**
 * Hours of service: whether there is anything to show comes from the overview the page already
 * holds (connected, linked, reporting); the clocks, the logs and the records are their own reads.
 */
export default function HoursOfServiceView({ workerId }: { workerId: string }) {
  const t = useT();
  const overview = useQuery(workerOverviewQuery(workerId));

  if (overview.isPending) return <TimeSkeleton view="hours-of-service" />;
  if (overview.isError) {
    return (
      <ErrorState
        layout="sheet"
        error={overview.error}
        onRetry={() => void overview.refetch()}
        title={t("The hours of service could not be loaded")}
      />
    );
  }
  const hos = overview.data.hos;
  if (hos === null) return <HosLinkState link={null} />;
  if (hos.link !== "Reporting") return <HosLinkState link={hos.link} />;
  return <HoursReporting workerId={workerId} />;
}

function HoursReporting({ workerId }: { workerId: string }) {
  const t = useT();
  // The view's reads cover whole days, and today's open segment runs to now: both follow one
  // minute clock, so the window turns over at midnight and the band's figures move each minute.
  const nowSec = useNow("minute");
  const span = hosWindow(nowSec);
  const [selectedKey, setSelectedKey] = useState<string | null>(null);

  const clocksQuery = useQuery({
    ...telematics.workerHosClocks(workerId),
    refetchInterval: HOS_LIVE_REFRESH_MS,
  });
  const dailyLogsQuery = useQuery({
    ...telematics.workerHosDailyLogs(workerId, span.startDate, span.endDate),
    staleTime: HOS_LIVE_REFRESH_MS,
  });
  const slots = buildDaySlots(dailyLogsQuery.data ?? [], span.today);
  const today = slots[slots.length - 1];
  const selected = slots.find((slot) => slot.key === selectedKey) ?? today;
  // Today's log is the one live log read: the daily log panel reads the same key when today is
  // selected, so one request a minute serves both.
  const todayLogsQuery = useQuery({
    ...telematics.workerHosLogs(workerId, today.startAt, today.endAt),
    enabled: dailyLogsQuery.isSuccess,
    staleTime: HOS_LIVE_REFRESH_MS,
    refetchInterval: HOS_LIVE_REFRESH_MS,
  });
  const todaySegments = buildLogSegments(
    todayLogsQuery.data ?? [],
    today.startAt,
    today.endAt,
    nowSec,
  );
  const openSegment = [...todaySegments].reverse().find((segment) => segment.ongoing) ?? null;

  if (clocksQuery.isPending) return <TimeSkeleton view="hours-of-service" />;
  if (clocksQuery.isError) {
    return (
      <ErrorState
        layout="sheet"
        error={clocksQuery.error}
        onRetry={() => void clocksQuery.refetch()}
        title={t("The hours-of-service clocks could not be loaded")}
      />
    );
  }
  if (clocksQuery.data === null) return <HosLinkState link="NoData" />;

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <HosBand
        clocks={clocksQuery.data}
        vehicleName={openSegment?.vehicleName ?? null}
        drivenTodayMs={todayLogsQuery.isSuccess ? laneTotalsMs(todaySegments)[2] : null}
      />
      {dailyLogsQuery.isPending ? (
        <div
          aria-busy
          aria-label={t("Loading the daily log")}
          className="h-72 w-full ui-shimmer rounded-lg border"
        />
      ) : dailyLogsQuery.isError ? (
        <SectionPanel title={t("Daily log")}>
          <div className="p-3">
            <ErrorState
              layout="compact"
              error={dailyLogsQuery.error}
              onRetry={() => void dailyLogsQuery.refetch()}
              title={t("The daily logs could not be loaded")}
            />
          </div>
        </SectionPanel>
      ) : (
        <DailyLogPanel
          workerId={workerId}
          slots={slots}
          selected={selected}
          nowSec={nowSec}
          onSelect={setSelectedKey}
        />
      )}
      <div className="grid items-start gap-4 lg:grid-cols-2">
        <ViolationsPanel workerId={workerId} since={span.since} />
        <DriverFormsPanel workerId={workerId} startTime={span.since} endTime={span.until} />
      </div>
    </div>
  );
}
